package listing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, listing *Listing) error
	GetByID(ctx context.Context, id uuid.UUID) (*Listing, error)
	GetByIDLock(ctx context.Context, id uuid.UUID) (*Listing, error)
	GetAllPublic(ctx context.Context) ([]Listing, error)
	GetBySellerID(ctx context.Context, sellerID uuid.UUID) ([]Listing, error)
	Update(ctx context.Context, listing *Listing) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo   Repository
	logger *slog.Logger
}

func NewService(repo Repository, logger *slog.Logger) *service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

func (s *service) Create(ctx context.Context, sellerID uuid.UUID, req *CreateListingRequest) (*Listing, error) {
	listing := &Listing{
		SellerID:           sellerID,
		Title:              req.Title,
		Description:        req.Description,
		Category:           req.Category,
		Price:              req.Price,
		Quantity:           req.Quantity,
		PickupInstructions: req.PickupInstructions,
		ExpiresAt:          req.ExpiresAt,
		IsActive:           true,
	}

	if err := s.repo.Create(ctx, listing); err != nil {
		return nil, fmt.Errorf("creating listing: %w", err)
	}

	s.logger.Info("listing created",
		slog.String("listing_id", listing.ID.String()),
		slog.String("seller_id", sellerID.String()),
		slog.String("title", listing.Title))

	return listing, nil
}

func (s *service) GetByID(ctx context.Context, id uuid.UUID) (*Listing, error) {
	listing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting listing: %w", err)
	}
	if listing == nil {
		return nil, errors.New("listing not found")
	}
	return listing, nil
}

func (s *service) GetByIDLock(ctx context.Context, id uuid.UUID) (*Listing, error) {
	listing, err := s.repo.GetByIDLock(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting listing: %w", err)
	}
	if listing == nil {
		return nil, errors.New("listing not found")
	}
	return listing, nil
}

func (s *service) GetAllPublic(ctx context.Context) ([]Listing, error) {
	listings, err := s.repo.GetAllPublic(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting public listings: %w", err)
	}
	return listings, nil
}

func (s *service) GetMyListings(ctx context.Context, sellerID uuid.UUID) ([]Listing, error) {
	listings, err := s.repo.GetBySellerID(ctx, sellerID)
	if err != nil {
		return nil, fmt.Errorf("getting user listings: %w", err)
	}
	return listings, nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, sellerID uuid.UUID, req *UpdateListingRequest) (*Listing, error) {
	// Get existing listing
	listing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting listing: %w", err)
	}
	if listing == nil {
		return nil, errors.New("listing not found")
	}

	// Check ownership
	if listing.SellerID != sellerID {
		return nil, errors.New("unauthorized: you can only update your own listings")
	}

	// Update fields if provided
	if req.Title != nil {
		listing.Title = *req.Title
	}
	if req.Description != nil {
		listing.Description = req.Description
	}
	if req.Price != nil {
		if *req.Price < 0.01 {
			return nil, errors.New("price must be at least 0.01")
		}
		listing.Price = *req.Price
	}
	if req.Quantity != nil {
		if *req.Quantity < 0 {
			return nil, errors.New("quantity cannot be negative")
		}
		listing.Quantity = *req.Quantity
	}
	if req.PickupInstructions != nil {
		listing.PickupInstructions = req.PickupInstructions
	}
	if req.IsActive != nil {
		listing.IsActive = *req.IsActive
	}
	if req.ExpiresAt != nil {
		listing.ExpiresAt = req.ExpiresAt
	}

	// Save updates
	if err := s.repo.Update(ctx, listing); err != nil {
		return nil, fmt.Errorf("updating listing: %w", err)
	}

	s.logger.Info("listing updated",
		slog.String("listing_id", id.String()),
		slog.String("seller_id", sellerID.String()))

	return listing, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID, sellerID uuid.UUID) error {
	// Get listing to check ownership
	listing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("getting listing: %w", err)
	}
	if listing == nil {
		return errors.New("listing not found")
	}

	// Check ownership
	if listing.SellerID != sellerID {
		return errors.New("unauthorized: you can only delete your own listings")
	}

	// Delete listing
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting listing: %w", err)
	}

	s.logger.Info("listing deleted",
		slog.String("listing_id", id.String()),
		slog.String("seller_id", sellerID.String()))

	return nil
}
