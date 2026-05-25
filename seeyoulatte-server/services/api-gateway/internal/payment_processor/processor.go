package paymentprocessor

import (
	"context"

	"github.com/google/uuid"
)

type CreateCustomerRequest struct {
	UserId   uuid.UUID
	Email    string
	Metadata map[string]string
}

type CreateCustomerResponse struct {
	CustomerID string `json:"customer_id"`
}

type CreatePaymentIntentResponse struct {
	ClientSecret    string `json:"client_secret"`
	PaymentIntentID string `json:"payment_intent_id"`
}

type CreatePaymentIntentRequest struct {
	Amount     int64  `json:"amount"`
	CustomerID string `json:"customer_id"`
}

type CustomerState struct {
	CustomerID string
	Email      string
	Payments   []*PaymentState
}

type PaymentState struct {
	IntentID string
	OrderID  string // from metadata you set when creating the checkout
	Status   string // succeeded, failed, canceled, processing
	Amount   int64
	Currency string
}

// core layer for payment processors, allowing adapters to build custom connections to
// various specific processors e.g. like stripe
type PaymentProcessor interface {
	CreateCustomer(ctx context.Context, req *CreateCustomerRequest) (*CreateCustomerResponse, error)
	CreatePaymentIntent(ctx context.Context, amount int64, customerId string, metadata map[string]string) (*CreatePaymentIntentResponse, error)
	ProcessWebhookEvent(ctx context.Context, webhookEvent *WebhookEvent) (customerId string, error error)
	FetchCurrentState(ctx context.Context, customerId string) (*CustomerState, error)
}

// represents a webhook event, with its type and raw payload
// as a general all purpose type that would fit multiple processors
type WebhookEvent struct {
	EventType string
	RawBody   []byte
}
