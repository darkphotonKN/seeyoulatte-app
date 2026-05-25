package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	commonauth "github.com/darkphotonKN/seeyoulatte-app/common/auth"
	commonbroker "github.com/darkphotonKN/seeyoulatte-app/common/broker"
	commonconstants "github.com/darkphotonKN/seeyoulatte-app/common/constants"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTTL  = time.Hour
	refreshTTL = 24 * 7 * time.Hour
)

type Repository interface {
	Create(ctx context.Context, u *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByGoogleID(ctx context.Context, googleID string) (*User, error)
	GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*User, error)
	ListAll(ctx context.Context) ([]*User, error)
	UpdateProfile(ctx context.Context, in *UpdateProfileInput) (*User, error)
	LinkGoogleAccount(ctx context.Context, id uuid.UUID, googleID, avatarURL string) error
	UpdateStripeCustomerID(ctx context.Context, id uuid.UUID, stripeCustomerID string) error
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo           Repository
	publishCh      commonbroker.Publisher
	jwtSecret      string
	googleClientID string
}

func NewService(repo Repository, publishCh commonbroker.Publisher, jwtSecret, googleClientID string) *service {
	return &service{repo: repo, publishCh: publishCh, jwtSecret: jwtSecret, googleClientID: googleClientID}
}

func (s *service) SignUp(ctx context.Context, in *SignUpInput) (*AuthTokens, error) {
	if in.Email == "" || in.Password == "" || in.Name == "" {
		return nil, commonconstants.ErrInvalidInput
	}

	// Reject if email already in use
	if existing, err := s.repo.GetByEmail(ctx, in.Email); err == nil && existing != nil {
		return nil, commonconstants.ErrDuplicateResource
	} else if err != nil && !errors.Is(err, commonconstants.ErrNotFound) {
		return nil, err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	hashedStr := string(hashed)

	u := &User{
		Email:        in.Email,
		Name:         in.Name,
		PasswordHash: &hashedStr,
		IsVerified:   false,
		IsFrozen:     false,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}

	_ = s.repo.UpdateLastLogin(ctx, u.ID)
	s.PublishUserCreated(ctx, u)

	return s.issueTokens(u)
}

func (s *service) SignIn(ctx context.Context, in *SignInInput) (*AuthTokens, error) {
	u, err := s.repo.GetByEmail(ctx, in.Email)
	if err != nil {
		if errors.Is(err, commonconstants.ErrNotFound) {
			return nil, commonconstants.ErrUnauthorized
		}
		return nil, err
	}
	if u.IsFrozen {
		return nil, commonconstants.ErrForbidden
	}
	if u.PasswordHash == nil {
		return nil, fmt.Errorf("please sign in with Google")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(in.Password)); err != nil {
		return nil, commonconstants.ErrUnauthorized
	}
	_ = s.repo.UpdateLastLogin(ctx, u.ID)
	return s.issueTokens(u)
}

func (s *service) GoogleAuth(ctx context.Context, in *GoogleAuthInput) (*AuthTokens, error) {
	googleUser, err := s.verifyGoogleIDToken(in.IDToken)
	if err != nil {
		return nil, fmt.Errorf("verifying google token: %w", err)
	}

	// Try Google ID first
	u, err := s.repo.GetByGoogleID(ctx, googleUser.ID)
	if err != nil && !errors.Is(err, commonconstants.ErrNotFound) {
		return nil, err
	}

	// If not found by Google ID, try email — link account if existing user
	if u == nil {
		u, err = s.repo.GetByEmail(ctx, googleUser.Email)
		if err != nil && !errors.Is(err, commonconstants.ErrNotFound) {
			return nil, err
		}
		if u != nil {
			if err := s.repo.LinkGoogleAccount(ctx, u.ID, googleUser.ID, googleUser.Picture); err != nil {
				return nil, fmt.Errorf("linking google account: %w", err)
			}
			// Refresh u to pick up linked fields
			u, err = s.repo.GetByID(ctx, u.ID)
			if err != nil {
				return nil, err
			}
		}
	}

	// Still no user — create a new account from Google profile
	if u == nil {
		gid := googleUser.ID
		pic := googleUser.Picture
		u = &User{
			Email:      googleUser.Email,
			Name:       googleUser.Name,
			GoogleID:   &gid,
			AvatarURL:  &pic,
			IsVerified: true,
			IsFrozen:   false,
		}
		if err := s.repo.Create(ctx, u); err != nil {
			return nil, fmt.Errorf("creating user from google: %w", err)
		}
		s.PublishUserCreated(ctx, u)
	}

	if u.IsFrozen {
		return nil, commonconstants.ErrForbidden
	}

	_ = s.repo.UpdateLastLogin(ctx, u.ID)
	return s.issueTokens(u)
}

func (s *service) Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error) {
	userID, err := commonauth.ValidateRefreshToken(refreshToken, s.jwtSecret)
	if err != nil {
		return nil, commonconstants.ErrUnauthorized
	}
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u.IsFrozen {
		return nil, commonconstants.ErrForbidden
	}
	return s.issueTokens(u)
}

