package domain

import "errors"

var (
	ErrTransitionNotAllowed = errors.New("transition attempted was not allowed")
)

// finite state machine rules for orders
var validTrans = map[OrderState]map[OrderState]bool{
	StatePaid:           {StateAccepted: true, StateCancelled: true},
	StatePendingPayment: {StatePaid: true, StateCancelled: true},
	StateAccepted:       {StateFulfilled: true},
	StateFulfilled:      {StateDisputed: true, StateCompleted: true},
	StateDisputed:       {StateRefunded: true, StateCompleted: true},
}

// assists in making transition from one state to another
func (o *Order) transitionTo(to OrderState) error {
	// validate transition
	vt, ok := validTrans[o.state]

	// handle unacceptable cases first
	if !ok {
		return ErrTransitionNotAllowed
	}
	if !vt[to] {
		return ErrTransitionNotAllowed
	}

	// allow transition
	o.state = to
	return nil
}
