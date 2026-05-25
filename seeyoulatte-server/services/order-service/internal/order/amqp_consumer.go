package order

import (
	"log/slog"

	commonconstants "github.com/darkphotonKN/seeyoulatte-app/common/constants"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer scaffolds AMQP consumption for order-service. Stub for now — once
// auth-service publishes user.frozen / user.deleted events we'll bind those
// here and react (cancel pending orders, etc.).
type Consumer struct {
	service ConsumerService
	channel *amqp.Channel
}

type ConsumerService interface{}

func NewConsumer(service ConsumerService, ch *amqp.Channel) *Consumer {
	return &Consumer{service: service, channel: ch}
}

func (c *Consumer) Listen() {
	go c.consumeEvents()
	slog.Info("order-service consumer listening for events...")
}

func (c *Consumer) consumeEvents() {
	msgs, err := c.channel.Consume(
		commonconstants.OrderServiceEventsQueue,
		"",    // consumer
		false, // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,
	)
	if err != nil {
		slog.Error("order-service: failed to register consumer", "error", err)
		return
	}
	for msg := range msgs {
		slog.Debug("order-service: received event (no handler bound)",
			"routing_key", msg.RoutingKey)
		msg.Ack(false)
	}
}

func SetupAMQPInfrastructure(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		commonconstants.OrderEventsExchange, "topic", true, false, false, false, nil,
	); err != nil {
		return err
	}
	if _, err := ch.QueueDeclare(
		commonconstants.OrderServiceEventsQueue,
		true, false, false, false, nil,
	); err != nil {
		return err
	}
	slog.Info("order-service AMQP infrastructure setup complete",
		"exchange", commonconstants.OrderEventsExchange,
		"queue", commonconstants.OrderServiceEventsQueue)
	return nil
}
