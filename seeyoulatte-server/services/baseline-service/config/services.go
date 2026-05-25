package config

import (
	"log/slog"

	pb "github.com/darkphotonKN/seeyoulatte-app/common/api/proto/baseline"
	commonbroker "github.com/darkphotonKN/seeyoulatte-app/common/broker"
	"github.com/darkphotonKN/seeyoulatte-app/common/discovery"
	"github.com/darkphotonKN/seeyoulatte-app/services/baseline-service/internal/baseline"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

// SetupServices wires the baseline-service dependencies and returns a configured
// gRPC server. baseline-service is decorative — no DB; exists as a template
// for future extractions.
func SetupServices(amqpChannel *amqp.Channel, _ discovery.Registry) *grpc.Server {
	publisher := commonbroker.NewAmqpPublisher(amqpChannel)

	service := baseline.NewService(publisher)
	handler := baseline.NewHandler(service)

	if err := baseline.SetupAMQPInfrastructure(amqpChannel); err != nil {
		slog.Error("Failed to setup AMQP infrastructure", "error", err)
	}
	consumer := baseline.NewConsumer(service, amqpChannel)
	consumer.Listen()

	grpcServer := grpc.NewServer()
	pb.RegisterBaselineServiceServer(grpcServer, handler)

	slog.Info("baseline-service initialized successfully")
	return grpcServer
}
