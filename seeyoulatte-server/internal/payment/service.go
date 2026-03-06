package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	// TODO: Uncomment when order service integration is ready
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
}

// PaymentProcessor interface for Stripe operations
type PaymentProcessor interface {
	CreateCustomer(ctx context.Context, req *paymentprocessor.CreateCustomerRequest) (*paymentprocessor.CreateCustomerResponse, error)
	CreatePaymentIntent(ctx context.Context, amount int64, customerID string, metadata map[string]string) (*paymentprocessor.CreatePaymentIntentResponse, error)
	ProcessWebhookEvent(ctx context.Context, webhookEvent *paymentprocessor.WebhookEvent) (customerId string, error error)
}

// OrderService interface - what we need from order service
type OrderService interface {
	GetByID(ctx context.Context, orderID uuid.UUID) (*order.Order, error)
	// TODO: Implement TransitionState in order service for state machine transitions
	// TransitionState(ctx context.Context, orderID uuid.UUID, event string, actor string) error
}

// UserService interface. what we need from user service
type UserService interface {
	GetByID(ctx context.Context, userID uuid.UUID) (*user.User, error)
	UpdateStripeCustomerID(ctx context.Context, userID uuid.UUID, stripeCustomerID string) error
}

func NewService(logger *slog.Logger, paymentProcessor PaymentProcessor, orderService OrderService, userService UserService, db *sqlx.DB) *service {
	return &service{
		logger:           logger,
		paymentProcessor: paymentProcessor,
		orderService:     orderService,
		userService:      userService,
		db:               db,
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
func (s *service) SyncStripeDataToStorage(ctx context.Context, customerId string) error {
	// get latest up-to-date data from the payment processor
	customer, err := customer.Get(customerId, nil)

	// --- Data Organization ---

	if err != nil {
		fmt.Printf("\nFailed to get customer from stripe: %+v\n\n", err)
		return fmt.Errorf("failed to get customer from stripe: %w", err)
	}

	stripeUpdateCusWithCusIdKey := s.cacheClient.GetCustomerDataFromCustomerIdKey(customerId)

	// -- subscriptions --

	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerId),
		Status:   stripe.String("all"),
	}

	// expand the payment method to get card details
	params.AddExpand("data.default_payment_method")

	// get subscription data
	subIter := subscription.List(params)

	// subscriptions slice
	subscriptions := []*stripe.Subscription{}

	// validates that customer has subscriptions
	for subIter.Next() {
		// get the subscription
		sub := subIter.Subscription()

		// Handle iteration error
		if err := subIter.Err(); err != nil {
			fmt.Printf("\nFailed to fetch subscriptions from Stripe: %+v\n\n", err)
			return fmt.Errorf("failed to fetch subscriptions from Stripe: %w", err)
		}

		subscriptions = append(subscriptions, sub)
	}

	// -- payments --

	payments := []*stripe.PaymentIntent{}

	paymentParams := &stripe.PaymentIntentListParams{
		Customer: stripe.String(customerId),
	}

	// include payment method details
	paymentParams.AddExpand("data.payment_method")

	paymentIter := paymentintent.List(paymentParams)

	for paymentIter.Next() {
		pi := paymentIter.PaymentIntent()
		fmt.Printf("\npayment intent: %+v\n\n", pi)

		payments = append(payments, pi)
	}

	// --- DB Storage ---
	// we do this first and roll back before even updating cache in case of error

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		fmt.Printf("\nError when attempting to start transaction: %+v\n\n", err)
		return fmt.Errorf("error when attempting to start transaction: %+v", err)
	}

	// NOTE: safe to run even if commit was successful - in that case it will be a no-op
	defer tx.Rollback()

	// update application database for the respective tables

	// -- user

	// get userId from cache / db depending on availability
	userId, err := s.GetCachedUserIdByCustomerId(ctx, customerId)

	if err != nil {
		fmt.Printf("\nUnexpected error when trying to get userId from cache: %s\n\n", err)
		return err
	}

	// if not availble, use direct db query
	if err == redislib.Nil {
		user, err := s.userService.GetByStripeCustomerID(ctx, customerId)

		if err != nil {
			fmt.Printf("unexpected error when trying to query for userId: %s\n", err)
			return err
		}

		// update userId with this version
		userId = user.ID
	}

	// -- subscription --

	for _, sub := range subscriptions {
		err := s.repo.UpsertSubscriptionRecord(ctx, &Subscription{
			UserID:               userId,
			Status:               string(sub.Status),
			StripeSubscriptionID: sub.ID,
			StripeCustomerID:     customerId,
		})

		if err != nil {
			fmt.Printf("\nError when attempting to batch upsert subscriptions during sync: %+v\n\n", err)
			return fmt.Errorf("error when attempting to batch upsert subscriptions during sync: %+v", err)
		}
	}

	// -- payments --

	for _, payment := range payments {
		err := s.repo.UpsertPayment(ctx, payment.ID, &Payment{
			UserID:           userId,
			StripeCustomerID: customerId,
			StripeIntentID:   payment.ID,
			Amount:           payment.Amount,
			Status:           string(payment.Status),
			Currency:         string(payment.Currency),
		})

		if err != nil {
			fmt.Printf("\nError when attempting to upsert payment during sync: %+v\n\n", err)
			return err
		}
	}

	// --- Caching ---

	subCache := make([]*StripeSubscriptionCache, len(subscriptions))

	// -- subscriptions --

	for index, sub := range subscriptions {
		// extract payment method info safely
		var pmInfo *PaymentMethodInfo

		if len(subscriptions) > 0 {
			sub := subscriptions[0]

			if sub.DefaultPaymentMethod != nil && sub.DefaultPaymentMethod.Card != nil {
				pmInfo = &PaymentMethodInfo{
					Brand: string(sub.DefaultPaymentMethod.Card.Brand),
					Last4: sub.DefaultPaymentMethod.Card.Last4,
				}
			}

		}

		// add to cache slice
		subCache[index] = &StripeSubscriptionCache{
			SubscriptionID:    sub.ID,
			Status:            string(sub.Status),
			PriceID:           sub.Items.Data[0].Price.ID,
			CancelAtPeriodEnd: sub.CancelAtPeriodEnd,
			PaymentMethod:     pmInfo,
		}
	}

	// -- payments --

	paymentCache := make([]*StripePaymentsCache, len(payments))

	for index, payment := range payments {
		paymentCache[index] = &StripePaymentsCache{
			ID:     payment.ID,
			Status: string(payment.Status),
		}
	}

	// -- user --

	stripeCusData := StripeCustomerDataRes{
		ID:                   customer.ID,
		Address:              convertAddress(customer.Address),
		Balance:              customer.Balance,
		CashBalance:          convertCashBalance(customer.CashBalance),
		Created:              customer.Created,
		Currency:             string(customer.Currency),
		DefaultSource:        convertDefaultSource(customer.DefaultSource),
		Deleted:              customer.Deleted,
		Delinquent:           customer.Delinquent,
		Description:          customer.Description,
		Discount:             convertDiscount(customer.Discount),
		Email:                customer.Email,
		InvoiceCreditBalance: customer.InvoiceCreditBalance,
		InvoicePrefix:        customer.InvoicePrefix,
		InvoiceSettings:      convertInvoiceSettings(customer.InvoiceSettings),
		Livemode:             customer.Livemode,
		Metadata:             customer.Metadata,
		Name:                 customer.Name,
		NextInvoiceSequence:  customer.NextInvoiceSequence,
		Object:               customer.Object,
		Phone:                customer.Phone,
		PreferredLocales:     customer.PreferredLocales,
		Subscriptions:        convertSubscriptions(customer.Subscriptions),
		Tax:                  convertTax(customer.Tax),
		TaxExempt:            string(customer.TaxExempt),
	}

	// combine the two pieces of information into one cache state
	cacheState := StripeCacheData{
		CustomerData:  stripeCusData,
		Subscriptions: subCache,
		Payments:      paymentCache,
	}

	cacheStateJSON, err := json.Marshal(cacheState)

	if err != nil {
		fmt.Printf("\nFailed to marshal cacheState: %+v\n\n", err)
		return fmt.Errorf("failed to marshal cacheState: %w", err)
	}

	// update redis
	err = s.cacheClient.Set(ctx, stripeUpdateCusWithCusIdKey, cacheStateJSON, 0)

	if err != nil {
		fmt.Printf("\nFailed to sync and store stripe data into cache: %+v\n\n", err)
		return fmt.Errorf("failed to sync and store stripe data into cache: %w", err)
	}

	return nil
}

