package order

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/darkphotonKN/seeyoulatte-app/internal/listing"
	"github.com/darkphotonKN/seeyoulatte-app/internal/user"
	dbutils "github.com/darkphotonKN/seeyoulatte-app/internal/utils/db"
	"github.com/darkphotonKN/seeyoulatte-app/internal/utils/errorutils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	Create(ctx context.Context, order *Order) error
	GetAll(ctx context.Context) ([]Order, error)
	Update(ctx context.Context, order *Order) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ListingService interface {
	GetByIDLock(ctx context.Context, id uuid.UUID) (*listing.Listing, error)
	Update(ctx context.Context, id uuid.UUID, sellerID uuid.UUID, req *listing.UpdateListingRequest) (*listing.Listing, error)
}

type UserService interface {
	GetByIDLock(ctx context.Context, id uuid.UUID) (*user.User, error)
}

type service struct {
	repo           Repository
	db             *sqlx.DB
	listingService ListingService
	userService    UserService
	logger         *slog.Logger
}

func NewService(repo Repository, db *sqlx.DB, logger *slog.Logger, listingService ListingService, userService UserService) *service {
	return &service{
		repo:           repo,
		db:             db,
		listingService: listingService,
		userService:    userService,
		logger:         logger,
	}
}

func (s *service) Create(ctx context.Context, userID uuid.UUID, req *CreateOrderRequest) (*Order, error) {
	var order *Order

	err := dbutils.ExecTx(ctx, s.db, func(tx *sqlx.Tx) error {
		// 1. validate l exists, quantity sufficient and is not expired
		l, err := s.listingService.GetByIDLock(ctx, req.ListingID)
		if err != nil {
			s.logger.Error("failed to get listing", "error", err, "listing_id", req.ListingID)
			return fmt.Errorf("listing not found: %w", err)
		}

		if l.Quantity < req.Quantity {
			s.logger.Error("insufficient listing quantity", "available", l.Quantity, "requested", req.Quantity)
			return fmt.Errorf("insufficient quantity available")
		}

		// 2. validate buyer is not the seller
		if l.SellerID == userID {
			s.logger.Error("buyer cannot purchase their own listing", "user_id", userID, "SellerID", l.SellerID)
			return fmt.Errorf("cannot purchase your own listing")
		}

		// 3. check user is not frozen with lock
		user, err := s.userService.GetByIDLock(ctx, userID)
		if err != nil {
			s.logger.Error("User could not be retrived", "user_id", userID)
			return err
		}

		if user.IsFrozen {
			s.logger.Error("Frozen user attmped to place order on listing.", "user_id", userID)
			return errorutils.ErrUserIsFrozen
		}

		// --- checks succeeded, start processing ---
		updatedQuantity := l.Quantity - 1

		// 4. decrement quantity of listing
		s.listingService.Update(ctx, userID, l.SellerID, &listing.UpdateListingRequest{
			Quantity: &updatedQuantity,
		})

		// 4. calculate total amount
		amount := l.Price * float64(req.Quantity)

		// 4. create the order
		order = &Order{
			ListingID: req.ListingID,
			BuyerID:   userID,
			SellerID:  l.SellerID,
			Quantity:  req.Quantity,
			Amount:    amount,
			State:     "pending_payment",
		}

		if err := s.repo.Create(ctx, order); err != nil {
			return fmt.Errorf("creating order: %w", err)
		}

		// TODO: 5. decrement listing quantity
		s.listingService.Update(ctx, l.ID, l.SellerID)

		// TODO: 6. insert ESCROW ledger entry

		s.logger.Info("order created",
			slog.String("order_id", order.ID.String()),
			slog.String("buyer_id", userID.String()),
			slog.String("seller_id", l.SellerID.String()))

		return nil
	})

	if err != nil {
		s.logger.Error("transaction failed, rolled back", "error", err, "buyer_id", userID, "listing_id", req.ListingID)
		return nil, err
	}

	return order, nil
}

func (s *service) GetAll(ctx context.Context) ([]Order, error) {
	orders, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting orders: %w", err)
	}
	return orders, nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *UpdateOrderRequest) (*Order, error) {
	// TODO: Add validation to ensure user is either buyer or seller of this order
	// For now, just update as before
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
		slog.String("order_id", id.String()),
		slog.String("user_id", userID.String()))

	return order, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// TODO: Add validation to ensure user has permission to delete this order
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting order: %w", err)
	}

	s.logger.Info("order deleted",
		slog.String("order_id", id.String()),
		slog.String("user_id", userID.String()))

	return nil
}
