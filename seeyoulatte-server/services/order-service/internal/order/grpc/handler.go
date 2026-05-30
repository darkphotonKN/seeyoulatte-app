package grpc

import (
	"context"

	pb "github.com/darkphotonKN/seeyoulatte-app/common/api/proto/order"
	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order/domain"
	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order/usecase"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/internal/status"
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
	// 1. Parse + validate request shape (uuid, target state)
	//    Bad input → codes.InvalidArgument, return early.
	//    NOTE: state validation is a design call — see comment below.

	// parse and validate transition
	if !domain.OrderState(req.TargetState).IsValid() {
		return nil, status.Errorf(codes.InvalidArgument, "invalid target_state: %q", req.TargetState)

	}

	// 2. Call the usecase. Pass time.Now() as the injected clock.
	//    order, err := h.transitionUC.Handle(ctx, orderID, target, req.Actor, time.Now())

	// 3. Map errors → gRPC status codes via errors.Is on the domain sentinels.
	//    Use mapError(err) — the helper below — for clean separation.

	// 4. Map the returned *domain.Order → *pb.Order via orderToProto(snap).
	//    The order's Snapshot() is the read-out door.

}

// orderToProto translates a domain order's snapshot to the proto response.
// Pure, mechanical mapping.
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
// One sentinel per domain error variant — no string matching.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	// TODO(you): one case per domain sentinel
	//   - domain.ErrOrderNotFound          → codes.NotFound
	//   - domain.ErrTransitionNotAllowed   → codes.FailedPrecondition
	//   - domain.ErrInvalidQuantity / ErrInvalidAmount / ErrBuyerIsSeller / ErrInvalidID / ErrInvalidState
	//                                      → codes.InvalidArgument
	//   - default                          → codes.Internal
	}
	return nil // placeholder; replace with the switch above
}
