package order

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/darkphotonKN/seeyoulatte-app/internal/listing"
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
	GetByIDWithSellerForUpdateTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID) (*listing.ListingWithSeller, error)
	UpdateTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, sellerID uuid.UUID, req *listing.UpdateListingRequest) (*listing.Listing, error)
}

type UserService interface {
	VerifyUserNotFrozen(ctx context.Context, id uuid.UUID) error
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

	// 1. buyer already validated through JWT token, but check that they are not frozen here
	err := s.userService.VerifyUserNotFrozen(ctx, userID)
	if err != nil {
		return nil, err
	}

	err = dbutils.ExecTx(ctx, s.db, func(tx *sqlx.Tx) error {

		// 2. validate the listing exists, quantity sufficient and is not expired and if SELLER is frozen
		// locks both table rows to prevent race condition collision
		l, err := s.listingService.GetByIDWithSellerForUpdateTx(ctx, tx, req.ListingID)
		if err != nil {
			s.logger.Error("failed to get listing", "error", err, "listing_id", req.ListingID)
			return fmt.Errorf("listing not found: %w", err)
		}

		if l.Quantity < req.Quantity {
			s.logger.Error("insufficient listing quantity", "available", l.Quantity, "requested", req.Quantity)
			return fmt.Errorf("insufficient quantity available")
		}

		if l.UserIsFrozen {
			s.logger.Error("Attempting to sell to a seller that is frozen.", "seller_id", l.SellerID)
			return errorutils.ErrUserIsFrozen
		}

		// 3. validate buyer is not the seller
		if l.SellerID == userID {
			s.logger.Error("buyer cannot purchase their own listing", "user_id", userID, "SellerID", l.SellerID)
			return fmt.Errorf("cannot purchase your own listing")
		}

		// --- checks succeeded, start processing ---
		updatedQuantity := l.Quantity - req.Quantity

		// 4. decrement quantity of listing
		_, err = s.listingService.UpdateTx(ctx, tx, l.ID, l.SellerID, &listing.UpdateListingRequest{
			Quantity: &updatedQuantity,
		})

		if err != nil {
			s.logger.Error("Could not decrement the listing.",
				"listing_id", l.ID,
			)
			return err
		}

		// 5. calculate total amount
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
