package order

import (
	"context"
	"log/slog"

	eventspb "github.com/darkphotonKN/seeyoulatte-app/common/api/proto/events"
	commonbroker "github.com/darkphotonKN/seeyoulatte-app/common/broker"
	commonconstants "github.com/darkphotonKN/seeyoulatte-app/common/constants"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *service) PublishOrderCreated(ctx context.Context, o *Order) {
	body, err := proto.Marshal(&eventspb.OrderCreatedEvent{
		Id:        o.ID.String(),
		BuyerId:   o.BuyerID.String(),
		SellerId:  o.SellerID.String(),
		ListingId: o.ListingID.String(),
		Amount:    o.Amount,
		CreatedAt: timestamppb.New(o.CreatedAt),
	})
	if err != nil {
		slog.Error("failed to marshal order.created event", "error", err, "order_id", o.ID)
		return
	}
	if err := s.publisher.PublishWithContext(ctx,
		commonconstants.OrderEventsExchange,
		commonconstants.OrderCreated,
		commonbroker.Message{
			ContentType:  "application/protobuf",
			Body:         body,
			DeliveryMode: commonbroker.Persistent,
		},
	); err != nil {
		slog.Error("failed to publish order.created", "error", err, "order_id", o.ID)
	}
}

func (s *service) PublishOrderStateChanged(ctx context.Context, o *Order, fromState, toState, actor string) {
	body, err := proto.Marshal(&eventspb.OrderStateChangedEvent{
		Id:        o.ID.String(),
		FromState: fromState,
		ToState:   toState,
		Actor:     actor,
		ChangedAt: timestamppb.Now(),
	})
	if err != nil {
		slog.Error("failed to marshal order.state_changed event", "error", err, "order_id", o.ID)
		return
	}
	routingKey := stateToRoutingKey(toState)
	if err := s.publisher.PublishWithContext(ctx,
		commonconstants.OrderEventsExchange,
		routingKey,
		commonbroker.Message{
			ContentType:  "application/protobuf",
			Body:         body,
			DeliveryMode: commonbroker.Persistent,
		},
	); err != nil {
		slog.Error("failed to publish order state change", "error", err, "order_id", o.ID, "routing_key", routingKey)
	}
}

func stateToRoutingKey(state string) string {
	switch state {
	case "paid":
		return commonconstants.OrderPaid
	case "accepted":
		return commonconstants.OrderAccepted
	case "fulfilled":
		return commonconstants.OrderFulfilled
	case "completed":
		return commonconstants.OrderCompleted
	case "cancelled":
		return commonconstants.OrderCancelled
	case "disputed":
		return commonconstants.OrderDisputed
	case "refunded":
		return commonconstants.OrderRefunded
	default:
		return "order." + state
	}
}
