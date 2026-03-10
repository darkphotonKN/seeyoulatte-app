package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	// TODO: Uncomment when order service integration is ready
	"github.com/darkphotonKN/seeyoulatte-app/internal/interfaces"
	"github.com/darkphotonKN/seeyoulatte-app/internal/order"
	paymentprocessor "github.com/darkphotonKN/seeyoulatte-app/internal/payment_processor"
	"github.com/darkphotonKN/seeyoulatte-app/internal/user"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stripe/stripe-go/v82"
)

type service struct {
	logger           *slog.Logger
	paymentProcessor PaymentProcessor
	// TODO: Add orderService when order service implements required methods
	orderService OrderService
	userService  UserService
	db           *sqlx.DB
	cacheClient  interfaces.Cache
}

// PaymentProcessor interface for Stripe operations
type PaymentProcessor interface {
	CreateCustomer(ctx context.Context, req *paymentprocessor.CreateCustomerRequest) (*paymentprocessor.CreateCustomerResponse, error)
	CreatePaymentIntent(ctx context.Context, amount int64, customerID string, metadata map[string]string) (*paymentprocessor.CreatePaymentIntentResponse, error)
	ProcessWebhookEvent(ctx context.Context, webhookEvent *paymentprocessor.WebhookEvent) (customerId string, error error)
	FetchCurrentState(ctx context.Context, customerId string) (*paymentprocessor.CustomerState, error)
}

// OrderService interface - what we need from order service
type OrderService interface {
	GetByID(ctx context.Context, orderID uuid.UUID) (*order.Order, error)
	GetPendingPaymentOrdersByUser(ctx context.Context, userID uuid.UUID) ([]*order.Order, error)
	// TODO: Implement TransitionState in order service for state machine transitions
	// TransitionState(ctx context.Context, orderID uuid.UUID, event string, actor string) error
}

// UserService interface. what we need from user service
type UserService interface {
	GetByID(ctx context.Context, userID uuid.UUID) (*user.User, error)
	GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*user.User, error)
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
	// Get order details
	order, err := s.orderService.GetByID(ctx, orderID)
	if err != nil {
		s.logger.Error("failed to get order",
			slog.String("error", err.Error()),
			slog.String("order_id", orderID.String()))
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Verify the order belongs to this user
	if order.BuyerID != userID {
		s.logger.Warn("user attempting to pay for order they don't own",
			slog.String("user_id", userID.String()),
			slog.String("order_id", orderID.String()),
			slog.String("actual_buyer_id", order.BuyerID.String()))
		return nil, fmt.Errorf("order does not belong to this user")
	}

	// Verify order is in pending_payment state
	if order.State != "pending_payment" {
		s.logger.Warn("attempting to create payment for order not in pending_payment state",
			slog.String("order_id", orderID.String()),
			slog.String("current_state", order.State))
		return nil, fmt.Errorf("order is not in pending payment state")
	}

	// Get user to check for payment processor customer ID
	user, err := s.userService.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()))
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Create payment processor customer if doesn't exist
	var stripeCustomerID string
	if user.StripeCustomerID == nil || *user.StripeCustomerID == "" {
		s.logger.Info("creating new Stripe customer",
			slog.String("user_id", userID.String()),
			slog.String("email", user.Email))

		customerReq := &paymentprocessor.CreateCustomerRequest{
			UserId: userID,
			Email:  user.Email,
		}

		customerResp, err := s.paymentProcessor.CreateCustomer(ctx, customerReq)
		if err != nil {
			s.logger.Error("failed to create Stripe customer",
				slog.String("error", err.Error()),
				slog.String("user_id", userID.String()))
			return nil, fmt.Errorf("failed to create Stripe customer: %w", err)
		}
		stripeCustomerID = customerResp.CustomerID

		// 	// Update user with payment processor customer ID
		if err := s.userService.UpdateStripeCustomerID(ctx, userID, stripeCustomerID); err != nil {
			s.logger.Error("failed to update user with Stripe customer ID",
				slog.String("error", err.Error()),
				slog.String("user_id", userID.String()),
				slog.String("stripe_customer_id", stripeCustomerID))
			// Continue anyway - payment can still proceed
		}
	} else {
		stripeCustomerID = *user.StripeCustomerID
	}

	// Convert amount to cents (Stripe requires amounts in smallest currency unit)
	amountInCents := int64(order.Amount * 100)

	// Create metadata for the payment intent
	metadata := map[string]string{
		"order_id":  orderID.String(),
		"buyer_id":  userID.String(),
		"seller_id": order.SellerID.String(),
	}

	s.logger.Info("creating payment intent",
		slog.String("order_id", orderID.String()),
		slog.Int64("amount_cents", amountInCents),
		slog.String("customer_id", stripeCustomerID))

	// Create payment intent with Stripe
	stripeResponse, err := s.paymentProcessor.CreatePaymentIntent(ctx, amountInCents, stripeCustomerID, metadata)
	if err != nil {
		s.logger.Error("failed to create Stripe payment intent",
			slog.String("error", err.Error()),
			slog.String("order_id", orderID.String()))
		return nil, fmt.Errorf("failed to create payment intent: %w", err)
	}

	// TODO: Store payment intent ID in database for tracking

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
	// Get user details
	user, err := s.userService.GetByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()))
		return "", fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user already has a Stripe customer ID
	if user.StripeCustomerID != nil && *user.StripeCustomerID != "" {
		s.logger.Info("user already has Stripe customer ID",
			slog.String("user_id", userID.String()),
			slog.String("stripe_customer_id", *user.StripeCustomerID))
		return *user.StripeCustomerID, nil
	}

	// Create Stripe customer
	customerReq := paymentprocessor.CreateCustomerRequest{
		UserId: userID,
		Email:  user.Email,
	}
	customerResp, err := s.paymentProcessor.CreateCustomer(ctx, &customerReq)
	if err != nil {
		s.logger.Error("failed to create Stripe customer",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()))
		return "", fmt.Errorf("failed to create Stripe customer: %w", err)
	}

	// Update user with Stripe customer ID
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

	// Handle different event types
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
		return nil // Return nil for unhandled events to prevent retries
	}
}

