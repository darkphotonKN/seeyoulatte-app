package payment

import (
	"time"

	"github.com/google/uuid"
)

// Payment represents a payment transaction
type Payment struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	OrderID         uuid.UUID  `json:"order_id" db:"order_id"`
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	Amount          float64    `json:"amount" db:"amount"`
	Currency        string     `json:"currency" db:"currency"`
	Status          string     `json:"status" db:"status"`
	PaymentMethod   string     `json:"payment_method" db:"payment_method"`
	StripePaymentID *string    `json:"stripe_payment_id,omitempty" db:"stripe_payment_id"`
	StripeChargeID  *string    `json:"stripe_charge_id,omitempty" db:"stripe_charge_id"`
	FailureReason   *string    `json:"failure_reason,omitempty" db:"failure_reason"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

// CreatePaymentRequest represents the request to create a payment
type CreatePaymentRequest struct {
	OrderID       uuid.UUID `json:"order_id" validate:"required"`
	PaymentMethod string    `json:"payment_method" validate:"required,oneof=card bank_transfer"`
	Token         string    `json:"token,omitempty"`      // Stripe token for card payments
	ReturnURL     string    `json:"return_url,omitempty"` // URL to redirect after payment
}

// CreatePaymentResponse represents the response after creating a payment
type CreatePaymentResponse struct {
	PaymentID    uuid.UUID `json:"payment_id"`
	Status       string    `json:"status"`
	Amount       float64   `json:"amount"`
	Currency     string    `json:"currency"`
	ClientSecret *string   `json:"client_secret,omitempty"` // For Stripe payment confirmation
	RedirectURL  *string   `json:"redirect_url,omitempty"`  // For redirect-based payment methods
	NextAction   *string   `json:"next_action,omitempty"`   // Instructions for next step
}

// PaymentStatus constants
const (
	PaymentStatusPending    = "pending"
	PaymentStatusProcessing = "processing"
	PaymentStatusSucceeded  = "succeeded"
	PaymentStatusFailed     = "failed"
	PaymentStatusCanceled   = "canceled"
	PaymentStatusRefunded   = "refunded"
)

// PaymentMethod constants
const (
	PaymentMethodCard         = "card"
	PaymentMethodBankTransfer = "bank_transfer"
)

// CreatePaymentIntentResponse represents the response from creating a Stripe payment intent
type CreatePaymentIntentResponse struct {
	PaymentIntentID string `json:"payment_intent_id"`
	ClientSecret    string `json:"client_secret"`
	Amount          int64  `json:"amount"` // Amount in cents
	Currency        string `json:"currency"`
	OrderID         string `json:"order_id"`
	Status          string `json:"status"`
}

