package order

import (
	"context"
	"fmt"
	"log/slog"

	commonbroker "github.com/darkphotonKN/seeyoulatte-app/common/broker"
	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/constants"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	Create(ctx context.Context, order *Order) error
	CreateTx(ctx context.Context, tx *sqlx.Tx, order *Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*Order, error)
	GetAll(ctx context.Context) ([]Order, error)
	GetPendingPaymentOrdersByUser(ctx context.Context, userID uuid.UUID) ([]*Order, error)
	Update(ctx context.Context, order *Order) error
	UpdateStateTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, newState constants.OrderState) (*Order, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type LedgerService interface {
	CreateEscrowEntry(ctx context.Context, orderID uuid.UUID, amount float64, actorID uuid.UUID) error
	CreatePayoutEntry(ctx context.Context, orderID uuid.UUID, amount float64) error
	CreateRefundEntry(ctx context.Context, orderID uuid.UUID, amount float64, notes string) error
}

type service struct {
	repo          Repository
	db            *sqlx.DB
	ledgerService LedgerService
	publisher     commonbroker.Publisher
	logger        *slog.Logger
}

func NewService(repo Repository, db *sqlx.DB, logger *slog.Logger, ledgerService LedgerService, publisher commonbroker.Publisher) *service {
	return &service{
		repo:          repo,
		db:            db,
		ledgerService: ledgerService,
		publisher:     publisher,
		logger:        logger,
	}
}

// Create persists a new order. The caller (api-gateway) is responsible for
// validating the listing + seller and passing pre-validated data here.
//
// TODO(integration): once auth-service has gRPC wired into this service,
// re-introduce a buyer-frozen check here via authClient.VerifyUserNotFrozen.
// For now the freeze check is the gateway's responsibility (and currently
// missing — left broken per user direction).
func (s *service) Create(ctx context.Context, buyerID uuid.UUID, listingID, sellerID uuid.UUID, quantity int, amount float64) (*Order, error) {
	if buyerID == sellerID {
		return nil, fmt.Errorf("cannot purchase your own listing")
	}

	o := &Order{
		ListingID: listingID,
		BuyerID:   buyerID,
		SellerID:  sellerID,
		Quantity:  quantity,
		Amount:    amount,
		State:     string(constants.StatePendingPayment),
	}
	if err := s.repo.Create(ctx, o); err != nil {
		return nil, fmt.Errorf("creating order: %w", err)
	}

	s.logger.Info("order created",
		slog.String("order_id", o.ID.String()),
		slog.String("buyer_id", buyerID.String()),
		slog.String("seller_id", sellerID.String()))

	s.PublishOrderCreated(ctx, o)
	return o, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) GetAll(ctx context.Context) ([]Order, error) {
	return s.repo.GetAll(ctx)
}

func (s *service) GetPendingPaymentOrdersByUser(ctx context.Context, userID uuid.UUID) ([]*Order, error) {
	return s.repo.GetPendingPaymentOrdersByUser(ctx, userID)
}

func (s *service) Update(ctx context.Context, id, userID uuid.UUID, req *UpdateOrderRequest) (*Order, error) {
	o := &Order{ID: id}
	if req.State != nil {
		o.State = *req.State
	}
	if req.SellerRespondBy != nil {
		o.SellerRespondBy = req.SellerRespondBy
	}
	if req.ReviewEndsAt != nil {
		o.ReviewEndsAt = req.ReviewEndsAt
	}

	if err := s.repo.Update(ctx, o); err != nil {
		return nil, fmt.Errorf("updating order: %w", err)
	}

	s.logger.Info("order updated",
		slog.String("order_id", id.String()),
		slog.String("user_id", userID.String()))

	return s.repo.GetByID(ctx, id)
}

func (s *service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting order: %w", err)
	}
	s.logger.Info("order deleted",
		slog.String("order_id", id.String()),
		slog.String("user_id", userID.String()))
	return nil
}

// TransitionState moves an order through its state machine. Currently only
// pending_payment → paid is wired (creates the ESCROW ledger entry); the rest
// of the transitions table remains a TODO from the monolith era.
func (s *service) TransitionState(ctx context.Context, orderID uuid.UUID, targetState string, actor string) (*Order, error) {
	o, err := s.repo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	from := constants.OrderState(o.State)
	to := constants.OrderState(targetState)

	// Minimal transition wiring: pending_payment → paid creates ESCROW + advances state.
	if from == constants.StatePendingPayment && to == constants.StatePaid {
		if err := s.ledgerService.CreateEscrowEntry(ctx, o.ID, o.Amount, o.BuyerID); err != nil {
			return nil, fmt.Errorf("creating escrow entry: %w", err)
		}
		o.State = string(constants.StatePaid)
		if err := s.repo.Update(ctx, o); err != nil {
			return nil, err
		}
		s.PublishOrderStateChanged(ctx, o, string(from), string(to), actor)
		return o, nil
	}

	// TODO(state-machine): port the full transitions table from the monolith
	// (see services/order-service/internal/order/transitions.go for shape).
	return nil, fmt.Errorf("transition %s → %s not implemented", from, to)
}