/**
* adds/sets the mapping between userId and customerId in cache
**/
func (s *service) AddCacheUserIdToCusId(ctx context.Context, userId uuid.UUID, customerId string) error {
	fmt.Printf("\nupdating customerId to userId in cache..\ncustomerId key provided was :%s\nuserId value provided was: %s\n\n", customerId, userId)
	key := s.cacheClient.GetUserIdFromCustomerIdKey(customerId)
	err := s.cacheClient.Set(ctx, key, userId.String(), 0)

	if err != nil {
		return fmt.Errorf("failed to cache userId to customerId mapping: %w", err)
	}

	return nil
}

/**
* adds/sets the mapping between customerId and userId in cache
**/
func (s *service) AddCacheCusIdToUserId(ctx context.Context, customerId string, userId uuid.UUID) error {
	key := s.cacheClient.GetCustomerIdFromUserIdKey(userId.String())
	err := s.cacheClient.Set(ctx, key, customerId, 0)

	if err != nil {
		return fmt.Errorf("failed to cache userId to customerId mapping: %w", err)
	}

	return nil
}

/**
* gets cached customerId with the userId
**/
func (s *service) GetCachedCusIdFromUserId(ctx context.Context, userId uuid.UUID) (string, error) {
	key := s.cacheClient.GetCustomerIdFromUserIdKey(userId.String())
	customerId, err := s.cacheClient.Get(ctx, key)

	// doesn't exist in cache, error, cache is supposed to have a mapping from this point
	fmt.Printf("key: %s\n", key)
	if err == redislib.Nil {
		fmt.Printf("No customerId exists for this userId in cache %s", userId)
		// retrive from database instead
		user, err := s.userService.GetByID(ctx, userId)

		if err != nil {
			fmt.Printf("No customerId exists for this userId %s", userId)
			return "", err
		}

		return user.ID.String(), nil
	}

	fmt.Printf("\ncustomerId from cache: %s\n\n", customerId)

	return customerId, nil
}

