package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	authgw "github.com/darkphotonKN/seeyoulatte-app/services/api-gateway/internal/gateway/auth"
	ordergw "github.com/darkphotonKN/seeyoulatte-app/services/api-gateway/internal/gateway/order"
	"github.com/darkphotonKN/seeyoulatte-app/services/api-gateway/internal/interfaces"
	paymentprocessor "github.com/darkphotonKN/seeyoulatte-app/services/api-gateway/internal/payment_processor"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stripe/stripe-go/v82"
)

type service struct {
	logger           *slog.Logger
	paymentProcessor PaymentProcessor
	orderService     OrderService
	userService      UserService
	db               *sqlx.DB
	cacheClient      interfaces.Cache
}

// PaymentProcessor interface for Stripe operations
type PaymentProcessor interface {
	CreateCustomer(ctx context.Context, req *paymentprocessor.CreateCustomerRequest) (*paymentprocessor.CreateCustomerResponse, error)
	CreatePaymentIntent(ctx context.Context, amount int64, customerID string, metadata map[string]string) (*paymentprocessor.CreatePaymentIntentResponse, error)
	ProcessWebhookEvent(ctx context.Context, webhookEvent *paymentprocessor.WebhookEvent) (customerId string, error error)
	FetchCurrentState(ctx context.Context, customerId string) (*paymentprocessor.CustomerState, error)
}

// OrderService — what payment needs from the order-service gRPC client.
type OrderService interface {
	GetByID(ctx context.Context, orderID uuid.UUID) (*ordergw.Order, error)
	GetPendingPaymentOrdersByUser(ctx context.Context, userID uuid.UUID) ([]*ordergw.Order, error)
	TransitionState(ctx context.Context, orderID uuid.UUID, targetState, actor string) error
}

// UserService — what payment needs from the auth-service gRPC client.
type UserService interface {
	GetByID(ctx context.Context, userID uuid.UUID) (*authgw.User, error)
	GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*authgw.User, error)
	GetUserIDByStripeCustomerID(ctx context.Context, stripeCustomerID string) (uuid.UUID, error)
	UpdateStripeCustomerID(ctx context.Context, userID uuid.UUID, stripeCustomerID string) error
}

func NewService(logger *slog.Logger, paymentProcessor PaymentProcessor, orderService OrderService, userService UserService, db *sqlx.DB, cacheClient interfaces.Cache) *service {
	return &service{
		logger:           logger,
		paymentProcessor: paymentProcessor,
		orderService:     orderService,
		userService:      userService,
		db:               db,
		cacheClient:      cacheClient,
	}
}

