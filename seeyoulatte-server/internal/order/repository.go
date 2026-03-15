package order

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/darkphotonKN/seeyoulatte-app/internal/utils/errorutils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"google.golang.org/grpc/balancer/grpclb/state"
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

func (r *repository) CreateTx(ctx context.Context, tx *sqlx.Tx, order *Order) error {
	query := `
		INSERT INTO orders (
			listing_id, buyer_id, seller_id, quantity, amount,
			state, seller_respond_by, review_ends_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) RETURNING id, created_at
	`

	err := tx.QueryRowContext(
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

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	var order Order
	query := `
		SELECT
			id, listing_id, buyer_id, seller_id, quantity, amount,
			state, seller_respond_by, review_ends_at, created_at
		FROM orders
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &order, query, id)
	if err != nil {
		dbErr := errorutils.AnalyzeDBErr(err)
		if dbErr == errorutils.ErrNotFound {
			return nil, fmt.Errorf("order not found: %w", dbErr)
		}
		return nil, fmt.Errorf("failed to get order: %w", dbErr)
	}

	return &order, nil
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

func (r *repository) UpdateStateTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, userID uuid.UUID, newState state.OrderState) (*Order, error) {
	query := `
		UPDATE orders SET
			state = $2,
		WHERE id = $1 
		AND user_id = $3
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
	)

	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("checking affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return nil, errorutils.ErrNotFound
	}

	return nil, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// TODO: Add validation to ensure user has permission to delete this order
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting order: %w", err)
	}

	s.logger.Info("order deleted",
		slog.String("order_id", id.String()),
		slog.String("user_id", userID.String()))

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

// GetPendingPaymentOrdersByUser retrieves all orders with pending_payment state for a specific user
func (r *repository) GetPendingPaymentOrdersByUser(ctx context.Context, userID uuid.UUID) ([]*Order, error) {
	query := `
		SELECT id, listing_id, buyer_id, seller_id, quantity, amount,
		       state, seller_respond_by, review_ends_at, created_at
		FROM orders
		WHERE buyer_id = $1 AND state = 'pending_payment'
		ORDER BY created_at DESC
	`

	var orders []*Order
	err := r.db.SelectContext(ctx, &orders, query, userID)
	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	return orders, nil
}

