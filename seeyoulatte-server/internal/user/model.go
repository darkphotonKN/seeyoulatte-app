package user

import (
	"time"
	"github.com/google/uuid"
)

type User struct {
	ID                        uuid.UUID  `db:"id" json:"id"`
	Email                     string     `db:"email" json:"email"`
	PasswordHash              *string    `db:"password_hash" json:"-"`
	Name                      string     `db:"name" json:"name"`
	Bio                       *string    `db:"bio" json:"bio,omitempty"`
	LocationText              *string    `db:"location_text" json:"location_text,omitempty"`
	IsFrozen                  bool       `db:"is_frozen" json:"is_frozen"`
	GoogleID                  *string    `db:"google_id" json:"-"`
	AvatarURL                 *string    `db:"avatar_url" json:"avatar_url,omitempty"`
	IsVerified                bool       `db:"is_verified" json:"is_verified"`
	PreferredPickupInstructions *string  `db:"preferred_pickup_instructions" json:"preferred_pickup_instructions,omitempty"`
	CreatedAt                 time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt                 time.Time  `db:"updated_at" json:"updated_at"`
	LastLoginAt               *time.Time `db:"last_login_at" json:"last_login_at,omitempty"`
}

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
	User  *User  `json:"user"`
	Token string `json:"token"`
}

type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
}