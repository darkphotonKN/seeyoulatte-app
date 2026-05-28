package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// orderRow is the DB-shaped struct. Lives ONLY in this package, never leaves it.
// Separate from domain.Order on purpose: schema can evolve independently from
// the domain model, and the mapper is the bridge.
type orderRow struct {
	id         uuid.UUID `db:"id"`
	listing_id uuid.UUID `db:"listing_id"`
	buyer_id   uuid.UUID `db:"buyer_id"`
	seller_id  uuid.UUID `db:"seller_id"`
	quantity   int       `db:"quantity"`
	amount     uuid.UUID `db:"amount"`
	state      uuid.UUID `db:"state"`
}

// OrderRepository implements domain.OrderRepository.
// Interface = port (domain package). This struct = adapter.
type OrderRepository struct {
	db *sqlx.DB
}

func NewOrderRepository(db *sqlx.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// FindByID loads a row and rebuilds the rich entity via Reconstitute.
// This is the only place in the codebase that turns a DB row into a *domain.Order.
func (r *OrderRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	// 1. SELECT into orderRow (sqlx GetContext, plain SELECT by id).
	//    On sql.ErrNoRows → return domain.ErrOrderNotFound (translate
	//    persistence error → domain error so the usecase sees only domain vocab).
	query := `
	SELECT
	`

	// 2. Map orderRow → domain.ReconstituteParams (mechanical, field by field).

	// 3. Call domain.Reconstitute(params). Wrap its error if any —
	//    Reconstitute only fails on structural sanity (invalid id, invalid state),
	//    which means the DB has garbage. That's a data integrity bug, not a user
	//    error — surface it loudly.

	// 4. Return the rich entity.
}

// Save persists a mutated aggregate. Reads state out via Snapshot()
// (the one read-out door for sealed fields).
func (r *OrderRepository) Save(ctx context.Context, order *domain.Order) error {
	// 1. snap := order.Snapshot()

	// 2. UPDATE orders SET <mutable fields> WHERE id = $1
	//    Mutable: state, seller_respond_by, review_ends_at
	//    NOT mutable: id, listing_id, buyer_id, seller_id, quantity, amount, created_at
	//    (those are set at birth and never change — don't include them in the UPDATE)

	// 3. Check RowsAffected == 0 → the row vanished or never existed.
	//    Return domain.ErrOrderNotFound.

	// TODO(concurrency): optimistic locking goes here eventually —
	// add `version` column to WHERE, bump on UPDATE, treat 0 rows as conflict
	// (translate to domain.ErrConcurrentModification or similar).
}

// Insert persists a newly-created aggregate. Separate from Save because the
// shapes differ: Insert writes ALL fields including the immutable ones;
// Save updates only the mutable ones. Lets each method be honest about its job.
func (r *OrderRepository) Insert(ctx context.Context, order *domain.Order) error {
	// 1. snap := order.Snapshot()

	// 2. INSERT INTO orders (...) VALUES (...) — all 10 fields.
	//    No RETURNING clause needed: id and createdAt are domain-owned
	//    (set in NewOrder), so we already have them.

	// 3. Return any error wrapped with context.
}
