package commonconstants

/**
NOTE: naming conventions are kept consistent across the codebase.

Simple rule:

Constant      |    Pattern                             |    Example
Exchange         {domain}.events                            auth.events
Routing Key      {resource}.{action}                        user.created
Queue            {service}.events                           auth-service.events

For the scaffolded MVP we use a single per-service "events" queue and split
per routing-key once real consumers come online.
**/

/**
* Exchanges — one per domain.
**/
const (
	AuthEventsExchange     = "auth.events"
	OrderEventsExchange    = "order.events"
	BaselineEventsExchange = "baseline.events"

	DlxEventsExchange = "dlx.exchange"
	RetryExchange     = "retry.exchange"
)

/**
* Routing keys, also acting as event names.
* {resource}.{action}
**/
const (
	// auth events (published by auth-service)
	UserCreated = "user.created"
	UserUpdated = "user.updated"
	UserDeleted = "user.deleted"
	UserFrozen  = "user.frozen"

	// order events (published by order-service)
	OrderCreated    = "order.created"
	OrderPaid       = "order.paid"
	OrderAccepted   = "order.accepted"
	OrderFulfilled  = "order.fulfilled"
	OrderCompleted  = "order.completed"
	OrderCancelled  = "order.cancelled"
	OrderDisputed   = "order.disputed"
	OrderRefunded   = "order.refunded"
)

/**
* Queue names — {service}.events
**/
const (
	AuthServiceEventsQueue     = "auth-service.events"
	OrderServiceEventsQueue    = "order-service.events"
	BaselineServiceEventsQueue = "baseline-service.events"
	ApiGatewayEventsQueue      = "api-gateway.events"
)
