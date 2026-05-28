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

func (s OrderState) IsValid() bool {
	switch s {
	case StateAccepted, StateCancelled, StateCompleted, StateDisputed, StateFulfilled, StatePendingPayment, StatePaid, StateRefunded:
		return true
	default:
		return false
	}
}

var (
	ErrInvalidQuantity = errors.New("invalid quantity")
	ErrInvalidAmount   = errors.New("invalid amount")
	ErrBuyerIsSeller   = errors.New("buyer is seller")
	ErrInvalidID       = errors.New("invalid order id")
	ErrInvalidState    = errors.New("invalid order state")
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

type ReconstituteParams struct {
	ID              uuid.UUID
	ListingID       uuid.UUID
	BuyerID         uuid.UUID
	SellerID        uuid.UUID
	Quantity        int
	Amount          float64
	State           OrderState
	SellerRespondBy *time.Time
	ReviewEndsAt    *time.Time
	CreatedAt       time.Time
}

func Reconstitute(p ReconstituteParams) (*Order, error) {
	// sanity checks
	if p.ID == uuid.Nil {
		return nil, ErrInvalidID
	}
	if !p.State.IsValid() {
		return nil, ErrInvalidState
	}
	return &Order{
		id:              p.ID,
		listingID:       p.ListingID,
		buyerID:         p.BuyerID,
		sellerID:        p.SellerID,
		quantity:        p.Quantity,
		amount:          p.Amount,
		state:           p.State,
		sellerRespondBy: p.SellerRespondBy,
		reviewEndsAt:    p.ReviewEndsAt,
		createdAt:       p.CreatedAt,
	}, nil
}

// DDD verbs

// attempt a payment
// NOTE: DISCUSS BUG WITH KIKI & NICK
func (o *Order) Pay(now time.Time) error {
	// set deadline for seller
	deadline := now.Add(time.Hour * 24)
	o.sellerRespondBy = &deadline

	// validate and mutate struct
	return o.transitionTo(StatePaid)
}

// seller accepts a paid order and commits to fulfilling it
func (o *Order) Accept() error {
	return o.transitionTo(StateAccepted)
}

// seller marks the order as delivered to the buyer
func (o *Order) Fulfill(now time.Time) error {
	if err := o.transitionTo(StateFulfilled); err != nil {
		return ErrTransitionNotAllowed
	}

	deadline := now.Add(time.Hour * 24)
	o.reviewEndsAt = &deadline

	return nil
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

// OrderSnapshot is a read-only copy of an order's state for the edges
// (persistence mapping, gRPC response, event payloads). One-way: entity → outside.
// Mutating a snapshot never affects the entity. Writes still go through verbs.
type OrderSnapshot struct {
	ID              uuid.UUID
	ListingID       uuid.UUID
	BuyerID         uuid.UUID
	SellerID        uuid.UUID
	Quantity        int
	Amount          float64
	State           OrderState
	SellerRespondBy *time.Time
	ReviewEndsAt    *time.Time
	CreatedAt       time.Time
}

func (o *Order) Snapshot() OrderSnapshot {
	return OrderSnapshot{
		ID:              o.id,
		ListingID:       o.listingID,
		BuyerID:         o.buyerID,
		SellerID:        o.sellerID,
		Quantity:        o.quantity,
		Amount:          o.amount,
		State:           o.state,
		SellerRespondBy: o.sellerRespondBy,
		ReviewEndsAt:    o.reviewEndsAt,
		CreatedAt:       o.createdAt,
	}
}

// getters
func (o *Order) ID() uuid.UUID     { return o.id }
func (o *Order) State() OrderState { return o.state }
