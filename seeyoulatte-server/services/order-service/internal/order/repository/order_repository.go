package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// orderRow is the DB-shaped struct. Lives ONLY in this package, never leaves it.
// Separate from domain.Order on purpose: schema can evolve independently from
// the domain model, and the mapper is the bridge.
type orderRow struct {
	ID              uuid.UUID  `db:"id"`
	ListingID       uuid.UUID  `db:"listing_id"`
	BuyerID         uuid.UUID  `db:"buyer_id"`
	SellerID        uuid.UUID  `db:"seller_id"`
	Quantity        int        `db:"quantity"`
	Amount          float64    `db:"amount"`
	State           string     `db:"state"`
	SellerRespondBy *time.Time `db:"seller_respond_by"`
	ReviewEndsAt    *time.Time `db:"review_ends_at"`
	CreatedAt       time.Time  `db:"created_at"`
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
	query := `
	SELECT
		id,
		listing_id,
		buyer_id,
		seller_id,
		quantity,
		amount,
		state,
		seller_respond_by,
		review_ends_at,
		created_at
	FROM orders
	WHERE id = $1
	`
	var row orderRow // db specific struct

	// 1. pull order row from the database first
	err := r.db.GetContext(ctx, &row, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("error querying order %s: %w", id, err)
	}

	mappedToDomain := domain.ReconstituteParams{
		ID:              row.ID,
		ListingID:       row.ListingID,
		BuyerID:         row.BuyerID,
		SellerID:        row.SellerID,
		Quantity:        row.Quantity,
		Amount:          row.Amount,
		State:           domain.OrderState(row.State),
		SellerRespondBy: row.SellerRespondBy,
		ReviewEndsAt:    row.ReviewEndsAt,
		CreatedAt:       row.CreatedAt,
	}

	order, err := domain.Reconstitute(mappedToDomain)
	if err != nil {
		return nil, fmt.Errorf("reconstituting order %s from db row: %w", id, err)
	}

	return order, nil
}

// Save persists a mutated aggregate. Reads state out via Snapshot()
// (the one read-out door for sealed fields).
func (r *OrderRepository) Save(ctx context.Context, order *domain.Order) error {
	// get the fields out from the order and save them in a copy
	snap := order.Snapshot()

	// update row in db
	query := `
	UPDATE orders SET
		state = :state,
		seller_respond_by = :seller_respond_by,
		review_ends_at = :review_ends_at
	WHERE id = :id
	`
	res, err := r.db.NamedExecContext(ctx, query, map[string]interface{}{
		"state":             string(snap.State),
		"seller_respond_by": snap.SellerRespondBy,
		"review_ends_at":    snap.ReviewEndsAt,
		"id":                snap.ID,
	})

	if err != nil {
		return fmt.Errorf("updating order %s: %w", snap.ID, err)
	}

	// 3. Check RowsAffected == 0 → the row vanished or never existed.
	rows, err := res.RowsAffected()
	// driver error
	if err != nil {
		return fmt.Errorf("checking affected rows for order %s: %w", snap.ID, err)
	}

	if rows == 0 {
		return domain.ErrOrderNotFound
	}

	// TODO(concurrency): optimistic locking goes here eventually —
	// add `version` column to WHERE, bump on UPDATE, treat 0 rows as conflict
	// (translate to domain.ErrConcurrentModification or similar).
	return nil
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
	return nil
}
