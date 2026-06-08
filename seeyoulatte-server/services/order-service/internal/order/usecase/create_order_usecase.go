package usecase

import (
	"context"

	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order/domain"
	"github.com/google/uuid"
)

type CreateOrderUC struct {
	orderRepo domain.OrderRepository
	publisher EventPublisher
}

func NewCreateOrderUC(orderRepo domain.OrderRepository, publisher EventPublisher) *CreateOrderUC {
	return &CreateOrderUC{
		orderRepo: orderRepo,
		publisher: publisher,
	}
}

type CreateOrderParams struct {
	BuyerID   uuid.UUID
	SellerID  uuid.UUID
	ListingID uuid.UUID
	Quantity  int32
	Amount    float64
}

func (uc *CreateOrderUC) Handle(ctx context.Context, params CreateOrderParams) (*domain.Order, error) {
	// birth with invariants validated
	order, err := domain.NewOrder(params.ListingID, params.BuyerID, params.SellerID, int(params.Quantity), params.Amount)

	if err != nil {
		// simply propogate domain error
		return nil, err
	}

	// propogate insert error
	if err := uc.orderRepo.Insert(ctx, order); err != nil {
		return nil, err
	}

	// publish event
	snapshot := order.Snapshot()
	if err := uc.publisher.PublishOrderCreated(ctx, &PublishOrderCreatedParams{
		OrderID:   snapshot.ID,
		BuyerID:   snapshot.BuyerID,
		SellerID:  snapshot.SellerID,
		ListingID: snapshot.ListingID,
		Amount:    snapshot.Amount,
	}); err != nil {
		return nil, err
	}

	return order, nil
}
