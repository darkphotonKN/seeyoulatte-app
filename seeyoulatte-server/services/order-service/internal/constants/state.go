package constants

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

type Actor string

const (
	ActorSystem Actor = "system"
	ActorBuyer  Actor = "buyer"
	ActorSeller Actor = "seller"
	ActorAdmin  Actor = "admin"
)
