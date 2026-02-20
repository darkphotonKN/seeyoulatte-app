package listing

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

func (r *repository) Create(ctx context.Context, listing *Listing) error {
	query := `
		INSERT INTO listings (
			seller_id, title, description, category, price,
			quantity, pickup_instructions, expires_at, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) RETURNING id, created_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		listing.SellerID,
		listing.Title,
		listing.Description,
		listing.Category,
		listing.Price,
		listing.Quantity,
		listing.PickupInstructions,
		listing.ExpiresAt,
		listing.IsActive,
	).Scan(&listing.ID, &listing.CreatedAt)

	if err != nil {
		return errorutils.AnalyzeDBErr(err)
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*Listing, error) {
	var listing Listing
	query := `
		SELECT
			id, seller_id, title, description, category, price,
			quantity, pickup_instructions, expires_at, is_active, created_at
		FROM listings
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &listing, query, id)
	if err != nil {
		dbErr := errorutils.AnalyzeDBErr(err)
		if dbErr == errorutils.ErrNotFound {
			return nil, nil
		}
		return nil, dbErr
	}

	return &listing, nil
}

func (r *repository) GetAllPublic(ctx context.Context) ([]Listing, error) {
	var listings []Listing
	query := `
		SELECT
			id, seller_id, title, description, category, price,
			quantity, pickup_instructions, expires_at, is_active, created_at
		FROM listings
		WHERE is_active = true
			AND quantity > 0
			AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &listings, query)
	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	return listings, nil
}

func (r *repository) GetBySellerID(ctx context.Context, sellerID uuid.UUID) ([]Listing, error) {
	var listings []Listing
	query := `
		SELECT
			id, seller_id, title, description, category, price,
			quantity, pickup_instructions, expires_at, is_active, created_at
		FROM listings
		WHERE seller_id = $1
		ORDER BY created_at DESC
	`

	err := r.db.SelectContext(ctx, &listings, query, sellerID)
	if err != nil {
		return nil, errorutils.AnalyzeDBErr(err)
	}

	return listings, nil
}

func (r *repository) Update(ctx context.Context, listing *Listing) error {
	query := `
		UPDATE listings SET
			title = $2,
			description = $3,
			price = $4,
			quantity = $5,
			pickup_instructions = $6,
			is_active = $7,
			expires_at = $8
		WHERE id = $1
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		listing.ID,
		listing.Title,
		listing.Description,
		listing.Price,
		listing.Quantity,
		listing.PickupInstructions,
		listing.IsActive,
		listing.ExpiresAt,
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
	query := `DELETE FROM listings WHERE id = $1`

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