/**
* The get version of the payment cache sync method. Gets the latest up-to-date data from the cache if it exists,
* otherwise calls the sync method to update the cache.
**/
func (s *service) GetStripeData(ctx context.Context, customerId string) (*StripeCacheData, error) {
	customerDataFromCustomerIDKey := s.cacheClient.GetCustomerDataFromCustomerIdKey(customerId)

	// check if customer data already exists in the cache
	dataJSON, err := s.cacheClient.Get(ctx, customerDataFromCustomerIDKey)

	// if it doesn't we sync the data right there
	if err == redislib.Nil {

		fmt.Printf("Customer data doesn't exist in cache.\n")

		// TODO: sync from database

		// log other exceptions
		log.Printf("error when attempting to get cache data for customerID %s\nerr was:\n%+v\n", customerId, err)

		return s.GetStripeData(ctx, customerId)
	}

	// data already exists, just unmarshal and return it
	fmt.Printf("\ncache data before unmarshal: %+v\n\n", dataJSON)

	var cacheData StripeCacheData

	err = json.Unmarshal([]byte(dataJSON), &cacheData)
	if err != nil {
		fmt.Printf("error when attempting to get unmarshal data for customerID %s\nerr was:\n%+v\n", customerId, err)
		return nil, err
	}

	fmt.Printf("\ncache data after unmarshal: %+v\n\n", cacheData)
	return &cacheData, nil
}

func (s *service) SetupProducts(ctx context.Context, request *SetupProductsReq) (*SetupProductsResp, error) {
	return s.paymentProcessor.SetupProducts(ctx, request)
}

