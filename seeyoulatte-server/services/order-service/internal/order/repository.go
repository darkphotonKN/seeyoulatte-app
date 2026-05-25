package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/constants"
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
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query,
		order.ListingID, order.BuyerID, order.SellerID,
		order.Quantity, order.Amount, order.State,
		order.SellerRespondBy, order.ReviewEndsAt,
	).Scan(&order.ID, &order.CreatedAt)
}

func (r *repository) CreateTx(ctx context.Context, tx *sqlx.Tx, order *Order) error {
	query := `
		INSERT INTO orders (
			listing_id, buyer_id, seller_id, quantity, amount,
			state, seller_respond_by, review_ends_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at
	`
	return tx.QueryRowContext(ctx, query,
		order.ListingID, order.BuyerID, order.SellerID,
		order.Quantity, order.Amount, order.State,
		order.SellerRespondBy, order.ReviewEndsAt,
	).Scan(&order.ID, &order.CreatedAt)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	var o Order
	err := r.db.GetContext(ctx, &o,
		`SELECT id, listing_id, buyer_id, seller_id, quantity, amount,
		        state, seller_respond_by, review_ends_at, created_at
		 FROM orders WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("order not found: %w", err)
		}
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	return &o, nil
}

func (r *repository) GetAll(ctx context.Context) ([]Order, error) {
	var orders []Order
	err := r.db.SelectContext(ctx, &orders,
		`SELECT id, listing_id, buyer_id, seller_id, quantity, amount,
		        state, seller_respond_by, review_ends_at, created_at
		 FROM orders
		 ORDER BY created_at DESC`)
	return orders, err
}

func (r *repository) GetPendingPaymentOrdersByUser(ctx context.Context, userID uuid.UUID) ([]*Order, error) {
	var orders []*Order
	err := r.db.SelectContext(ctx, &orders,
		`SELECT id, listing_id, buyer_id, seller_id, quantity, amount,
		        state, seller_respond_by, review_ends_at, created_at
		 FROM orders
		 WHERE buyer_id = $1 AND state = 'pending_payment'
		 ORDER BY created_at DESC`, userID)
	return orders, err
}

func (r *repository) Update(ctx context.Context, order *Order) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE orders SET
			state = $2,
			seller_respond_by = $3,
			review_ends_at = $4
		 WHERE id = $1`,
		order.ID, order.State, order.SellerRespondBy, order.ReviewEndsAt)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *repository) UpdateStateTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, newState constants.OrderState) (*Order, error) {
	res, err := tx.ExecContext(ctx,
		`UPDATE orders SET state = $2 WHERE id = $1`,
		id, string(newState))
	if err != nil {
		return nil, err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return nil, sql.ErrNoRows
	}
	var o Order
	if err := tx.GetContext(ctx, &o,
		`SELECT id, listing_id, buyer_id, seller_id, quantity, amount,
		        state, seller_respond_by, review_ends_at, created_at
		 FROM orders WHERE id = $1`, id); err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM orders WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
