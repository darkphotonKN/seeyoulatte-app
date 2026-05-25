package listing

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Service interface defines what the handler needs from the service
type Service interface {
	Create(ctx context.Context, sellerID uuid.UUID, req *CreateListingRequest) (*Listing, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Listing, error)
	GetAllPublic(ctx context.Context) ([]Listing, error)
	GetMyListings(ctx context.Context, sellerID uuid.UUID) ([]Listing, error)
	Update(ctx context.Context, id uuid.UUID, sellerID uuid.UUID, req *UpdateListingRequest) (*Listing, error)
	Delete(ctx context.Context, id uuid.UUID, sellerID uuid.UUID) error
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

// CreateListing - POST /api/listings (requires auth)
func (h *Handler) CreateListing(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req CreateListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	listing, err := h.service.Create(c.Request.Context(), userID, &req)
	if err != nil {
		h.logger.Error("failed to create listing",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create listing"})
		return
	}

	c.JSON(http.StatusCreated, listing)
}

// GetListing - GET /api/listings/:id (public)
func (h *Handler) GetListing(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid listing ID"})
		return
	}

	listing, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		if err.Error() == "listing not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found"})
			return
		}
		h.logger.Error("failed to get listing",
			slog.String("error", err.Error()),
			slog.String("listing_id", id.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get listing"})
		return
	}

	c.JSON(http.StatusOK, listing)
}

// GetAllListings - GET /api/listings (public)
func (h *Handler) GetAllListings(c *gin.Context) {
	listings, err := h.service.GetAllPublic(c.Request.Context())
	if err != nil {
		h.logger.Error("failed to get listings",
			slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get listings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"listings": listings,
		"count":    len(listings),
	})
}

// GetMyListings - GET /api/listings/my (requires auth)
func (h *Handler) GetMyListings(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	listings, err := h.service.GetMyListings(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get user listings",
			slog.String("error", err.Error()),
			slog.String("user_id", userID.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get your listings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"listings": listings,
		"count":    len(listings),
	})
}

// UpdateListing - PUT /api/listings/:id (requires auth & ownership)
func (h *Handler) UpdateListing(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid listing ID"})
		return
	}

	var req UpdateListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	listing, err := h.service.Update(c.Request.Context(), id, userID, &req)
	if err != nil {
		if err.Error() == "listing not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found"})
			return
		}
		if err.Error() == "unauthorized: you can only update your own listings" {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own listings"})
			return
		}
		h.logger.Error("failed to update listing",
			slog.String("error", err.Error()),
			slog.String("listing_id", id.String()),
			slog.String("user_id", userID.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update listing"})
		return
	}

	c.JSON(http.StatusOK, listing)
}

// DeleteListing - DELETE /api/listings/:id (requires auth & ownership)
func (h *Handler) DeleteListing(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid listing ID"})
		return
	}

	err = h.service.Delete(c.Request.Context(), id, userID)
	if err != nil {
		if err.Error() == "listing not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Listing not found"})
			return
		}
		if err.Error() == "unauthorized: you can only delete your own listings" {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own listings"})
			return
		}
		h.logger.Error("failed to delete listing",
			slog.String("error", err.Error()),
			slog.String("listing_id", id.String()),
			slog.String("user_id", userID.String()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete listing"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Listing deleted successfully"})
}