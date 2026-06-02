package config

import (
	"log/slog"

	pb "github.com/darkphotonKN/seeyoulatte-app/common/api/proto/order"
	commonbroker "github.com/darkphotonKN/seeyoulatte-app/common/broker"
	"github.com/darkphotonKN/seeyoulatte-app/common/discovery"
	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/ledger"
	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order"
	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order/events"
	ordergrpc "github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order/grpc"
	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order/repository"
	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order/usecase"
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
	// --- publisher ---
	publisher := commonbroker.NewAmqpPublisher(amqpChannel)

	// --- domains ---

	// -- ledger setup --
	// TODO: temp before we implement ledger in DDD
	ledgerService := ledger.NewFacade(&ledger.CreateEscrowEntryUC{}, &ledger.CreatePayoutEntryUC{}, &ledger.CreateRefundEntryUC{})

	// -- orders set up --
	transitionEventPublisher := events.NewOrderEventPublisher(publisher)
	orderRepo := repository.NewOrderRepository(db)
	transitionOrderUC := usecase.NewTransitionOrderUC(orderRepo, ledgerService, transitionEventPublisher)
	orderHandler := ordergrpc.NewHandler(transitionOrderUC)

	if err := order.SetupAMQPInfrastructure(amqpChannel); err != nil {
		slog.Error("Failed to setup AMQP infrastructure", "error", err)
	}
	// TODO: need to write properly
	// consumer := order.NewConsumer(orderService, amqpChannel)
	// consumer.Listen()

	grpcServer := grpc.NewServer()
	pb.RegisterOrderServiceServer(grpcServer, orderHandler)

	slog.Info("order-service initialized successfully")
	return grpcServer
}
