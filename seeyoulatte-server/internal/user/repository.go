package user

import (
	"context"
	"fmt"
	"time"

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

func (r *repository) Create(ctx context.Context, user *User) error {
	query := `
		INSERT INTO users (
			email, password_hash, name, bio, location_text,
			google_id, avatar_url, is_verified
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		user.Email,
		user.PasswordHash,
		user.Name,
		user.Bio,
		user.LocationText,
		user.GoogleID,
		user.AvatarURL,
		user.IsVerified,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return errorutils.AnalyzeDBErr(err)
	}

	return nil
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	query := `
		SELECT
			id, email, password_hash, name, bio, location_text,
			is_frozen, google_id, avatar_url, is_verified,
			preferred_pickup_instructions, created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		dbErr := errorutils.AnalyzeDBErr(err)
		if dbErr == errorutils.ErrNotFound {
			return nil, nil
		}
		return nil, dbErr
	}

	return &user, nil
}

func (r *repository) GetByIDForUpdateTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID) (*User, error) {
	var user User
	query := `
		SELECT
			id, email, password_hash, name, bio, location_text,
			is_frozen, google_id, avatar_url, is_verified,
			preferred_pickup_instructions, created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
		FOR UPDATE
	`

	err := tx.GetContext(ctx, &user, query, id)
	if err != nil {
		dbErr := errorutils.AnalyzeDBErr(err)
		if dbErr == errorutils.ErrNotFound {
			return nil, nil
		}
		return nil, dbErr
	}

	return &user, nil
}

func (r *repository) GetByIDNotIsFrozen(ctx context.Context, id uuid.UUID) error {
	var user User
	query := `
		SELECT
			id, 
			is_frozen
		FROM users
		WHERE id = $1 AND is_frozen != true
	`

	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		dbErr := errorutils.AnalyzeDBErr(err)
		if dbErr == errorutils.ErrNotFound {
			return nil
		}
		return dbErr
	}

	return nil
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	query := `
		SELECT
			id, email, password_hash, name, bio, location_text,
			is_frozen, google_id, avatar_url, is_verified,
			preferred_pickup_instructions, created_at, updated_at, last_login_at
		FROM users
		WHERE email = $1
	`

	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		dbErr := errorutils.AnalyzeDBErr(err)
		if dbErr == errorutils.ErrNotFound {
			return nil, nil
		}
		return nil, dbErr
	}

	return &user, nil
}

func (r *repository) GetByGoogleID(ctx context.Context, googleID string) (*User, error) {
	var user User
	query := `
		SELECT
			id, email, password_hash, name, bio, location_text,
			is_frozen, google_id, avatar_url, is_verified,
			preferred_pickup_instructions, created_at, updated_at, last_login_at
		FROM users
		WHERE google_id = $1
	`

	err := r.db.GetContext(ctx, &user, query, googleID)
	if err != nil {
		dbErr := errorutils.AnalyzeDBErr(err)
		if dbErr == errorutils.ErrNotFound {
			return nil, nil
		}
		return nil, dbErr
	}

	return &user, nil
}

func (r *repository) Update(ctx context.Context, user *User) error {
	query := `
		UPDATE users SET
			email = $1,
			name = $2,
			bio = $3,
			location_text = $4,
			google_id = $5,
			avatar_url = $6,
			is_verified = $7,
			preferred_pickup_instructions = $8,
			updated_at = NOW()
		WHERE id = $9
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		user.Email,
		user.Name,
		user.Bio,
		user.LocationText,
		user.GoogleID,
		user.AvatarURL,
		user.IsVerified,
		user.PreferredPickupInstructions,
		user.ID,
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

func (r *repository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	query := `UPDATE users SET last_login_at = $1 WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, now, userID)
	if err != nil {
		return errorutils.AnalyzeDBErr(err)
	}

	return nil
}
