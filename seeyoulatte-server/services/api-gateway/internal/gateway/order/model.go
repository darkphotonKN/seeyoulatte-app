package ordergw

import (
	"time"

	"github.com/google/uuid"
)

// Order mirrors the order-service's HTTP/JSON shape. Kept identical to the
// monolith's previous Order so the seeyoulatte-client sees no change.
type Order struct {
	ID              uuid.UUID  `json:"id"`
	ListingID       uuid.UUID  `json:"listing_id"`
	BuyerID         uuid.UUID  `json:"buyer_id"`
	SellerID        uuid.UUID  `json:"seller_id"`
	Quantity        int        `json:"quantity"`
	Amount          float64    `json:"amount"`
	State           string     `json:"state"`
	SellerRespondBy *time.Time `json:"seller_respond_by,omitempty"`
	ReviewEndsAt    *time.Time `json:"review_ends_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
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