func (s *service) GetUser(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) ListUsers(ctx context.Context) ([]*User, error) {
	return s.repo.ListAll(ctx)
}

func (s *service) UpdateProfile(ctx context.Context, in *UpdateProfileInput) (*User, error) {
	if in.Name != nil && *in.Name == "" {
		return nil, errors.New("name cannot be empty")
	}
	u, err := s.repo.UpdateProfile(ctx, in)
	if err != nil {
		return nil, err
	}
	s.PublishUserUpdated(ctx, u)
	return u, nil
}

func (s *service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.PublishUserDeleted(ctx, id)
	return nil
}

// VerifyUserNotFrozen returns ErrForbidden if the user is frozen, ErrNotFound
// if missing, or nil if the user exists and isn't frozen. Used by order-service
// when creating an order to keep the freeze check inside the auth boundary.
func (s *service) VerifyUserNotFrozen(ctx context.Context, id uuid.UUID) error {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if u.IsFrozen {
		return commonconstants.ErrForbidden
	}
	return nil
}

func (s *service) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*User, error) {
	return s.repo.GetByStripeCustomerID(ctx, stripeCustomerID)
}

func (s *service) UpdateStripeCustomerID(ctx context.Context, userID uuid.UUID, stripeCustomerID string) error {
	return s.repo.UpdateStripeCustomerID(ctx, userID, stripeCustomerID)
}

func (s *service) issueTokens(u *User) (*AuthTokens, error) {
	access, err := commonauth.GenerateJWT(u.ID, commonauth.TokenTypeAccess, s.jwtSecret, accessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := commonauth.GenerateJWT(u.ID, commonauth.TokenTypeRefresh, s.jwtSecret, refreshTTL)
	if err != nil {
		return nil, err
	}
	return &AuthTokens{
		User:             u,
		AccessToken:      access,
		RefreshToken:     refresh,
		AccessExpiresIn:  accessTTL,
		RefreshExpiresIn: refreshTTL,
	}, nil
}

// verifyGoogleIDToken validates a Google ID token via Google's tokeninfo endpoint
// and returns the parsed profile info. Matches the legacy monolith behavior.
func (s *service) verifyGoogleIDToken(idToken string) (*GoogleUserInfo, error) {
	resp, err := http.Get(fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", idToken))
	if err != nil {
		return nil, fmt.Errorf("verifying token with google: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid google token")
	}

	var tokenInfo struct {
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Sub           string `json:"sub"`
		Aud           string `json:"aud"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, fmt.Errorf("decoding token info: %w", err)
	}

	if s.googleClientID != "" && tokenInfo.Aud != s.googleClientID {
		return nil, errors.New("invalid token audience")
	}

	return &GoogleUserInfo{
		ID:            tokenInfo.Sub,
		Email:         tokenInfo.Email,
		VerifiedEmail: tokenInfo.EmailVerified == "true",
		Name:          tokenInfo.Name,
		Picture:       tokenInfo.Picture,
	}, nil
}
