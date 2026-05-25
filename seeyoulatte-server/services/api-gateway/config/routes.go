package config

import (
	"context"
	"log/slog"
	"os"

	commondiscovery "github.com/darkphotonKN/seeyoulatte-app/common/discovery"
	"github.com/darkphotonKN/seeyoulatte-app/services/api-gateway/internal/cache"
	authgw "github.com/darkphotonKN/seeyoulatte-app/services/api-gateway/internal/gateway/auth"
	ordergw "github.com/darkphotonKN/seeyoulatte-app/services/api-gateway/internal/gateway/order"
	"github.com/darkphotonKN/seeyoulatte-app/services/api-gateway/internal/listing"
	"github.com/darkphotonKN/seeyoulatte-app/services/api-gateway/internal/middleware"
	"github.com/darkphotonKN/seeyoulatte-app/services/api-gateway/internal/payment"
	paymentprocessor "github.com/darkphotonKN/seeyoulatte-app/services/api-gateway/internal/payment_processor"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// SetupRoutes wires every domain handler, mounts public + protected route
// groups. The registry is used by `gateway/*` clients to discover downstream
// services (auth-service, order-service).
//
// auth-service and order-service are remote — the gateway calls them via gRPC.
// JWT tokens are still validated locally by middleware.AuthRequired using the
// shared JWT_SECRET, so per-request auth doesn't pay a remote-hop cost.
func SetupRoutes(db *sqlx.DB, logger *slog.Logger, registry commondiscovery.Registry) *gin.Engine {
	if os.Getenv("ENVIRONMENT") == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(middleware.RequestLogger(logger))
	router.Use(middleware.StructuredLogger(logger))
	router.Use(corsMiddleware())

	// Optional Redis cache
	var cacheClient *cache.Client
	if os.Getenv("REDIS_HOST") != "" {
		cacheClient = cache.NewClient(logger)
		if err := cacheClient.Connect(context.Background()); err != nil {
			logger.Warn("Redis cache not available, continuing without cache",
				slog.String("error", err.Error()))
			cacheClient = nil
		} else {
			logger.Info("Redis cache connected successfully")
		}
	} else {
		logger.Info("Redis not configured, running without cache")
	}

	// --- gateway clients (downstream services) ---
	authClient := authgw.NewClient(registry)
	authHandler := authgw.NewHandler(authClient)

	// --- listing stays in the gateway monolith for now ---
	listingRepo := listing.NewRepository(db)
	listingService := listing.NewService(listingRepo, logger)
	listingHandler := listing.NewHandler(listingService, logger)

	// --- order client needs listing-service to resolve listing→seller info
	// since listing hasn't been extracted yet. See gateway/order/client.go. ---
	orderClient := ordergw.NewClient(registry, listingService)
	orderHandler := ordergw.NewHandler(orderClient)

	// --- payment stays local; talks to auth/order via the gateway clients ---
	stripeProcessor := paymentprocessor.NewStripeProcessor()
	paymentService := payment.NewService(logger, stripeProcessor, orderClient, authClient, db, cacheClient)
	paymentHandler := payment.NewHandler(paymentService, logger)

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := router.Group("/api")
	{
		// Auth — proxied to auth-service via gRPC
		auth := api.Group("/auth")
		{
			auth.POST("/signup", authHandler.SignUp)
			auth.POST("/signin", authHandler.SignIn)
			auth.POST("/google", authHandler.GoogleAuth)
			auth.GET("/me", middleware.AuthRequired(), authHandler.GetCurrentUser)
		}

		// Listing — still local
		listings := api.Group("/listings")
		{
			listings.GET("", listingHandler.GetAllListings)
			listings.GET("/:id", listingHandler.GetListing)
			listings.POST("", middleware.AuthRequired(), listingHandler.CreateListing)
			listings.GET("/my", middleware.AuthRequired(), listingHandler.GetMyListings)
			listings.PUT("/:id", middleware.AuthRequired(), listingHandler.UpdateListing)
			listings.DELETE("/:id", middleware.AuthRequired(), listingHandler.DeleteListing)
		}

		// Orders — proxied to order-service via gRPC
		orders := api.Group("/orders")
		orders.Use(middleware.AuthRequired())
		{
			orders.POST("", orderHandler.CreateOrder)
			orders.GET("", orderHandler.GetAllOrders)
			orders.PUT("/:id", orderHandler.UpdateOrder)
			orders.DELETE("/:id", orderHandler.DeleteOrder)
		}

		// Payments — local; uses gateway clients to fetch order/user data
		payments := api.Group("/payments")
		{
			payments.POST("/intent", middleware.AuthRequired(), paymentHandler.CreatePaymentIntent)
			payments.POST("/customer", middleware.AuthRequired(), paymentHandler.CreateStripeCustomer)
		}

		// Stripe webhook (no auth — signature-verified)
		api.POST("/stripe/webhook", paymentHandler.HandleStripeWebhook)
	}

	return router
}

func corsMiddleware() gin.HandlerFunc {
	c := cors.DefaultConfig()
	c.AllowOrigins = []string{"*"}
	c.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	c.AllowHeaders = []string{"Content-Type", "Authorization"}
	return cors.New(c)
}
