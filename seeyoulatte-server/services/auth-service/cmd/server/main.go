package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"time"

	"github.com/darkphotonKN/seeyoulatte-app/common/broker"
	commonconstants "github.com/darkphotonKN/seeyoulatte-app/common/constants"
	"github.com/darkphotonKN/seeyoulatte-app/common/discovery"
	"github.com/darkphotonKN/seeyoulatte-app/common/discovery/consul"
	commontelemetry "github.com/darkphotonKN/seeyoulatte-app/common/telemetry"
	commonhelpers "github.com/darkphotonKN/seeyoulatte-app/common/utils"
	"github.com/darkphotonKN/seeyoulatte-app/services/auth-service/config"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
)

var (
	environment       = commonhelpers.GetEnvString("ENVIRONMENT", "development")
	collectorEndpoint = commonhelpers.GetEnvString("COLLECTOR_ENDPOINT", "localhost:4317")

	serviceName    = "auth"
	grpcAddr       = commonhelpers.GetEnvString("GRPC_AUTH_ADDR", "7301")
	consulAddr     = commonhelpers.GetEnvString("CONSUL_ADDR", "localhost:8521")
	serviceVersion = commonhelpers.GetEnvString("SERVICE_VERSION", "1.0.0")
	otelEnabled    = commonhelpers.GetEnvString("OTEL_ENABLED", "false") == "true"

	amqpUser     = commonhelpers.GetEnvString("RABBITMQ_USER", "seeyoulatte")
	amqpPassword = commonhelpers.GetEnvString("RABBITMQ_PASS", "seeyoulatte")
	amqpHost     = commonhelpers.GetEnvString("RABBITMQ_HOST", "localhost")
	amqpPort     = commonhelpers.GetEnvString("RABBITMQ_PORT", "5685")
)

func main() {
	commonhelpers.SetupLogger(environment)

	db := config.InitDB()
	defer db.Close()

	registry, err := consul.NewRegistry(consulAddr, serviceName)
	if err != nil {
		log.Fatal("Failed to create Consul registry")
	}

	ctx := context.Background()

	shutdown, err := commontelemetry.Init(ctx, commontelemetry.Config{
		ServiceName:       serviceName,
		ServiceVersion:    serviceVersion,
		Environment:       environment,
		CollectorEndpoint: collectorEndpoint,
		Enabled:           otelEnabled,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer shutdown(ctx)

	instanceID := discovery.GenerateInstanceID(serviceName)

	if err := registry.Register(ctx, instanceID, serviceName, "localhost:"+grpcAddr); err != nil {
		log.Printf("\nError when registering service:\n\n%s\n\n", err)
		panic(err)
	}

	go func() {
		for {
			if err := registry.HealthCheck(instanceID, serviceName); err != nil {
				log.Fatal("Health check failed.")
			}
			time.Sleep(time.Second * 1)
		}
	}()
	defer registry.Deregister(ctx, instanceID, serviceName)

	listener, err := net.Listen("tcp", "localhost:"+grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen at port %s: %v", grpcAddr, err)
	}
	defer listener.Close()

	ch, closeCh := broker.Connect(amqpUser, amqpPassword, amqpHost, amqpPort)
	broker.DeclareExchange(ch, commonconstants.AuthEventsExchange, "topic")
	defer func() {
		closeCh()
		ch.Close()
	}()

	grpcServer := config.SetupServices(db, ch, registry)

	slog.Info("auth-service gRPC server starting", "port", grpcAddr)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatal("Can't connect to grpc server. Error:", err.Error())
	}
}
