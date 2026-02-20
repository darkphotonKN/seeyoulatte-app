package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Repository interface defines what the service needs from the repository
type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByGoogleID(ctx context.Context, googleID string) (*User, error)
	Update(ctx context.Context, user *User) error
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error
}

type service struct {
	repo         Repository
	logger       *slog.Logger
	jwtSecret    []byte
	googleConfig *oauth2.Config
}

func NewService(repo Repository, logger *slog.Logger) *service {
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key-change-in-production"
	}

	googleConfig := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &service{
		repo:         repo,
		logger:       logger,
		jwtSecret:    []byte(jwtSecret),
		googleConfig: googleConfig,
	}
}

func (s *service) SignUp(ctx context.Context, req *SignUpRequest) (*AuthResponse, error) {
	// Check if user already exists
	existingUser, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("checking existing user: %w", err)
	}
	if existingUser != nil {
		return nil, errors.New("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	hashedStr := string(hashedPassword)
	user := &User{
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: &hashedStr,
		IsVerified:   false,
		IsFrozen:     false,
	}

	// Create user
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	// Generate JWT
	token, err := s.GenerateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	// Update last login
	_ = s.repo.UpdateLastLogin(ctx, user.ID)

	return &AuthResponse{
		User:  user,
		Token: token,
	}, nil
}

func (s *service) SignIn(ctx context.Context, req *SignInRequest) (*AuthResponse, error) {
	// Get user by email
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}

	// Check if account is frozen
	if user.IsFrozen {
		return nil, errors.New("account is frozen")
	}

	// Check password
	if user.PasswordHash == nil {
		return nil, errors.New("please sign in with Google")
	}

	err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Generate JWT
	token, err := s.GenerateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	// Update last login
	_ = s.repo.UpdateLastLogin(ctx, user.ID)

	return &AuthResponse{
		User:  user,
		Token: token,
	}, nil
}

func (s *service) GoogleAuth(ctx context.Context, idToken string) (*AuthResponse, error) {
	// Verify the ID token with Google
	googleUser, err := s.verifyGoogleIDToken(idToken)
	if err != nil {
		return nil, fmt.Errorf("verifying Google token: %w", err)
	}

	// Try to find user by Google ID
	user, err := s.repo.GetByGoogleID(ctx, googleUser.ID)
	if err != nil {
		return nil, fmt.Errorf("getting user by Google ID: %w", err)
	}

	// If no user with Google ID, try to find by email
	if user == nil {
		user, err = s.repo.GetByEmail(ctx, googleUser.Email)
		if err != nil {
			return nil, fmt.Errorf("getting user by email: %w", err)
		}

		// If user exists with email, link Google account
		if user != nil {
			user.GoogleID = &googleUser.ID
			user.AvatarURL = &googleUser.Picture
			user.IsVerified = true // Google accounts are verified
			if err := s.repo.Update(ctx, user); err != nil {
				return nil, fmt.Errorf("linking Google account: %w", err)
			}
		}
	}

	// If still no user, create new account
	if user == nil {
		user = &User{
			Email:      googleUser.Email,
			Name:       googleUser.Name,
			GoogleID:   &googleUser.ID,
			AvatarURL:  &googleUser.Picture,
			IsVerified: true, // Google accounts are verified
			IsFrozen:   false,
		}

		if err := s.repo.Create(ctx, user); err != nil {
			return nil, fmt.Errorf("creating user from Google: %w", err)
		}
	}

	// Check if account is frozen
	if user.IsFrozen {
		return nil, errors.New("account is frozen")
	}

	// Generate JWT
	token, err := s.GenerateJWT(user)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	// Update last login
	_ = s.repo.UpdateLastLogin(ctx, user.ID)

	return &AuthResponse{
		User:  user,
		Token: token,
	}, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *service) GenerateJWT(user *User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"name":    user.Name,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *service) verifyGoogleIDToken(idToken string) (*GoogleUserInfo, error) {
	// Verify token with Google's tokeninfo endpoint
	resp, err := http.Get(fmt.Sprintf("https://oauth2.googleapis.com/tokeninfo?id_token=%s", idToken))
	if err != nil {
		return nil, fmt.Errorf("verifying token with Google: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("invalid Google token")
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

	// Verify the audience matches our client ID
	expectedClientID := os.Getenv("GOOGLE_CLIENT_ID")
	if expectedClientID != "" && tokenInfo.Aud != expectedClientID {
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

