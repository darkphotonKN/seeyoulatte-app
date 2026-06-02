package dto

import (
	"time"

	"github.com/google/uuid"
)

type OrderDetail struct {
	ID              uuid.UUID  `db:"id"`
	ListingID       uuid.UUID  `db:"listing_id"`
	BuyerID         uuid.UUID  `db:"buyer_id"`
	SellerID        uuid.UUID  `db:"seller_id"`
	Quantity        int        `db:"quantity"`
	Amount          float64    `db:"amount"`
	State           string     `db:"state"`
	SellerRespondBy *time.Time `db:"seller_respond_by"`
	ReviewEndsAt    *time.Time `db:"review_ends_at"`
	CreatedAt       time.Time  `db:"created_at"`
}