// CreatePaymentIntent creates a payment intent for an order
func (s *service) CreatePaymentIntent(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (*CreatePaymentIntentResponse, error) {
	order, err := s.orderService.GetByID(ctx, orderID)
	if err != nil {
		s.logger.Error("failed to get order",
			slog.String("error", err.Error()),
			slog.String("order_id", orderID.String()))
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order.BuyerID != userID {
		s.logger.Warn("user attempting to pay for order they don't own",
			slog.String("user_id", userID.String()),
			slog.String("order_id", orderID.String()),
			slog.String("actual_buyer_id", order.BuyerID.String()))
		return nil, fmt.Errorf("order does not belong to this user")
	}

	if order.State != "pending_payment" {
		s.logger.Warn("attempting to create payment for order not in pending_payment state",
			slog.String("order_id", orderID.String()),
			slog.String("current_state", order.State))
		return nil, fmt.Errorf("order is not in pending payment state")
	}

	user, err := s.userService.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	var stripeCustomerID string
	if user.StripeCustomerID == nil || *user.StripeCustomerID == "" {
		s.logger.Info("creating new Stripe customer",
			slog.String("user_id", userID.String()),
			slog.String("email", user.Email))

		customerResp, err := s.paymentProcessor.CreateCustomer(ctx, &paymentprocessor.CreateCustomerRequest{
			UserId: userID,
			Email:  user.Email,
		})
		if err != nil {
			s.logger.Error("failed to create Stripe customer",
				slog.String("error", err.Error()),
				slog.String("user_id", userID.String()))
			return nil, fmt.Errorf("failed to create Stripe customer: %w", err)
		}
		stripeCustomerID = customerResp.CustomerID

		if err := s.userService.UpdateStripeCustomerID(ctx, userID, stripeCustomerID); err != nil {
			s.logger.Error("failed to update user with Stripe customer ID",
				slog.String("error", err.Error()),
				slog.String("user_id", userID.String()),
				slog.String("stripe_customer_id", stripeCustomerID))
			// Continue — payment can still proceed
		}
	} else {
		stripeCustomerID = *user.StripeCustomerID
	}

	amountInCents := int64(order.Amount * 100)

	metadata := map[string]string{
		"order_id":  orderID.String(),
		"buyer_id":  userID.String(),
		"seller_id": order.SellerID.String(),
	}

	s.logger.Info("creating payment intent",
		slog.String("order_id", orderID.String()),
		slog.Int64("amount_cents", amountInCents),
		slog.String("customer_id", stripeCustomerID))

	stripeResponse, err := s.paymentProcessor.CreatePaymentIntent(ctx, amountInCents, stripeCustomerID, metadata)
	if err != nil {
		s.logger.Error("failed to create Stripe payment intent",
			slog.String("error", err.Error()),
			slog.String("order_id", orderID.String()))
		return nil, fmt.Errorf("failed to create payment intent: %w", err)
	}

	return &CreatePaymentIntentResponse{
		PaymentIntentID: stripeResponse.PaymentIntentID,
		ClientSecret:    stripeResponse.ClientSecret,
		Amount:          amountInCents,
		Currency:        "usd",
		OrderID:         orderID.String(),
		Status:          "requires_payment_method",
	}, nil
}

// CreateStripeCustomer creates a Stripe customer for a user
func (s *service) CreateStripeCustomer(ctx context.Context, userID uuid.UUID) (string, error) {
	user, err := s.userService.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()))
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	if user.StripeCustomerID != nil && *user.StripeCustomerID != "" {
		s.logger.Info("user already has Stripe customer ID",
			slog.String("user_id", userID.String()),
			slog.String("stripe_customer_id", *user.StripeCustomerID))
		return *user.StripeCustomerID, nil
	}

	customerResp, err := s.paymentProcessor.CreateCustomer(ctx, &paymentprocessor.CreateCustomerRequest{
		UserId: userID,
		Email:  user.Email,
	})
	if err != nil {
		s.logger.Error("failed to create Stripe customer",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()))
		return "", fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	if err := s.userService.UpdateStripeCustomerID(ctx, userID, customerResp.CustomerID); err != nil {
		s.logger.Error("failed to update user with Stripe customer ID",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()),
			slog.String("stripe_customer_id", customerResp.CustomerID))
		return "", fmt.Errorf("failed to update user: %w", err)
	}

	return customerResp.CustomerID, nil
}

// ProcessWebhookEvent processes Stripe webhook events
func (s *service) ProcessWebhookEvent(ctx context.Context, event *stripe.Event) error {
	s.logger.Info("processing webhook event",
		slog.String("event_type", string(event.Type)),
		slog.String("event_id", event.ID))

	switch event.Type {
	case stripe.EventTypePaymentIntentSucceeded:
		return s.handlePaymentIntentSucceeded(ctx, event)
	case stripe.EventTypePaymentIntentPaymentFailed:
		return s.handlePaymentIntentFailed(ctx, event)
	case stripe.EventTypePaymentIntentCanceled:
		return s.handlePaymentIntentCanceled(ctx, event)
	default:
		s.logger.Info("unhandled webhook event type",
			slog.String("event_type", string(event.Type)))
		return nil
	}
}

