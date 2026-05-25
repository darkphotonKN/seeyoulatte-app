package config

import (
	"log/slog"

	pb "github.com/darkphotonKN/seeyoulatte-app/common/api/proto/order"
	commonbroker "github.com/darkphotonKN/seeyoulatte-app/common/broker"
	"github.com/darkphotonKN/seeyoulatte-app/common/discovery"
	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/ledger"
	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order"
	"github.com/jmoiron/sqlx"
	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

// SetupServices wires the order-service dependencies and returns a configured
// gRPC server with handlers registered.
//
// TODO(integration): once auth-service has VerifyUserNotFrozen wired here,
// inject an auth-service gRPC client into order.NewService. For now the
// freeze check is no-op'd inside service.go.
func SetupServices(db *sqlx.DB, amqpChannel *amqp.Channel, _ discovery.Registry) *grpc.Server {
	publisher := commonbroker.NewAmqpPublisher(amqpChannel)

	ledgerRepo := ledger.NewRepository(db)
	ledgerLogger := slog.Default().With("component", "ledger")
	ledgerService := ledger.NewService(ledgerRepo, ledgerLogger)

	orderRepo := order.NewRepository(db)
	orderService := order.NewService(orderRepo, db, slog.Default().With("component", "order"), ledgerService, publisher)
	handler := order.NewHandler(orderService)

	if err := order.SetupAMQPInfrastructure(amqpChannel); err != nil {
		slog.Error("Failed to setup AMQP infrastructure", "error", err)
	}
	consumer := order.NewConsumer(orderService, amqpChannel)
	consumer.Listen()

	grpcServer := grpc.NewServer()
	pb.RegisterOrderServiceServer(grpcServer, handler)

	slog.Info("order-service initialized successfully")
	return grpcServer
}
