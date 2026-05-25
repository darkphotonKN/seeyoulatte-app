package domain

import (
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

// attempt a payment
func (o *Order) Pay() error {
	// validate and mutate struct
	return o.transitionTo(StatePaid)
}