// handlePaymentIntentSucceeded handles successful payment events
func (s *service) handlePaymentIntentSucceeded(ctx context.Context, event *stripe.Event) error {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		s.logger.Error("failed to unmarshal payment intent",
			slog.String("error", err.Error()),
			slog.String("event_id", event.ID))
		return fmt.Errorf("failed to unmarshal payment intent: %w", err)
	}

	// Extract order ID from metadata
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

	// TODO: Transition order to PAID state when TransitionState is implemented
	// This will trigger the creation of ESCROW ledger entry in the order service
	// if err := s.orderService.TransitionState(ctx, orderID, "payment_confirmed", "system"); err != nil {
	// 	s.logger.Error("failed to transition order to paid state",
	// 		slog.String("error", err.Error()),
	// 		slog.String("order_id", orderID.String()))
	// 	return fmt.Errorf("failed to transition order state: %w", err)
	// }
	s.logger.Warn("TODO: TransitionState not yet implemented in order service",
		slog.String("order_id", orderID.String()))

	// TODO: Store payment details in payments table

	return nil
}

// handlePaymentIntentFailed handles failed payment events
func (s *service) handlePaymentIntentFailed(ctx context.Context, event *stripe.Event) error {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		s.logger.Error("failed to unmarshal payment intent",
			slog.String("error", err.Error()),
			slog.String("event_id", event.ID))
		return fmt.Errorf("failed to unmarshal payment intent: %w", err)
	}

	// Extract order ID from metadata
	orderIDStr, ok := paymentIntent.Metadata["order_id"]
	if !ok {
		s.logger.Warn("order_id not found in failed payment intent metadata",
			slog.String("payment_intent_id", paymentIntent.ID))
		return nil // Don't error for missing metadata on failed payments
	}

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		s.logger.Error("invalid order_id in metadata",
			slog.String("order_id", orderIDStr),
			slog.String("error", err.Error()))
		return fmt.Errorf("invalid order_id: %w", err)
	}

	s.logger.Info("payment failed for order",
		slog.String("order_id", orderID.String()),
		slog.String("payment_intent_id", paymentIntent.ID))

	// Order remains in pending_payment state, buyer can retry
	// TODO: Store failure reason in payments table

	return nil
}

