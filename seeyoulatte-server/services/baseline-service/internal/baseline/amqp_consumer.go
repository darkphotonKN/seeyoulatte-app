package baseline

import (
	"log/slog"

	commonconstants "github.com/darkphotonKN/seeyoulatte-app/common/constants"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Consumer scaffolds AMQP consumption for baseline-service. baseline-service is
// decorative, so the consumer is a no-op loop — its purpose is to demonstrate
// the consumer wiring (exchange + queue + bindings + per-message switch) that
// real services will replicate.
type Consumer struct {
	service ConsumerService
	channel *amqp.Channel
}

// ConsumerService is the slice of the service the consumer needs.
// Empty for now — real consumers extend this with their handler dependencies.
type ConsumerService interface{}

func NewConsumer(service ConsumerService, ch *amqp.Channel) *Consumer {
	return &Consumer{service: service, channel: ch}
}

func (c *Consumer) Listen() {
	go c.consumeEvents()
	slog.Info("baseline-service consumer listening for events...")
}

func (c *Consumer) consumeEvents() {
	msgs, err := c.channel.Consume(
		commonconstants.BaselineServiceEventsQueue,
		"",    // consumer
		false, // autoAck
		false, // exclusive
		false, // noLocal
		false, // noWait
		nil,
	)
	if err != nil {
		slog.Error("baseline-service: failed to register consumer", "error", err)
		return
	}

	for msg := range msgs {
		slog.Debug("baseline-service: received event (decorative, no handler)",
			"routing_key", msg.RoutingKey,
		)
		msg.Ack(false)
	}
}

// SetupAMQPInfrastructure declares baseline.events exchange + a per-service
// inbound queue. No bindings — baseline-service consumes nothing.
func SetupAMQPInfrastructure(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		commonconstants.BaselineEventsExchange, "topic", true, false, false, false, nil,
	); err != nil {
		return err
	}

	if _, err := ch.QueueDeclare(
		commonconstants.BaselineServiceEventsQueue,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,
	); err != nil {
		return err
	}

	slog.Info("baseline-service AMQP infrastructure setup complete",
		"exchange", commonconstants.BaselineEventsExchange,
		"queue", commonconstants.BaselineServiceEventsQueue,
	)
	return nil
}
