package events

import (
	"context"
	"fmt"

	eventspb "github.com/darkphotonKN/seeyoulatte-app/common/api/proto/events"
	commonbroker "github.com/darkphotonKN/seeyoulatte-app/common/broker"
	commonevents "github.com/darkphotonKN/seeyoulatte-app/common/constants"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderEventPublisher struct {
	broker commonbroker.Publisher
}

func NewOrderEventPublisher(broker commonbroker.Publisher) *OrderEventPublisher {
	return &OrderEventPublisher{
		broker: broker,
	}
}

func (o *OrderEventPublisher) PublishOrderStateChanged(ctx context.Context, orderID uuid.UUID, from, to, actor string) error {
	evtPb := &eventspb.OrderStateChangedEvent{
		Id:        orderID.String(),
		FromState: string(from),
		ToState:   string(to),
		Actor:     actor,
		ChangedAt: timestamppb.Now(),
	}

	evtPbMarshalled, err := proto.Marshal(evtPb)

	if err != nil {
		return fmt.Errorf("error marshalling into proto error for order %s: %w", orderID, err)
	}

	evtMsg := commonbroker.Message{
		ContentType:  "application/protobuf",
		Body:         evtPbMarshalled,
		DeliveryMode: commonbroker.Persistent,
	}

	routingKey := o.stateToRoutingKey(to)

	if err := o.broker.PublishWithContext(ctx, commonevents.OrderEventsExchange, routingKey, evtMsg); err != nil {
		return fmt.Errorf("error publishing order state transition order id %s : %w", orderID, err)
	}

	return nil
}

// Maps the target state string to the corresponding routing key constant.
func (o *OrderEventPublisher) stateToRoutingKey(state string) string {
	switch state {
	case "paid":
		return commonevents.OrderPaid
	case "accepted":
		return commonevents.OrderAccepted
	case "cancelled":
		return commonevents.OrderCancelled
	case "fulfilled":
		return commonevents.OrderFulfilled
	case "completed":
		return commonevents.OrderCompleted
	case "disputed":
		return commonevents.OrderDisputed
	case "refunded":
		return commonevents.OrderRefunded
	default:
		return "order." + string(state)
	}
}
