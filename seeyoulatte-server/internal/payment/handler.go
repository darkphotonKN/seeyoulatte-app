package payment

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
)

// Service interface defines what the handler needs from the service
type Service interface {
	CreatePaymentIntent(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (*CreatePaymentIntentResponse, error)
	ProcessWebhookEvent(ctx context.Context, event *stripe.Event) error
	CreateStripeCustomer(ctx context.Context, userID uuid.UUID) (string, error)
}

type handler struct {
	service Service
	logger  *slog.Logger
}

func NewHandler(service Service, logger *slog.Logger) *handler {
	return &handler{
		service: service,
		logger:  logger,
	}
}

// CreatePaymentIntent handles POST /api/payments/intent
// Creates a payment intent for an order
func (h *handler) CreatePaymentIntent(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Convert user_id to UUID
	var userID uuid.UUID
	switch v := userIDValue.(type) {
	case string:
		parsedID, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
			return
		}
		userID = parsedID
	case uuid.UUID:
		userID = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID type"})
		return
	}

	// Parse request body
	var req struct {
		OrderID uuid.UUID `json:"order_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("creating payment intent",
		slog.String("user_id", userID.String()),
		slog.String("order_id", req.OrderID.String()))

	// Create payment intent
	response, err := h.service.CreatePaymentIntent(c.Request.Context(), userID, req.OrderID)
	if err != nil {
		h.logger.Error("failed to create payment intent",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
			slog.String("order_id", req.OrderID.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create payment intent"})
		return
	}

	c.JSON(http.StatusCreated, response)
}

// CreateStripeCustomer handles POST /api/payments/customer
// Creates a Stripe customer for a user (typically during first purchase)
func (h *handler) CreateStripeCustomer(c *gin.Context) {
	// Get user ID from context
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Convert user_id to UUID
	var userID uuid.UUID
	switch v := userIDValue.(type) {
	case string:
		parsedID, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
			return
		}
		userID = parsedID
	case uuid.UUID:
		userID = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user ID type"})
		return
	}

	h.logger.Info("creating stripe customer", slog.String("user_id", userID.String()))

	customerID, err := h.service.CreateStripeCustomer(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to create stripe customer",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create customer"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"customer_id": customerID})
}

// HandleStripeWebhook handles POST /api/stripe/webhook
// Processes webhook events from Stripe
func (h *handler) HandleStripeWebhook(c *gin.Context) {
	// Read raw bytes instead of using gin's ShouldBindJSON because:
	// 1. Stripe's webhook signature is calculated from the exact bytes sent
	// 2. ShouldBindJSON would parse/reformat the JSON, breaking signature verification
	// 3. This ensures the webhook is authentic and hasn't been tampered with
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("error reading webhook body", slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "error reading body"})
		return
	}

	// Get Stripe signature from headers
	signature := c.GetHeader("Stripe-Signature")
	if signature == "" {
		h.logger.Warn("missing stripe signature in webhook request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing signature"})
		return
	}

	// Get webhook secret from environment
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if webhookSecret == "" {
		h.logger.Error("STRIPE_WEBHOOK_SECRET not configured")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "webhook secret not configured"})
		return
	}

	// Verify webhook signature and construct event
	event, err := webhook.ConstructEvent(body, signature, webhookSecret)
	if err != nil {
		h.logger.Warn("invalid webhook signature",
			slog.String("error", err.Error()))
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature"})
		return
	}

	h.logger.Info("processing webhook event",
		slog.String("event_type", string(event.Type)),
		slog.String("event_id", event.ID))

	// Process the webhook event
	if err := h.service.ProcessWebhookEvent(c.Request.Context(), &event); err != nil {
		h.logger.Error("failed to process webhook event",
			slog.String("error", err.Error()),
			slog.String("event_type", string(event.Type)),
			slog.String("event_id", event.ID))
		// Return 200 to prevent Stripe from retrying
		// We log the error but don't want Stripe to keep sending the same event
		c.JSON(http.StatusOK, gin.H{"received": true, "error": fmt.Sprintf("failed to process: %v", err)})
		return
	}

	h.logger.Info("webhook event processed successfully",
		slog.String("event_type", string(event.Type)),
		slog.String("event_id", event.ID))

	c.JSON(http.StatusOK, gin.H{"received": true})
}