// handlePaymentIntentCanceled handles canceled payment events
func (s *service) handlePaymentIntentCanceled(ctx context.Context, event *stripe.Event) error {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		s.logger.Error("failed to unmarshal payment intent",
			slog.String("error", err.Error()),
			slog.String("event_id", event.ID))
		return fmt.Errorf("failed to unmarshal payment intent: %w", err)
	}

	// Extract order ID from metadata
	orderIDStr, ok := paymentIntent.Metadata["order_id"]
	if !ok {
		s.logger.Warn("order_id not found in canceled payment intent metadata",
			slog.String("payment_intent_id", paymentIntent.ID))
		return nil // Don't error for missing metadata on canceled payments
	}

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		s.logger.Error("invalid order_id in metadata",
			slog.String("order_id", orderIDStr),
			slog.String("error", err.Error()))
		return fmt.Errorf("invalid order_id: %w", err)
	}

	s.logger.Info("payment canceled for order",
		slog.String("order_id", orderID.String()),
		slog.String("payment_intent_id", paymentIntent.ID))

	// Order remains in pending_payment state, buyer can retry

	return nil
}

/*
* Primary method for syncing up payment processor-related states and avoiding a split-brain problem.
*
* Since payment table's status comes from the processor, we need to at least get that validated from the processors's more
* consistent apis, as opposed to their webhooks, to store in the KV for easy access but at the same time update the database once we have it validated.
*
* The key-value cache structure will be as follows:

Key: paymentprocessor:customer:{customerId}

	Value: {
	  subscriptionId: "sub_xyz",
	  status: "active",           // Core validation field
	  priceId: "price_abc",       // Feature access control
	  currentPeriodEnd: 1234567,  // Billing cycle info
	  cancelAtPeriodEnd: false    // Immediate cancellation status
	}

Key: paymentprocessor:user:{userId}
Value: "cus_stripe123"        // Customer ID mapping

* Database storage will store the same things but into the respective tables.
*/
func (s *service) SyncPaymentProcessorDataToStorage(ctx context.Context, customerId string) error {
	// get payments and customer data synced from payment processor's
	// sync fetch request
	currentState, err := s.paymentProcessor.FetchCurrentState(ctx, customerId)

	if err != nil {
		s.logger.Error("Couldn't fetch for current state with payment processor helper FetchCurrentState",
			"error", err,
		)
		return err
	}

	slog.Debug("current state retrieved from payment processor",
		"currentState", currentState)

	userId, err := s.GetCachedUserIdByCustomerId(ctx, customerId)

	if err != nil {
		// exception
		s.logger.Warn("exception error when getting user id from customer id from cache",
			"customer_id", customerId)

		// io from database only if cache doesnt exist
		// TODO: get userId from database via customer id, or pending orders directly with customerId
		userId, err = s.userService.GetUserIDByStripeCustomerID(ctx, customerId)
		if err != nil {
			s.logger.Error("Couldn't get userId from database with customerId after cache already failed.",
				"customer_id", customerId,
			)
			return err
		}

		// update cache with the customerId to userId mapping
		s.AddCacheCustomerIdToUserId(ctx, customerId, userId)
	}

	pendingOrders, err := s.orderService.GetPendingPaymentOrdersByUser(ctx, userId)

	// store update in database
	processorPaymentsMap := make(map[string]*paymentprocessor.PaymentState)
	for _, payment := range currentState.Payments {
		processorPaymentsMap[payment.OrderID] = payment
	}

	for _, order := range pendingOrders {
		// cross reference which orders still pending if they have been updated
		// on stripe for this user
		matchingOrder := processorPaymentsMap[order.ID.String()]
		if matchingOrder != nil {
			// TODO: use state machine to update state
		}
	}

	return nil
}
