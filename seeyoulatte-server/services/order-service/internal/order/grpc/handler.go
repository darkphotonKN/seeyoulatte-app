package grpc

import (
	"context"
	"errors"
	"log/slog"
	"time"

	pb "github.com/darkphotonKN/seeyoulatte-app/common/api/proto/order"
	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order/domain"
	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order/usecase"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Handler is the inbound (driving) adapter for the order context's gRPC API.
// Translates proto requests into usecase calls; translates domain results +
// errors back to proto + gRPC status codes. NO business logic.
type Handler struct {
	pb.UnimplementedOrderServiceServer
	transitionUC *usecase.TransitionOrderUC
	// other usecases will land here as you add CreateOrderUC etc.
}

func NewHandler(transitionUC *usecase.TransitionOrderUC) *Handler {
	return &Handler{transitionUC: transitionUC}
}

func (h *Handler) TransitionState(ctx context.Context, req *pb.TransitionStateRequest) (*pb.Order, error) {
	// parse and validate order id as UUID
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", err)
	}

	// parse and validate transition
	if !domain.OrderState(req.TargetState).IsValid() {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target_state: %q", req.TargetState)
	}

	// validate actor presence
	if req.Actor == "" {
		return nil, status.Errorf(codes.InvalidArgument, "actor is required")
	}

	// call the usecase for business logic
	order, err := h.transitionUC.Handle(ctx, orderID, domain.OrderState(req.TargetState), req.Actor, time.Now())

	if err != nil {
		// map errors to grpc errors
		return nil, mapError(ctx, err)
	}

	// map and return proto from snapshot
	return orderToProto(order.Snapshot()), nil
}

// orderToProto translates a domain order's snapshot to the proto response.
func orderToProto(snap domain.OrderSnapshot) *pb.Order {
	out := &pb.Order{
		Id:        snap.ID.String(),
		ListingId: snap.ListingID.String(),
		BuyerId:   snap.BuyerID.String(),
		SellerId:  snap.SellerID.String(),
		Quantity:  int32(snap.Quantity),
		Amount:    snap.Amount,
		State:     string(snap.State),
		CreatedAt: timestamppb.New(snap.CreatedAt),
	}
	if snap.SellerRespondBy != nil {
		out.SellerRespondBy = timestamppb.New(*snap.SellerRespondBy)
	}
	if snap.ReviewEndsAt != nil {
		out.ReviewEndsAt = timestamppb.New(*snap.ReviewEndsAt)
	}
	return out
}

// mapError translates domain/usecase errors into gRPC status codes.
func mapError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	// expected client outcome, no log
	if errors.Is(err, domain.ErrOrderNotFound) {
		return status.Error(codes.NotFound, err.Error())
	}

	// expected business outcome, no log
	if errors.Is(err, domain.ErrTransitionNotAllowed) {
		return status.Error(codes.FailedPrecondition, err.Error())
	}

	// expected failed validation, expected client error, no log
	if errors.Is(err, domain.ErrInvalidAmount) || errors.Is(err, domain.ErrBuyerIsSeller) || errors.Is(err, domain.ErrInvalidQuantity) {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// unexpected state for domain when attempting to reconstitute
	if errors.Is(err, domain.ErrInvalidID) || errors.Is(err, domain.ErrInvalidState) {
		slog.ErrorContext(ctx, "data integrity error when reconstituting order",
			"error", err.Error(),
		)
		return status.Error(codes.Internal, "internal data error")
	}

	slog.ErrorContext(ctx, "unexpected error in TransitionState",
		"err", err.Error(),
	)
	return status.Error(codes.Internal, "internal error")
}
