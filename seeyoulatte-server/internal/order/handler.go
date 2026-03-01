package order

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/darkphotonKN/seeyoulatte-app/internal/utils/errorutils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Service interface {
	Create(ctx context.Context, buyerID uuid.UUID, req *CreateOrderRequest) (*Order, error)
	GetAll(ctx context.Context) ([]Order, error)
	Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *UpdateOrderRequest) (*Order, error)
	Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
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

func (h *Handler) CreateOrder(c *gin.Context) {
	// Extract authenticated user ID from context
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Convert user_id to UUID
	var buyerID uuid.UUID
	switch v := userIDValue.(type) {
	case string:
		parsedID, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
			return
		}
		buyerID = parsedID
	case uuid.UUID:
		buyerID = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.service.Create(c.Request.Context(), buyerID, &req)
	if err != nil {

		if errors.Is(err, errorutils.ErrBuyerIsFrozen) {
			h.logger.Error("Buyer is frozen but attempted purchase",
				slog.String("error", err.Error()),
				slog.String("buyer_id", buyerID.String()))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Attempted to buy when user, the buyer, is frozen."})
			return
		}

		if errors.Is(err, errorutils.ErrSellerIsFrozen) {
			h.logger.Error("Seller is frozen but attempted purchase",
				slog.String("error", err.Error()))
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Attempted to buy from a frozen seller"})
			return
		}

		h.logger.Error("failed to create order",
			slog.String("error", err.Error()),
			slog.String("buyer_id", buyerID.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *Handler) GetAllOrders(c *gin.Context) {
	orders, err := h.service.GetAll(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to get orders",
			slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"count":  len(orders),
	})
}

func (h *Handler) UpdateOrder(c *gin.Context) {
	// Extract authenticated user ID from context
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Convert user_id to UUID
	var userID uuid.UUID
	switch v := userIDValue.(type) {
	case string:
		parsedID, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
			return
		}
		userID = parsedID
	case uuid.UUID:
		userID = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var req UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.service.Update(c.Request.Context(), id, userID, &req)
	if err != nil {
		h.logger.Error("failed to update order",
			slog.String("error", err.Error()),
			slog.String("order_id", id.String()),
			slog.String("user_id", userID.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *Handler) DeleteOrder(c *gin.Context) {
	// Extract authenticated user ID from context
	userIDValue, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	// Convert user_id to UUID
	var userID uuid.UUID
	switch v := userIDValue.(type) {
	case string:
		parsedID, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
			return
		}
		userID = parsedID
	case uuid.UUID:
		userID = v
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID type"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	err = h.service.Delete(c.Request.Context(), id, userID)
	if err != nil {
		h.logger.Error("failed to delete order",
			slog.String("error", err.Error()),
			slog.String("order_id", id.String()),
			slog.String("user_id", userID.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order deleted successfully"})
}