func (s *service) CreateCustomer(ctx context.Context, userId uuid.UUID, email string) (string, error) {
	// create customer on stripe and get customer id
	customerId, err := s.paymentProcessor.CreateCustomer(ctx, userId, email)

	if err != nil {
		fmt.Printf("Error occured when attemtping to create customer on stripe, %s\n", err.Error())
		return "", err
	}

	// update local user repo for mapping
	err = s.userService.UpdateStripeCustomer(ctx, userId, customerId)

	if err != nil {
		fmt.Printf("Error occured when attempting to update stripe customerId to user repo in CreateCustomer method: %s\n", err.Error())
		return "", err
	}

	return customerId, nil
}

func (s *service) SaveCard(ctx context.Context, customerId string) (string, error) {
	return s.paymentProcessor.SaveCard(ctx, customerId)
}

func (s *service) CreatePaymentIntent(ctx context.Context, amount int64, customerId string) (*CreatePaymentIntentResponse, error) {
	return s.paymentProcessor.CreatePaymentIntent(ctx, amount, customerId)
}

func (s *service) GetProducts(ctx context.Context) (*ProductListResponse, error) {
	return s.paymentProcessor.GetProducts(ctx)
}

func (s *service) PurchaseProduct(ctx context.Context, userId uuid.UUID, req *PurchaseProductRequest) (*PurchaseProductResponse, error) {
	res, err := s.paymentProcessor.PurchaseProduct(ctx, req)

	if err != nil {
		return nil, err
	}

	// create payments record in database to map payment status to that on the payment service

	fmt.Printf("\npurchase product request: %+v\n\n", req)
	fmt.Printf("\npurchase product payment processor response: %+v\n\n", res)

	err = s.repo.Create(ctx, userId, &PaymentIntentRequest{
		CustomerID: req.CustomerID,
		Amount:     res.Amount,
		IntentID:   res.PaymentIntentID,
	})

	if err != nil {
		return nil, err
	}

	return &PurchaseProductResponse{
		ClientSecret:    res.ClientSecret,
		PaymentIntentID: res.PaymentIntentID,
	}, nil
}

func (s *service) SetupSubscription(ctx context.Context, request *SetupProductsReq) (*SetupProductsResp, error) {
	return s.paymentProcessor.SetupSubscription(ctx, request)
}

/**
* Default Stripe-Recommended Flow:
*
* 1. Subscribe action
* User clicks "Subscribe" →  Backend creates Stripe subscription with
* `payment_behavior: "default_incomplete"` →  Returns client_secret
* → Frontend confirms payment with Stripe Elements →  User completes payment →
* Stripe redirects to success page
*
* 2. Database Storage (naive implementation, but recommended by Stripe)
* When subscription created → Store in DB as status: "incomplete" →  Wait for webhooks to update status to "active"
**/
func (s *service) SubscribeToProduct(ctx context.Context, userId uuid.UUID, req *SubscribeRequest) (*SubscribeResponse, error) {
	res, err := s.paymentProcessor.SubscribeToProduct(ctx, req)

	if err != nil {
		return nil, err
	}

	// store in database a new subscription with data from successful Stripe response
	err = s.repo.UpsertSubscriptionRecord(ctx, &Subscription{
		UserID:               userId,
		StripeCustomerID:     req.CustomerID,
		StripeSubscriptionID: res.SubscriptionID,
		Status:               res.Status,
	})

	if err != nil {
		fmt.Printf("\nError when creating a subscription record in DB: %+v\n\n", err)
		return nil, err
	}

	return res, nil
}

