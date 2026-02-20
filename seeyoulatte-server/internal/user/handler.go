package user

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Service interface defines what the handler needs from the service
type Service interface {
	SignUp(ctx context.Context, req *SignUpRequest) (*AuthResponse, error)
	SignIn(ctx context.Context, req *SignInRequest) (*AuthResponse, error)
	GoogleAuth(ctx context.Context, idToken string) (*AuthResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GenerateJWT(user *User) (string, error)
}

type Handler struct {
	service Service
	logger  *slog.Logger
}

func NewHandler(service Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

func (h *Handler) SignUp(c *gin.Context) {
	var req SignUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	resp, err := h.service.SignUp(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("signup failed",
			slog.String("email", req.Email),
			slog.String("error", err.Error()))

		// Check for specific errors
		if err.Error() == "user with this email already exists" {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Email already registered",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create account",
		})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) SignIn(c *gin.Context) {
	var req SignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	resp, err := h.service.SignIn(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("signin failed",
			slog.String("email", req.Email),
			slog.String("error", err.Error()))

		// Check for specific errors
		if err.Error() == "invalid credentials" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid email or password",
			})
			return
		}
		if err.Error() == "account is frozen" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Your account has been suspended",
			})
			return
		}
		if err.Error() == "please sign in with Google" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "This account uses Google Sign-In",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to sign in",
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GoogleAuth(c *gin.Context) {
	var req GoogleAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}

	resp, err := h.service.GoogleAuth(c.Request.Context(), req.IDToken)
	if err != nil {
		h.logger.Error("google auth failed",
			slog.String("error", err.Error()))

		if err.Error() == "account is frozen" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Your account has been suspended",
			})
			return
		}

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Google authentication failed",
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetCurrentUser(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Not authenticated",
		})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	user, err := h.service.GetByID(context.Background(), userID)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}