package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/darkphotonKN/seeyoulatte-app/common/broker"
	commonconstants "github.com/darkphotonKN/seeyoulatte-app/common/constants"
	"github.com/darkphotonKN/seeyoulatte-app/common/discovery/consul"
	commonhelpers "github.com/darkphotonKN/seeyoulatte-app/common/utils"
	"github.com/darkphotonKN/seeyoulatte-app/services/api-gateway/config"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	logLevel := slog.LevelInfo
	if level := os.Getenv("LOG_LEVEL"); level == "debug" {
		logLevel = slog.LevelDebug
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// database
	db, err := config.NewDatabase(logger)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	// consul registry — used by gateway/* clients to discover downstream
	// services (auth-service, order-service). The gateway itself does NOT
	// register, since it's only invoked externally over HTTP.
	consulAddr := commonhelpers.GetEnvString("CONSUL_ADDR", "localhost:8521")
	registry, err := consul.NewRegistry(consulAddr, "api-gateway")
	if err != nil {
		logger.Error("failed to connect to Consul", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// rabbitmq channel — wired even if the gateway doesn't publish events yet,
	// so the auth.events and order.events exchanges are present before consumers
	// (gateway-side) come online.
	amqpUser := commonhelpers.GetEnvString("RABBITMQ_USER", "seeyoulatte")
	amqpPass := commonhelpers.GetEnvString("RABBITMQ_PASS", "seeyoulatte")
	amqpHost := commonhelpers.GetEnvString("RABBITMQ_HOST", "localhost")
	amqpPort := commonhelpers.GetEnvString("RABBITMQ_PORT", "5685")
	ch, closeCh := broker.Connect(amqpUser, amqpPass, amqpHost, amqpPort)
	broker.DeclareExchange(ch, commonconstants.AuthEventsExchange, "topic")
	broker.DeclareExchange(ch, commonconstants.OrderEventsExchange, "topic")
	defer func() {
		closeCh()
		ch.Close()
	}()

	router := config.SetupRoutes(db, logger, registry)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: router,
	}

	go func() {
		logger.Info("starting server", slog.String("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("failed to start server", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("server shutdown complete")
}
