package user

import (
	"context"
)

// GetByStripeCustomerID retrieves a user by their Stripe customer ID
func (s *service) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*User, error) {
	return s.repo.GetByStripeCustomerID(ctx, stripeCustomerID)
}