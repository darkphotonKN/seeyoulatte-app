package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	commonconstants "github.com/darkphotonKN/seeyoulatte-app/common/constants"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type repository struct {
	DB *sqlx.DB
}

func NewRepository(db *sqlx.DB) *repository {
	return &repository{DB: db}
}

func (r *repository) Create(ctx context.Context, u *User) error {
	query := `
		INSERT INTO users (
			email, password_hash, name, bio, location_text,
			google_id, avatar_url, is_verified, stripe_customer_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		) RETURNING id, created_at, updated_at
	`
	return r.DB.QueryRowContext(ctx, query,
		u.Email, u.PasswordHash, u.Name, u.Bio, u.LocationText,
		u.GoogleID, u.AvatarURL, u.IsVerified, u.StripeCustomerID,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *repository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var u User
	err := r.DB.GetContext(ctx, &u,
		`SELECT id, email, password_hash, name, bio, location_text,
		        is_frozen, google_id, avatar_url, is_verified,
		        preferred_pickup_instructions, stripe_customer_id,
		        created_at, updated_at, last_login_at
		 FROM users WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, commonconstants.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &u, nil
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.DB.GetContext(ctx, &u,
		`SELECT id, email, password_hash, name, bio, location_text,
		        is_frozen, google_id, avatar_url, is_verified,
		        preferred_pickup_instructions, stripe_customer_id,
		        created_at, updated_at, last_login_at
		 FROM users WHERE email = $1`, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, commonconstants.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &u, nil
}

func (r *repository) GetByGoogleID(ctx context.Context, googleID string) (*User, error) {
	var u User
	err := r.DB.GetContext(ctx, &u,
		`SELECT id, email, password_hash, name, bio, location_text,
		        is_frozen, google_id, avatar_url, is_verified,
		        preferred_pickup_instructions, stripe_customer_id,
		        created_at, updated_at, last_login_at
		 FROM users WHERE google_id = $1`, googleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, commonconstants.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by google id: %w", err)
	}
	return &u, nil
}

func (r *repository) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*User, error) {
	var u User
	err := r.DB.GetContext(ctx, &u,
		`SELECT id, email, password_hash, name, bio, location_text,
		        is_frozen, google_id, avatar_url, is_verified,
		        preferred_pickup_instructions, stripe_customer_id,
		        created_at, updated_at, last_login_at
		 FROM users WHERE stripe_customer_id = $1`, stripeCustomerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, commonconstants.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by stripe customer id: %w", err)
	}
	return &u, nil
}

func (r *repository) ListAll(ctx context.Context) ([]*User, error) {
	var users []*User
	err := r.DB.SelectContext(ctx, &users,
		`SELECT id, email, NULL AS password_hash, name, bio, location_text,
		        is_frozen, NULL AS google_id, avatar_url, is_verified,
		        preferred_pickup_instructions, stripe_customer_id,
		        created_at, updated_at, last_login_at
		 FROM users
		 ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}

func (r *repository) UpdateProfile(ctx context.Context, in *UpdateProfileInput) (*User, error) {
	set := []string{}
	args := []any{in.ID}
	pos := 2

	if in.Name != nil {
		set = append(set, fmt.Sprintf("name = $%d", pos))
		args = append(args, *in.Name)
		pos++
	}
	if in.Bio != nil {
		set = append(set, fmt.Sprintf("bio = $%d", pos))
		args = append(args, *in.Bio)
		pos++
	}
	if in.LocationText != nil {
		set = append(set, fmt.Sprintf("location_text = $%d", pos))
		args = append(args, *in.LocationText)
		pos++
	}
	if in.PreferredPickupInstructions != nil {
		set = append(set, fmt.Sprintf("preferred_pickup_instructions = $%d", pos))
		args = append(args, *in.PreferredPickupInstructions)
		pos++
	}

	if len(set) == 0 {
		return r.GetByID(ctx, in.ID)
	}

	query := "UPDATE users SET " + strings.Join(set, ", ") + ", updated_at = NOW() WHERE id = $1"
	if _, err := r.DB.ExecContext(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}
	return r.GetByID(ctx, in.ID)
}

func (r *repository) LinkGoogleAccount(ctx context.Context, id uuid.UUID, googleID, avatarURL string) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE users SET google_id = $2, avatar_url = $3, is_verified = TRUE, updated_at = NOW() WHERE id = $1`,
		id, googleID, avatarURL)
	return err
}

func (r *repository) UpdateStripeCustomerID(ctx context.Context, id uuid.UUID, stripeCustomerID string) error {
	res, err := r.DB.ExecContext(ctx,
		`UPDATE users SET stripe_customer_id = $2, updated_at = NOW() WHERE id = $1`,
		id, stripeCustomerID)
	if err != nil {
		return fmt.Errorf("failed to update stripe customer id: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return commonconstants.ErrNotFound
	}
	return nil
}

func (r *repository) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	_, err := r.DB.ExecContext(ctx,
		`UPDATE users SET last_login_at = $2 WHERE id = $1`, id, time.Now())
	return err
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return commonconstants.ErrNotFound
	}
	return nil
}
