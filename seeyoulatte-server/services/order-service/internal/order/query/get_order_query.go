package query

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/dto"
	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order/domain"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type GetOrderQuery struct {
	db *sqlx.DB
}

func NewGetOrderQuery(db *sqlx.DB) *GetOrderQuery {
	return &GetOrderQuery{db: db}
}

func (q *GetOrderQuery) Execute(ctx context.Context, id uuid.UUID) (*dto.OrderDetail, error) {
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

	var orderDetail dto.OrderDetail

	err := q.db.GetContext(ctx, &orderDetail, query, id)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("unexpected error: %v", err)
	}

	return &orderDetail, nil
}
