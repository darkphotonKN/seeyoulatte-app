package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order/domain"
	"github.com/google/uuid"
)

type TransitionOrderUC struct {
	orderRepo domain.OrderRepository
	ledger    LedgerService  // narrow ISP interface, only methods that THIS usecase needs
	publisher EventPublisher // narrow ISP interface, only publishes state changed
}

type LedgerService interface {
	CreateEscrowEntry(ctx context.Context, orderID uuid.UUID, amount float64, actorID uuid.UUID) error
	CreatePayoutEntry(ctx context.Context, orderID uuid.UUID, amount float64) error
	CreateRefundEntry(ctx context.Context, orderID uuid.UUID, amount float64, notes string) error
}

type EventPublisher interface {
	PublishOrderStateChanged(ctx context.Context, orderID uuid.UUID, from, to, actor string) error
}

func NewTransitionOrderUC(repo domain.OrderRepository, ledger LedgerService, pub EventPublisher) *TransitionOrderUC {
	return &TransitionOrderUC{orderRepo: repo, ledger: ledger, publisher: pub}
}

func (uc *TransitionOrderUC) Handle(ctx context.Context, orderID uuid.UUID, target domain.OrderState, actor string, now time.Time) (*domain.Order, error) {
	// load order from repo
	// repo builds it from reconstitute not mapper (proper DDD)
	order, err := uc.orderRepo.FindByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	// capture current STATE into a variable
	fromState := order.State()

	// update state based on FSM transition verbs
	switch target {
	case domain.StatePaid:
		err = order.Pay(now)
	case domain.StateAccepted:
		err = order.Accept()
	case domain.StateFulfilled:
		err = order.Fulfill(now)
	case domain.StateDisputed:
		err = order.Dispute()
	case domain.StateCompleted:
		err = order.Complete()
	case domain.StateCancelled:
		err = order.Cancel()
	case domain.StateRefunded:
		err = order.Refund()
	default:
		return nil, fmt.Errorf("unsupported target state: %s", target)
	}

	if err != nil {
		return nil, err // FSM rejected so no mutation
	}

	// 4. SIDE EFFECTS (cross-aggregate, e.g. ledger) — deferred.
	//    TODO(money): escrow on paid, payout on completed, refund on cancelled/refunded.
	//    TODO(consistency): wrap side-effects + save in one tx when added.

	// 5. SAVE the mutated aggregate via the repo. Return early on error.
	if err := uc.orderRepo.Save(ctx, order); err != nil {
		return nil, err
	}

	// 6. PUBLISH state-changed. Convert domain states → string at the call site
	//    (publisher speaks wire/string, not domain). Log-and-continue: a publish
	//    failure must NOT fail the transition — the writes already committed.

	if err := uc.publisher.PublishOrderStateChanged(ctx, order.ID(), string(fromState), string(target), actor); err != nil {
		slog.Error("publish order state change failed", "order_id", order.ID(), "error", err)
	}

	// return mutated order
	return order, nil
}
