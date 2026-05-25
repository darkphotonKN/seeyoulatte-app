package authgw

import (
	"time"

	"github.com/google/uuid"
)

// User mirrors the auth-service's user shape over HTTP/JSON so external
// clients (seeyoulatte-client) see no change after the gRPC cutover.
type User struct {
	ID                          uuid.UUID  `json:"id"`
	Email                       string     `json:"email"`
	Name                        string     `json:"name"`
	Bio                         *string    `json:"bio,omitempty"`
	LocationText                *string    `json:"location_text,omitempty"`
	IsFrozen                    bool       `json:"is_frozen"`
	AvatarURL                   *string    `json:"avatar_url,omitempty"`
	IsVerified                  bool       `json:"is_verified"`
	PreferredPickupInstructions *string    `json:"preferred_pickup_instructions,omitempty"`
	StripeCustomerID            *string    `json:"stripe_customer_id,omitempty"`
	CreatedAt                   time.Time  `json:"created_at"`
	UpdatedAt                   time.Time  `json:"updated_at"`
	LastLoginAt                 *time.Time `json:"last_login_at,omitempty"`
}

// HTTP request/response shapes — kept 1:1 with the monolith's previous user API
// so external clients see no change after the gRPC cutover.

type SignUpRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"required,min=1,max=255"`
}

type SignInRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type GoogleAuthRequest struct {
	IDToken string `json:"id_token" binding:"required"`
}

type AuthResponse struct {
	User             *User  `json:"user"`
	AccessToken      string `json:"token"`
	RefreshToken     string `json:"refresh_token,omitempty"`
	AccessExpiresIn  int64  `json:"access_expires_in,omitempty"`
	RefreshExpiresIn int64  `json:"refresh_expires_in,omitempty"`
}

type UpdateProfileRequest struct {
	Name                        *string `json:"name,omitempty"`
	Bio                         *string `json:"bio,omitempty"`
	LocationText                *string `json:"location_text,omitempty"`
	PreferredPickupInstructions *string `json:"preferred_pickup_instructions,omitempty"`
}
