package order

import (
	"time"

	"github.com/google/uuid"
)

type Order struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	ListingID       uuid.UUID  `db:"listing_id" json:"listing_id"`
	BuyerID         uuid.UUID  `db:"buyer_id" json:"buyer_id"`
	SellerID        uuid.UUID  `db:"seller_id" json:"seller_id"`
	Quantity        int        `db:"quantity" json:"quantity"`
	Amount          float64    `db:"amount" json:"amount"`
	State           string     `db:"state" json:"state"`
	SellerRespondBy *time.Time `db:"seller_respond_by" json:"seller_respond_by,omitempty"`
	ReviewEndsAt    *time.Time `db:"review_ends_at" json:"review_ends_at,omitempty"`
	CreatedAt       time.Time  `db:"created_at" json:"created_at"`
}

type CreateOrderRequest struct {
	ListingID uuid.UUID `json:"listing_id" binding:"required"`
	Quantity  int       `json:"quantity" binding:"required,min=1"`
}

type UpdateOrderRequest struct {
	State           *string    `json:"state,omitempty"`
	SellerRespondBy *time.Time `json:"seller_respond_by,omitempty"`
	ReviewEndsAt    *time.Time `json:"review_ends_at,omitempty"`
}