func (s *service) SubscribeToSite(ctx context.Context, userId uuid.UUID) (*SubscribeToSiteResponse, error) {
	customerID, err := s.GetCachedCusIdFromUserId(ctx, userId)

	if err != nil {
		fmt.Printf("\nError occured when attempting to get cached customerId from userId: %s\n\n", err)
		return nil, err
	}

	fmt.Printf("\ncustomerID from cache: %s\n\n", customerID)

	var ProSubscriptionProductID string = util.GetEnv("SUBSCRIPTION_PROD_ID", "")

	// subscribe users to the pre-setup subscription pro product
	_, err = s.SubscribeToProduct(ctx, userId, &SubscribeRequest{
		CustomerID: customerID,
		ProductID:  ProSubscriptionProductID, // subscription pro productID
	})

	if err != nil {
		fmt.Printf("\nError occured when attempting to subsribe to the site for pro plan: %s\n\n", err)
		return nil, err
	}

	// confirmed subscription, update user's subscribe status
	err = s.userService.Update(ctx, userId, &user.User{
		Subscribed: true,
	})

	// rollback subscription if DB fails
	if err != nil {
		fmt.Printf("Subscription update in DB failed, err: %s\n", err)
		// TODO: add the rollback
	}

	return &SubscribeToSiteResponse{}, nil
}

func (s *service) GetCachedUserIdByCustomerId(ctx context.Context, customerID string) (uuid.UUID, error) {
	key := s.cacheClient.GetUserIdFromCustomerIdKey(customerID)
	userIdStr, err := s.cacheClient.Get(ctx, key)

	// key doesn't exist, acquire userId to fill in cache
	if err == redislib.Nil {
		user, err := s.userService.GetByStripeCustomerID(ctx, customerID)

		if err != nil {
			fmt.Printf("err when attempting to get user with customerId %s: %v\n", customerID, err)
			return uuid.Nil, err
		}

		// store in cache
		s.cacheClient.Set(ctx, key, user.ID.String(), 0)

		// return the id
		return user.ID, nil
	}

	// unexpected errors
	if err != nil {
		fmt.Printf("err when attempting to get userId from cache with customerId %s: %v\n", customerID, err)
		return uuid.Nil, err
	}

	// convert back to uuid
	userId, err := uuid.Parse(userIdStr)

	if err != nil {
		return uuid.Nil, err
	}

	return userId, err
}

// --- Full Flow Methods ---

/**
* Recieves a payment processor event and parses the event
**/
func (s *service) ProcessWebhookEvent(ctx context.Context, event *stripe.Event) error {
	customerId, err := s.paymentProcessor.ProcessWebhookEvent(ctx, event)

	if err != nil {
		fmt.Printf("\npaymentProcessor method ProcessWebhookEvent could not process incoming event of %+v, err :%+v\n\n", event, err)
		return err
	}

	fmt.Printf("Service layer - customerId: %s\n", customerId)

	s.SyncStripeDataToStorage(ctx, customerId)

	return nil
}

/**
*  utilizing cache for checking the user's subscription status
**/
func (s *service) GetSubscriptionStatusCache(ctx context.Context, userId uuid.UUID) (*bool, error) {
	// get customerId from cache
	cusId, err := s.GetCachedCusIdFromUserId(ctx, userId)

	if err != nil {
		return nil, err
	}

	fmt.Printf("\ncusId from cache: \n%+v\n\n", cusId)

	// get subscription status
	stripeCacheData, err := s.GetStripeData(ctx, cusId)

	fmt.Printf("\nstripeCachData when getting subscription status cache: \n%+v\n\n", stripeCacheData)

	return nil, nil
}

/**
* retrieves the user's subscription status from cache or database depending on availability.
**/
func (s *service) GetSubscriptionStatus(ctx context.Context, userId uuid.UUID) (*bool, error) {
	subStatusCache, err := s.GetSubscriptionStatusCache(ctx, userId)

	if err != nil {
		fmt.Printf("unexpected error when getting subsription status from cache: %s\n", err)
		return nil, err
	}

	if !*subStatusCache {
		subStatus, err := s.userService.GetSubscriptionStatus(ctx, userId)

		if err != nil {
			return nil, err
		}

		fmt.Printf("\nsubStatus when getting subscription status: \n%+v\n\n", subStatus)
		return &subStatus, nil
	}

	return subStatusCache, err
}

// Helper functions to convert Stripe types to our cache types

func convertAddress(addr *stripe.Address) *CustomerAddress {
	if addr == nil {
		return nil
	}
	return &CustomerAddress{
		City:       addr.City,
		Country:    addr.Country,
		Line1:      addr.Line1,
		Line2:      addr.Line2,
		PostalCode: addr.PostalCode,
		State:      addr.State,
	}
}

