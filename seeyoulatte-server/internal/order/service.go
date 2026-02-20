package order

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/darkphotonKN/seeyoulatte-app/internal/listing"
	"github.com/darkphotonKN/seeyoulatte-app/internal/user"
	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, order *Order) error
	GetAll(ctx context.Context) ([]Order, error)
	Update(ctx context.Context, order *Order) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ListingService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*listing.Listing, error)
}

type UserService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*user.User, error)
}

type service struct {
	repo           Repository
	listingService ListingService
	userService    UserService
	logger         *slog.Logger
}

func NewService(repo Repository, logger *slog.Logger, listingService ListingService, userService UserService) *service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

func (s *service) Create(ctx context.Context, req *CreateOrderRequest) (*Order, error) {
	// 1. validate listing exists, quantity sufficient and is not exired

	// 2. validate seller is not frozen or trying to buy their own product

	// 3. calculate total amount

	// 4. decrement quantity

	// 5. insert row, state update to pending_payment

	// 6. insert ESCROW ledger

	order := &Order{
		ListingID: req.ListingID,
		BuyerID:   req.BuyerID,
		SellerID:  req.SellerID,
		Quantity:  req.Quantity,
		Amount:    req.Amount,
		State:     "pending_payment",
	}

	if err := s.repo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("creating order: %w", err)
	}

	s.logger.Info("order created",
		slog.String("order_id", order.ID.String()),
		slog.String("buyer_id", order.BuyerID.String()),
		slog.String("seller_id", order.SellerID.String()))

	return order, nil
}

func (s *service) GetAll(ctx context.Context) ([]Order, error) {
	orders, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting orders: %w", err)
	}
	return orders, nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, req *UpdateOrderRequest) (*Order, error) {
	order := &Order{
		ID: id,
	}

	if req.State != nil {
		order.State = *req.State
	}
	if req.SellerRespondBy != nil {
		order.SellerRespondBy = req.SellerRespondBy
	}
	if req.ReviewEndsAt != nil {
		order.ReviewEndsAt = req.ReviewEndsAt
	}

	if err := s.repo.Update(ctx, order); err != nil {
		return nil, fmt.Errorf("updating order: %w", err)
	}

	s.logger.Info("order updated",
		slog.String("order_id", id.String()))

	return order, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting order: %w", err)
	}

	s.logger.Info("order deleted",
		slog.String("order_id", id.String()))

	return nil
}

