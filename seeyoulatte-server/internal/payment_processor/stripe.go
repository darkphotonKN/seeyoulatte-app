package paymentprocessor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/customer"
	"github.com/stripe/stripe-go/v82/paymentintent"
)

type StripeProcessor struct{}

func NewStripeProcessor() *StripeProcessor {
	// Set Stripe secret key from environment
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	return &StripeProcessor{}
}

// CreateCustomer creates a new Stripe customer for a user
func (s *StripeProcessor) CreateCustomer(ctx context.Context, req *CreateCustomerRequest) (*CreateCustomerResponse, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(req.Email),
		Metadata: map[string]string{
			"user_id": req.UserId.String(),
		},
	}

	// Create customer on Stripe
	cust, err := customer.New(params)
	if err != nil {
		return nil, fmt.Errorf("creating stripe customer: %w", err)
	}

	return &CreateCustomerResponse{
		CustomerID: cust.ID,
	}, nil
}

// CreatePaymentIntent creates a payment authorization token that allows the frontend to charge a specific
// amount for a specific customer. The backend validates the request and gets Stripe's
// permission to charge, but the actual payment happens when the frontend confirms
// with card data and is only executed on the frontend through Stripe's secure iFrame
// elements. This prevents unauthorized charges while keeping card data secure.
func (s *StripeProcessor) CreatePaymentIntent(ctx context.Context, amount int64, customerId string, metadata map[string]string) (*CreatePaymentIntentResponse, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String("usd"),
		Customer: stripe.String(customerId),

		// Automatic confirmation means frontend confirms - this is default confirmation
		ConfirmationMethod: stripe.String("automatic"),
		Confirm:            stripe.Bool(false),

		// Only allow CARD payments
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),

		// Add metadata for tracking
		Metadata: metadata,
	}

	intent, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment intent: %w", err)
	}

	return &CreatePaymentIntentResponse{
		PaymentIntentID: intent.ID,
		ClientSecret:    intent.ClientSecret,
	}, nil
}

// ProcessWebhookEvent processes and validates webhook events from Stripe
func (s *StripeProcessor) ProcessWebhookEvent(ctx context.Context, event *WebhookEvent) (customerId string, error error) {
	isEventSupported := s.isWebhookEventSupported(ctx, event)

	if !isEventSupported {
		return "", fmt.Errorf("the event type that was resulted from the action was not allowed")
	}

	customerId, err := s.extractCustomerIdFromWebhook(event)
	if err != nil {
		return "", err
	}

	return customerId, nil
}

// IsWebhookEventSupported checks if the webhook event is one we handle
func (s *StripeProcessor) isWebhookEventSupported(ctx context.Context, event *WebhookEvent) bool {
	// Store allowed / expected webhook events
	expectedEvents := map[stripe.EventType]bool{
		stripe.EventTypePaymentIntentSucceeded:     true,
		stripe.EventTypePaymentIntentPaymentFailed: true,
		stripe.EventTypePaymentIntentCanceled:      true,
	}

	// adapting to stripe's own type
	stripeEventType := stripe.EventType(event.EventType)
	if !expectedEvents[stripeEventType] {
		return false
	}

	return true
}

// ExtractCustomerIdFromWebhook extracts the Stripe customer ID from a webhook event
func (s *StripeProcessor) extractCustomerIdFromWebhook(event *WebhookEvent) (string, error) {

	// unmarshal to expected stripe.Event type
	var stripeEvent stripe.Event
	err := json.Unmarshal(event.RawBody, &stripeEvent)

	if err != nil {
		slog.Error("Error when attempting to unmarshal raw body in webhook event",
			"error", err)
		return "", fmt.Errorf("invalid event type: expected *stripe.Event")
	}

	var eventData map[string]interface{}
	err = json.Unmarshal(stripeEvent.Data.Raw, &eventData)
	if err != nil {
		slog.Error("Error when attempting to unmarshal stripeEvent.Data.Raw in webhook event",
			"error", err)
		return "", fmt.Errorf("unmarshaling event data: %w", err)
	}

	if customer, ok := eventData["customer"].(string); ok && customer != "" {
		return customer, nil
	}

	return "", fmt.Errorf("no customer ID found in stripe event type: %s", stripeEvent.Type)
}

// Retrieves current state from stripe with teh customer's current information like
// list of payments as well as current payment details

func (s *StripeProcessor) FetchCurrentState(ctx context.Context, customerId string) (*CustomerState, error) {

	// -- customer --

	customer, err := customer.Get(customerId, nil)

	if err != nil {
		slog.Error("Failed to get customer from stripe",
			"error", err)
		return nil, fmt.Errorf("failed to get customer from stripe: %w", err)
	}

	// -- payments --

	stripePayments := []*stripe.PaymentIntent{}

	paymentParams := &stripe.PaymentIntentListParams{
		Customer: stripe.String(customerId),
	}

	// include payment method details
	paymentParams.AddExpand("data.payment_method")

	paymentIter := paymentintent.List(paymentParams)

	for paymentIter.Next() {
		pi := paymentIter.PaymentIntent()
		slog.Debug("Current iteration of payment intent",
			"payment_intent", pi)

		stripePayments = append(stripePayments, pi)
	}

	// convert to conform to abstracted type
	payments := make([]*PaymentState, len(stripePayments))

	for idx, payment := range stripePayments {
		slog.Debug("Current iteration of payment intent",
			"loop_index", idx,
			"payment", payment,
		)

		payments[idx] = &PaymentState{
			IntentID: payment.ID,
			OrderID:  payment.Metadata["order_id"],
			Status:   string(payment.Status),
			Amount:   payment.Amount,
			Currency: string(payment.Currency),
		}
	}

	return &CustomerState{
		CustomerID: customer.ID,
		Email:      customer.Email,
		Payments:   payments,
	}, nil
}
