package user

import (
	"context"

	"github.com/google/uuid"
)

// GetByStripeCustomerID retrieves a user by their Stripe customer ID
func (s *service) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*User, error) {
	return s.repo.GetByStripeCustomerID(ctx, stripeCustomerID)
}

// GetUserIDByStripeCustomerID retrieves only the user ID by their Stripe customer ID
func (s *service) GetUserIDByStripeCustomerID(ctx context.Context, stripeCustomerID string) (uuid.UUID, error) {
	return s.repo.GetUserIDByStripeCustomerID(ctx, stripeCustomerID)
}