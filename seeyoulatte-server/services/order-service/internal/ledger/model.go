package ledger

import (
	"time"

	"github.com/google/uuid"
)

// EntryType represents the type of ledger entry
type EntryType string

const (
	EntryTypeEscrow   EntryType = "ESCROW"
	EntryTypePayout   EntryType = "PAYOUT"
	EntryTypeRefund   EntryType = "REFUND"
	EntryTypeReversal EntryType = "REVERSAL"
)

// ActorType represents who triggered the ledger entry
type ActorType string

const (
	ActorTypeBuyer  ActorType = "BUYER"
	ActorTypeSeller ActorType = "SELLER"
	ActorTypeSystem ActorType = "SYSTEM"
	ActorTypeAdmin  ActorType = "ADMIN"
)

// LedgerEntry represents an immutable financial record
// This is an append-only table - entries are never updated or deleted
type LedgerEntry struct {
	ID         int        `db:"id" json:"id"`
	OrderID    uuid.UUID  `db:"order_id" json:"order_id"`
	EntryType  EntryType  `db:"entry_type" json:"entry_type"`
	Amount     float64    `db:"amount" json:"amount"` // Always positive, direction implied by entry_type
	ActorID    *uuid.UUID `db:"actor_id" json:"actor_id,omitempty"`
	ActorType  *ActorType `db:"actor_type" json:"actor_type,omitempty"`
	Notes      *string    `db:"notes" json:"notes,omitempty"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
}

// CreateLedgerEntryRequest represents the request to create a new ledger entry
type CreateLedgerEntryRequest struct {
	OrderID   uuid.UUID  `json:"order_id" binding:"required"`
	EntryType EntryType  `json:"entry_type" binding:"required"`
	Amount    float64    `json:"amount" binding:"required,gt=0"`
	ActorID   *uuid.UUID `json:"actor_id,omitempty"`
	ActorType *ActorType `json:"actor_type,omitempty"`
	Notes     *string    `json:"notes,omitempty"`
}

// Validate checks if the entry type is valid
func (e EntryType) IsValid() bool {
	switch e {
	case EntryTypeEscrow, EntryTypePayout, EntryTypeRefund, EntryTypeReversal:
		return true
	default:
		return false
	}
}

// Validate checks if the actor type is valid
func (a ActorType) IsValid() bool {
	switch a {
	case ActorTypeBuyer, ActorTypeSeller, ActorTypeSystem, ActorTypeAdmin:
		return true
	default:
		return false
	}
}

// BalanceCalculation represents the result of calculating an order's escrow balance
type BalanceCalculation struct {
	OrderID       uuid.UUID `json:"order_id"`
	EscrowBalance float64   `json:"escrow_balance"`
	TotalEscrow   float64   `json:"total_escrow"`
	TotalPayout   float64   `json:"total_payout"`
	TotalRefund   float64   `json:"total_refund"`
	TotalReversal float64   `json:"total_reversal"`
}