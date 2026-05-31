package events

import (
	"context"
	"errors"
	"time"

	eventspb "github.com/darkphotonKN/seeyoulatte-app/common/api/proto/events"
	commonevents "github.com/darkphotonKN/seeyoulatte-app/common/constants"
	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order/domain"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	ErrProtoMarshal = errors.New("error marshalling into protobuf")
)

type OrderEventPublisher struct {
	publisher EventPublisher
}

// abstract this to a common method without leaky abstractions
type EventPublisher interface {
	Publish(exchange, key string, mandatory, immediate bool, msg EventMsg) error
}
type EventMsg struct {
	ContentType string
	Timestamp   time.Time
	Body        []byte
}

// func Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
type AmqpAdator struct {
	amqpCh *amqp.Channel
}

func NewAmqpAdaptor(amqpCh *amqp.Channel) EventPublisher {
	return &AmqpAdator{amqpCh: amqpCh}
}

func (a *AmqpAdator) Publish(exchange, key string, mandatory, immediate bool, msg EventMsg) error {
	// publish with original method

	// convert our event msg to amqp specific
	return a.amqpCh.Publish(exchange, key, mandatory, immediate, amqp.Publishing{
		ContentType: msg.ContentType,
		Timestamp:   msg.Timestamp,
		Body:        msg.Body,
	})
}

func (o *OrderEventPublisher) Publish(ctx context.Context, id uuid.UUID, from, to domain.OrderState, actor string) error {
	evtPb := &eventspb.OrderStateChangedEvent{
		Id:        id.String(),
		FromState: string(from),
		ToState:   string(to),
		Actor:     actor,
		ChangedAt: timestamppb.Now(),
	}

	evtPbMarshalled, err := proto.Marshal(evtPb)

	if err != nil {
		return ErrProtoMarshal
	}

	evtMsg := EventMsg{
		ContentType: "protobuf",
		Timestamp:   evtPb.ChangedAt.AsTime(),
		Body:        evtPbMarshalled,
	}

	routingKey := o.stateToRoutingKey(to)

	return o.publisher.Publish(commonevents.OrderEventsExchange, routingKey, true, true, evtMsg)
}

// Maps the target state string to the corresponding routing key constant.
func (o *OrderEventPublisher) stateToRoutingKey(state domain.OrderState) string {
	switch state {
	case domain.StatePaid:
		return commonevents.OrderPaid
	default:
		return "order." + string(state)
	}
}
