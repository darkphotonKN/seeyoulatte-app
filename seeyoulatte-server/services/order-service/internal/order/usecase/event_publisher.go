package usecase

import (
	"context"

	"github.com/google/uuid"
)

type EventPublisher interface {
	PublishOrderStateChanged(ctx context.Context, orderID uuid.UUID, from, to, actor string) error
	PublishOrderCreated(ctx context.Context, params *PublishOrderCreatedParams) error
}

type PublishOrderCreatedParams struct {
	OrderID   uuid.UUID
	BuyerID   uuid.UUID
	SellerID  uuid.UUID
	ListingID uuid.UUID
	Amount    float64
}
