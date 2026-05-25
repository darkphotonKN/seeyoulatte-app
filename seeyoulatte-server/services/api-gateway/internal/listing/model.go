package listing

import (
	"time"

	"github.com/google/uuid"
)

type Listing struct {
	ID                 uuid.UUID  `db:"id" json:"id"`
	SellerID           uuid.UUID  `db:"seller_id" json:"seller_id"`
	Title              string     `db:"title" json:"title"`
	Description        *string    `db:"description" json:"description,omitempty"`
	Category           string     `db:"category" json:"category"`
	Price              float64    `db:"price" json:"price"`
	Quantity           int        `db:"quantity" json:"quantity"`
	PickupInstructions *string    `db:"pickup_instructions" json:"pickup_instructions,omitempty"`
	ExpiresAt          *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	IsActive           bool       `db:"is_active" json:"is_active"`
	CreatedAt          time.Time  `db:"created_at" json:"created_at"`
}

type CreateListingRequest struct {
	Title              string     `json:"title" binding:"required,min=1,max=255"`
	Description        *string    `json:"description"`
	Category           string     `json:"category" binding:"required,oneof=product experience"`
	Price              float64    `json:"price" binding:"required,min=0.01"`
	Quantity           int        `json:"quantity" binding:"required,min=1"`
	PickupInstructions *string    `json:"pickup_instructions"`
	ExpiresAt          *time.Time `json:"expires_at"`
}

type UpdateListingRequest struct {
	Title              *string    `json:"title,omitempty"`
	Description        *string    `json:"description,omitempty"`
	Price              *float64   `json:"price,omitempty"`
	Quantity           *int       `json:"quantity,omitempty"`
	PickupInstructions *string    `json:"pickup_instructions,omitempty"`
	IsActive           *bool      `json:"is_active,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
}

type ListingWithSeller struct {
	ID                 uuid.UUID  `db:"listing_id" json:"id"`
	SellerID           uuid.UUID  `db:"seller_id" json:"seller_id"`
	UserIsFrozen       bool       `db:"user_is_frozen" json:"user_is_frozen"`
	Title              string     `db:"title" json:"title"`
	Description        *string    `db:"description" json:"description,omitempty"`
	Category           string     `db:"category" json:"category"`
	Price              float64    `db:"price" json:"price"`
	Quantity           int        `db:"quantity" json:"quantity"`
	PickupInstructions *string    `db:"pickup_instructions" json:"pickup_instructions,omitempty"`
	ExpiresAt          *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	IsActive           bool       `db:"is_active" json:"is_active"`
	ListingCreatedAt   time.Time  `db:"listing_created_at" json:"listing_created_at"`
}

