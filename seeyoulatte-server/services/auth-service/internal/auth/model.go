package auth

import (
	"time"

	"github.com/google/uuid"
)

// User mirrors the auth-service users table (carries over the SeeYouLatte
// marketplace fields that the monolith already had: bio, location, frozen
// state, google_id, stripe customer linkage, etc.).
//
// JSON tag "-" keeps the password hash out of any responses.
type User struct {
	ID                          uuid.UUID  `db:"id" json:"id"`
	Email                       string     `db:"email" json:"email"`
	PasswordHash                *string    `db:"password_hash" json:"-"`
	Name                        string     `db:"name" json:"name"`
	Bio                         *string    `db:"bio" json:"bio,omitempty"`
	LocationText                *string    `db:"location_text" json:"location_text,omitempty"`
	IsFrozen                    bool       `db:"is_frozen" json:"is_frozen"`
	GoogleID                    *string    `db:"google_id" json:"-"`
	AvatarURL                   *string    `db:"avatar_url" json:"avatar_url,omitempty"`
	IsVerified                  bool       `db:"is_verified" json:"is_verified"`
	PreferredPickupInstructions *string    `db:"preferred_pickup_instructions" json:"preferred_pickup_instructions,omitempty"`
	StripeCustomerID            *string    `db:"stripe_customer_id" json:"stripe_customer_id,omitempty"`
	CreatedAt                   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt                   time.Time  `db:"updated_at" json:"updated_at"`
	LastLoginAt                 *time.Time `db:"last_login_at" json:"last_login_at,omitempty"`
}

type SignUpInput struct {
	Email    string
	Password string
	Name     string
}

type SignInInput struct {
	Email    string
	Password string
}

type GoogleAuthInput struct {
	IDToken string
}

type UpdateProfileInput struct {
	ID                          uuid.UUID
	Name                        *string
	Bio                         *string
	LocationText                *string
	PreferredPickupInstructions *string
}

type AuthTokens struct {
	User             *User
	AccessToken      string
	RefreshToken     string
	AccessExpiresIn  time.Duration
	RefreshExpiresIn time.Duration
}

type GoogleUserInfo struct {
	ID            string
	Email         string
	VerifiedEmail bool
	Name          string
	Picture       string
}
