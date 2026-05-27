package domain

import (
	"context"

	"github.com/google/uuid"
)

// OrderRepository is the aggregate's persistence port. The domain declares it;
// the repository package implements it. Returns/accepts rich entities only.
type OrderRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Order, error)
	Save(ctx context.Context, o *Order) error
}
