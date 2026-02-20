package config

import (
	"log/slog"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/darkphotonKN/seeyoulatte-app/internal/listing"
	"github.com/darkphotonKN/seeyoulatte-app/internal/middleware"
	"github.com/darkphotonKN/seeyoulatte-app/internal/order"
	"github.com/darkphotonKN/seeyoulatte-app/internal/user"
)

func SetupRoutes(db *sqlx.DB, logger *slog.Logger) *gin.Engine {
	// Set Gin mode based on environment
	if os.Getenv("ENVIRONMENT") == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(middleware.RequestID())
	router.Use(middleware.RequestLogger(logger))
	router.Use(middleware.StructuredLogger(logger))
	router.Use(corsMiddleware())

	// Initialize services
	// User/Auth service
	userRepo := user.NewRepository(db)
	userService := user.NewService(userRepo, logger)
	userHandler := user.NewHandler(userService, logger)

	// Listing service
	listingRepo := listing.NewRepository(db)
	listingService := listing.NewService(listingRepo, logger)
	listingHandler := listing.NewHandler(listingService, logger)

	// Order service
	orderRepo := order.NewRepository(db)
	orderService := order.NewService(orderRepo, logger)
	orderHandler := order.NewHandler(orderService, logger)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes
	api := router.Group("/api")
	{
		// Auth endpoints (public)
		auth := api.Group("/auth")
		{
			auth.POST("/signup", userHandler.SignUp)
			auth.POST("/signin", userHandler.SignIn)
			auth.POST("/google", userHandler.GoogleAuth)
			auth.GET("/me", middleware.AuthRequired(), userHandler.GetCurrentUser)
		}

		// Listing endpoints
		listings := api.Group("/listings")
		{
			// Public endpoints (no auth required)
			listings.GET("", listingHandler.GetAllListings)      // Get all public listings
			listings.GET("/:id", listingHandler.GetListing)      // Get single listing

			// Protected endpoints (auth required)
			listings.POST("", middleware.AuthRequired(), listingHandler.CreateListing)
			listings.GET("/my", middleware.AuthRequired(), listingHandler.GetMyListings)
			listings.PUT("/:id", middleware.AuthRequired(), listingHandler.UpdateListing)
			listings.DELETE("/:id", middleware.AuthRequired(), listingHandler.DeleteListing)
		}

		// Order endpoints
		orders := api.Group("/orders")
		{
			orders.POST("", orderHandler.CreateOrder)
			orders.GET("", orderHandler.GetAllOrders)
			orders.PUT("/:id", orderHandler.UpdateOrder)
			orders.DELETE("/:id", orderHandler.DeleteOrder)
		}
	}

	return router
}

func corsMiddleware() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Content-Type", "Authorization"}
	return cors.New(config)
}