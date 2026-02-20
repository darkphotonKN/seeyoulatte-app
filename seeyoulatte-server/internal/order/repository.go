package order

import (
	"context"
	"fmt"

	"github.com/darkphotonKN/seeyoulatte-app/internal/utils/errorutils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *repository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, order *Order) error {
	query := `
		INSERT INTO orders (
			listing_id, buyer_id, seller_id, quantity, amount,
			state, seller_respond_by, review_ends_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) RETURNING id, created_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		order.ListingID,
		order.BuyerID,
		order.SellerID,
		order.Quantity,
		order.Amount,
		order.State,
		order.SellerRespondBy,
		order.ReviewEndsAt,
	).Scan(&order.ID, &order.CreatedAt)

	if err != nil {
		return errorutils.AnalyzeDBErr(err)
	}

	return nil
}

func (r *repository) GetAll(ctx context.Context) ([]Order, error) {
	var orders []Order
	query := `
		SELECT
			id, listing_id, buyer_id, seller_id, quantity, amount,
			state, seller_respond_by, review_ends_at, created_at
		FROM orders
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &orders, query)
	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	return orders, nil
}

func (r *repository) Update(ctx context.Context, order *Order) error {
	query := `
		UPDATE orders SET
			state = $2,
			seller_respond_by = $3,
			review_ends_at = $4
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		order.ID,
		order.State,
		order.SellerRespondBy,
		order.ReviewEndsAt,
	)

	if err != nil {
		return errorutils.AnalyzeDBErr(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return errorutils.ErrNotFound
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM orders WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return errorutils.AnalyzeDBErr(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return errorutils.ErrNotFound
	}

	return nil
}