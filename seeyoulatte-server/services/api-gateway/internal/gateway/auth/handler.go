package authgw

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler is the gateway's HTTP shim for auth-service. Each handler mirrors
// the monolith's previous response shape exactly so seeyoulatte-client doesn't
// need to change after the gRPC cutover.
type Handler struct {
	client *Client
}

func NewHandler(client *Client) *Handler {
	return &Handler{client: client}
}

func (h *Handler) SignUp(c *gin.Context) {
	var req SignUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}
	resp, err := h.client.SignUp(c.Request.Context(), &req)
	if err != nil {
		code := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": fmt.Sprintf("signup failed: %s", err.Error())})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) SignIn(c *gin.Context) {
	var req SignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}
	resp, err := h.client.SignIn(c.Request.Context(), &req)
	if err != nil {
		code := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": fmt.Sprintf("signin failed: %s", err.Error())})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GoogleAuth(c *gin.Context) {
	var req GoogleAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}
	resp, err := h.client.GoogleAuth(c.Request.Context(), req.IDToken)
	if err != nil {
		code := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": fmt.Sprintf("google auth failed: %s", err.Error())})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetCurrentUser(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	id, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	u, err := h.client.GetByID(c.Request.Context(), id)
	if err != nil {
		code := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": fmt.Sprintf("user lookup failed: %s", err.Error())})
		return
	}
	c.JSON(http.StatusOK, u)
}

func httpStatusFromGRPC(err error) int {
	if err == nil {
		return http.StatusOK
	}
	s, ok := status.FromError(err)
	if !ok {
		return http.StatusInternalServerError
	}
	switch s.Code() {
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
