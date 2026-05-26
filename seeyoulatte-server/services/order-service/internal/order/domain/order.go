package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type OrderState string

const (
	StatePendingPayment OrderState = "pending_payment"
	StatePaid           OrderState = "paid"
	StateAccepted       OrderState = "accepted"
	StateFulfilled      OrderState = "fulfilled"
	StateCompleted      OrderState = "completed"
	StateCancelled      OrderState = "cancelled"
	StateDisputed       OrderState = "disputed"
	StateRefunded       OrderState = "refunded"
)

var (
	ErrInvalidQuantity = errors.New("invalid quanity")
	ErrInvalidAmount   = errors.New("invalid amount")
	ErrBuyerIsSeller   = errors.New("buyer is seller")
)

type Order struct {
	id              uuid.UUID
	listingID       uuid.UUID
	buyerID         uuid.UUID
	sellerID        uuid.UUID
	quantity        int
	amount          float64
	state           OrderState
	sellerRespondBy *time.Time
	reviewEndsAt    *time.Time
	createdAt       time.Time
}

func NewOrder(listingID, buyerID, sellerID uuid.UUID, quantity int, amount float64) (*Order, error) {

	// guard for invariant violation
	if quantity <= 0 {
		return nil, ErrInvalidQuantity
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if buyerID == sellerID {
		return nil, ErrBuyerIsSeller
	}

	return &Order{
		id:        uuid.New(),
		listingID: listingID,
		buyerID:   buyerID,
		sellerID:  sellerID,
		quantity:  quantity,
		amount:    amount,
		state:     StatePendingPayment, // intialize only acceptable starting state
		createdAt: time.Now(),
	}, nil
}

// NOTE: Reconstitute rebuilds an existing Order from persisted data (e.g. a DB row).
// Unlike NewOrder it runs NO birth invariants and assigns NO new identity
// the order was already valid when first created; we are restoring it, not minting it.
// This exists because Order's fields are private: the repository (a different
// package) cannot build the struct literally, so the domain exposes this door.
// ALSO: can use sanity checks like ID and state validation, but don't redo business
// logic checks.
func Reconstitute(
	id, listingID, buyerID, sellerID uuid.UUID,
	quantity int,
	amount float64,
	state OrderState,
	sellerRespondBy, reviewEndsAt *time.Time,
	createdAt time.Time,
) *Order {
	return &Order{
		id:              id,
		listingID:       listingID,
		buyerID:         buyerID,
		sellerID:        sellerID,
		quantity:        quantity,
		amount:          amount,
		state:           state,
		sellerRespondBy: sellerRespondBy,
		reviewEndsAt:    reviewEndsAt,
		createdAt:       createdAt,
	}
}

// attempt a payment
func (o *Order) Pay() error {
	// validate and mutate struct
	return o.transitionTo(StatePaid)
}

// seller accepts a paid order and commits to fulfilling it
func (o *Order) Accept() error {
	return o.transitionTo(StateAccepted)
}

// seller marks the order as delivered to the buyer
func (o *Order) Fulfill() error {
	return o.transitionTo(StateFulfilled)
}

// buyer raises a dispute against a fulfilled order
func (o *Order) Dispute() error {
	return o.transitionTo(StateDisputed)
}

// finalize the order — review window closed, or dispute resolved in seller's favor
func (o *Order) Complete() error {
	return o.transitionTo(StateCompleted)
}

// refund the buyer — only valid from a disputed state
func (o *Order) Refund() error {
	return o.transitionTo(StateRefunded)
}

// cancel the order — valid from pending_payment or paid
func (o *Order) Cancel() error {
	return o.transitionTo(StateCancelled)
}
