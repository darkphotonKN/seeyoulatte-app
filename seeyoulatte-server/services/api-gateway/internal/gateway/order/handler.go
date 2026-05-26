package ordergw

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Handler is the gateway's HTTP shim for order-service. Mirrors the monolith's
// previous order endpoints exactly.
type Handler struct {
	client *Client
}

func NewHandler(client *Client) *Handler {
	return &Handler{client: client}
}

func (h *Handler) CreateOrder(c *gin.Context) {
	buyerID, err := userIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	o, err := h.client.CreateOrder(c.Request.Context(), buyerID, &req)
	if err != nil {
		code := httpStatusFromGRPC(err)
		c.JSON(code, gin.H{"error": fmt.Sprintf("create order failed: %s", err.Error())})
		return
	}
	c.JSON(http.StatusCreated, o)
}

func (h *Handler) GetAllOrders(c *gin.Context) {
	orders, err := h.client.ListOrders(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get orders"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"orders": orders, "count": len(orders)})
}

func (h *Handler) UpdateOrder(c *gin.Context) {
	userID, err := userIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}
	var req UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	o, err := h.client.UpdateOrder(c.Request.Context(), id, userID, &req)
	if err != nil {
		c.JSON(httpStatusFromGRPC(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, o)
}

func (h *Handler) DeleteOrder(c *gin.Context) {
	userID, err := userIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}
	if err := h.client.DeleteOrder(c.Request.Context(), id, userID); err != nil {
		c.JSON(httpStatusFromGRPC(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Order deleted successfully"})
}

func userIDFromContext(c *gin.Context) (uuid.UUID, error) {
	val, exists := c.Get("userId")
	if exists {
		if id, ok := val.(uuid.UUID); ok {
			return id, nil
		}
	}
	// Fallback to string form (old monolith key)
	if v, ok := c.Get("user_id"); ok {
		if s, ok := v.(string); ok {
			id, err := uuid.Parse(s)
			if err == nil {
				return id, nil
			}
		}
	}
	return uuid.Nil, fmt.Errorf("authentication required")
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