func (s *service) handlePaymentIntentSucceeded(ctx context.Context, event *stripe.Event) error {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		s.logger.Error("failed to unmarshal payment intent",
			slog.String("error", err.Error()),
			slog.String("event_id", event.ID))
		return fmt.Errorf("failed to unmarshal payment intent: %w", err)
	}

	orderIDStr, ok := paymentIntent.Metadata["order_id"]
	if !ok {
		s.logger.Error("order_id not found in payment intent metadata",
			slog.String("payment_intent_id", paymentIntent.ID))
		return fmt.Errorf("order_id not found in metadata")
	}

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		s.logger.Error("invalid order_id in metadata",
			slog.String("order_id", orderIDStr),
			slog.String("error", err.Error()))
		return fmt.Errorf("invalid order_id: %w", err)
	}

	s.logger.Info("payment succeeded, transitioning order to paid",
		slog.String("order_id", orderID.String()),
		slog.String("payment_intent_id", paymentIntent.ID),
		slog.Int64("amount", paymentIntent.Amount))

	if err := s.orderService.TransitionState(ctx, orderID, "paid", "system"); err != nil {
		s.logger.Error("failed to transition order to paid state",
			slog.String("error", err.Error()),
			slog.String("order_id", orderID.String()))
		return fmt.Errorf("failed to transition order state: %w", err)
	}

	return nil
}

func (s *service) handlePaymentIntentFailed(_ context.Context, event *stripe.Event) error {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		return fmt.Errorf("failed to unmarshal payment intent: %w", err)
	}
	if orderIDStr, ok := paymentIntent.Metadata["order_id"]; ok {
		s.logger.Info("payment failed for order",
			slog.String("order_id", orderIDStr),
			slog.String("payment_intent_id", paymentIntent.ID))
	}
	return nil
}

func (s *service) handlePaymentIntentCanceled(_ context.Context, event *stripe.Event) error {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		return fmt.Errorf("failed to unmarshal payment intent: %w", err)
	}
	if orderIDStr, ok := paymentIntent.Metadata["order_id"]; ok {
		s.logger.Info("payment canceled for order",
			slog.String("order_id", orderIDStr),
			slog.String("payment_intent_id", paymentIntent.ID))
	}
	return nil
}

/*
SyncPaymentProcessorDataToStorage validates and syncs payment-processor state
with our database. See cache_helpers.go for the cache flow.
*/
func (s *service) SyncPaymentProcessorDataToStorage(ctx context.Context, customerId string) error {
	currentState, err := s.paymentProcessor.FetchCurrentState(ctx, customerId)
	if err != nil {
		s.logger.Error("Couldn't fetch current state from payment processor", "error", err)
		return err
	}

	s.logger.Debug("current state retrieved from payment processor", "currentState", currentState)

	userId, err := s.GetCachedUserIdByCustomerId(ctx, customerId)
	if err != nil {
		s.logger.Warn("cache miss for customerId→userId, falling back to auth-service",
			"customer_id", customerId)
		userId, err = s.userService.GetUserIDByStripeCustomerID(ctx, customerId)
		if err != nil {
			s.logger.Error("auth-service lookup failed for customerId",
				"customer_id", customerId)
			return err
		}
		s.AddCacheCustomerIdToUserId(ctx, customerId, userId)
	}

	pendingOrders, err := s.orderService.GetPendingPaymentOrdersByUser(ctx, userId)
	if err != nil {
		return err
	}

	processorPaymentsMap := make(map[string]*paymentprocessor.PaymentState)
	for _, payment := range currentState.Payments {
		processorPaymentsMap[payment.OrderID] = payment
	}

	for _, order := range pendingOrders {
		matchingOrder := processorPaymentsMap[order.ID.String()]
		if matchingOrder != nil {
			// TODO: use state machine to update state
			_ = matchingOrder
		}
	}

	return nil
}
