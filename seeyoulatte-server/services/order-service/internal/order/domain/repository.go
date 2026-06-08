package domain

import (
	"context"

	"github.com/google/uuid"
)

// OrderRepository is the aggregate's persistence port. The domain declares
// the interface; the repository package implements it. Returns and accepts
// rich entities only, DB rows, never DTOs
//
// Lives in the domain (not in any single usecase) because the contract is
// aggregate-scoped, not consumer-scoped: every usecase that touches Order
// needs the same load/save primitives, so the aggregate owns the contract.
// Contrast with EventPublisher / LedgerService, which are consumer-owned
// (declared in the usecase per ISP) because their shape varies per consumer
//
// Stays small (FindByID / Save / Insert) by discipline: any list, filter,
// or projection operation belongs in the query layer (CQRS), not here
type OrderRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Order, error)
	Save(ctx context.Context, o *Order) error
	Insert(ctx context.Context, o *Order) error
}
