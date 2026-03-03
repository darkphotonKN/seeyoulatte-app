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

// core layer for payment processors, allowing adapters to build custom connections to
// various specific processors e.g. like stripe
type PaymentProcessor interface {
	CreateCustomer(ctx context.Context, req CreateCustomerRequest) (*CreateCustomerResponse, error)
	CreatePaymentIntent(ctx context.Context, amount int64, customerId string, metadata map[string]string) (*CreatePaymentIntentResponse, error)
}