func convertCashBalance(cb *stripe.CashBalance) *CustomerCashBalance {
	if cb == nil {
		return nil
	}
	return &CustomerCashBalance{
		Object:    cb.Object,
		Available: cb.Available,
		Customer:  cb.Customer,
		Livemode:  cb.Livemode,
	}
}

func convertDefaultSource(ds *stripe.PaymentSource) *string {
	if ds == nil {
		return nil
	}
	id := ds.ID
	return &id
}

func convertDiscount(d *stripe.Discount) *CustomerDiscount {
	if d == nil {
		return nil
	}
	return &CustomerDiscount{
		Coupon:   convertCoupon(d.Coupon),
		Customer: d.ID,
		End:      d.End,
		Id:       d.ID,
		Object:   d.Object,
		Start:    d.Start,
	}
}

func convertCoupon(c *stripe.Coupon) *Coupon {
	if c == nil {
		return nil
	}
	return &Coupon{
		Id:               c.ID,
		Object:           c.Object,
		AmountOff:        c.AmountOff,
		Created:          c.Created,
		Currency:         string(c.Currency),
		Duration:         string(c.Duration),
		DurationInMonths: c.DurationInMonths,
		Livemode:         c.Livemode,
		MaxRedemptions:   c.MaxRedemptions,
		Name:             c.Name,
		PercentOff:       c.PercentOff,
		RedeemBy:         c.RedeemBy,
		TimesRedeemed:    c.TimesRedeemed,
		Valid:            c.Valid,
	}
}

func convertInvoiceSettings(is *stripe.CustomerInvoiceSettings) *CustomerInvoiceSettings {
	if is == nil {
		return nil
	}

	var customFields []*InvoiceCustomField
	for _, cf := range is.CustomFields {
		if cf != nil {
			customFields = append(customFields, &InvoiceCustomField{
				Name:  cf.Name,
				Value: cf.Value,
			})
		}
	}

	var defaultPM *string
	if is.DefaultPaymentMethod != nil {
		id := is.DefaultPaymentMethod.ID
		defaultPM = &id
	}

	var renderingOpts *InvoiceRenderingOptions
	if is.RenderingOptions != nil {
		renderingOpts = &InvoiceRenderingOptions{
			AmountTaxDisplay: string(is.RenderingOptions.AmountTaxDisplay),
		}
	}

	return &CustomerInvoiceSettings{
		CustomFields:         customFields,
		DefaultPaymentMethod: defaultPM,
		Footer:               is.Footer,
		RenderingOptions:     renderingOpts,
	}
}

func convertSubscriptions(s *stripe.SubscriptionList) *SubscriptionList {
	if s == nil {
		return nil
	}

	var data []interface{}
	for _, sub := range s.Data {
		data = append(data, sub)
	}

	return &SubscriptionList{
		Data:    data,
		HasMore: s.HasMore,
		Url:     s.URL,
	}
}

func convertTax(t *stripe.CustomerTax) *CustomerTax {
	if t == nil {
		return nil
	}

	var location *CustomerTaxLocation
	if t.Location != nil {
		location = &CustomerTaxLocation{
			Country: t.Location.Country,
			Source:  string(t.Location.Source),
			State:   t.Location.State,
		}
	}

	return &CustomerTax{
		AutomaticTax: string(t.AutomaticTax),
		IpAddress:    t.IPAddress,
		Location:     location,
	}
}

func convertTaxIds(t *stripe.TaxIDList) *CustomerTaxIdList {
	if t == nil {
		return nil
	}

	var data []CustomerTaxId
	for _, taxId := range t.Data {
		if taxId != nil {
			data = append(data, CustomerTaxId{
				Id:      taxId.ID,
				Object:  taxId.Object,
				Country: taxId.Country,
				Type:    string(taxId.Type),
				Value:   taxId.Value,
			})
		}
	}

	return &CustomerTaxIdList{
		Data:    data,
		HasMore: t.HasMore,
		Url:     t.URL,
	}
}
