package order

import (
	"context"
	"log/slog"

	"github.com/darkphotonKN/seeyoulatte-app/internal/constants"
	"github.com/darkphotonKN/seeyoulatte-app/internal/ledger"
	dbutils "github.com/darkphotonKN/seeyoulatte-app/internal/utils/db"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// transition rules
type Transition struct {
	From   constants.OrderState                                      // state starting
	To     constants.OrderState                                      // state attempting to transition to
	Actor  constants.Actor                                           // who is allowed to call this state
	Guard  func(order *Order, transitionCtx TransitionContext) error // guard for transition
	Action func(order *Order, transitionCtx TransitionContext) error // handles updates for transiion and side affects
}

// transition dependencies
type TransitionContext struct {
	ctx           context.Context
	db            *sqlx.DB
	ledgerService LedgerService
	orderService  StateTransitionOrderService
}

type LedgerService interface {
	Create(ctx context.Context, entry *ledger.LedgerEntry) error
}

type StateTransitionOrderService interface {
	UpdateStateTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, userID uuid.UUID, state constants.OrderState) (*Order, error)
}

var transitions []Transition = []Transition{
	Transition{
		From:  constants.StatePendingPayment,
		To:    constants.StatePaid,
		Actor: constants.ActorSystem,
		Guard: nil,
		Action: func(order *Order, transitionCtx TransitionContext) error {
			// making transition from pending payment to paid
			buyerActor := ledger.ActorTypeBuyer
			notes := "Payment received. Awaiting seller delivery."

			err := dbutils.ExecTx(transitionCtx.ctx, transitionCtx.db, func(tx *sqlx.Tx) error {
				// create payment to be held in ledger
				err := transitionCtx.ledgerService.Create(transitionCtx.ctx, &ledger.LedgerEntry{
					OrderID:   order.ID,
					EntryType: ledger.EntryTypeEscrow,
					ActorType: &buyerActor,
					Amount:    order.Amount,
					ActorID:   &order.BuyerID,
					Notes:     &notes,
				})

				if err != nil {
					slog.Error("Error when attempting to update ledger.",
						"orderID", order.ID,
					)
					return err
				}

				// update order
				transitionCtx.orderService.UpdateStateTx(
					transitionCtx.ctx,
					tx,
					order.ID,
					order.BuyerID,
					constants.StateCompleted,
				)

				return nil
			})

			if err != nil {
				return err
			}
			return nil
		},
	},
	Transition{},
}
