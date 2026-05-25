package order

import (
	"context"
	"errors"
	"strings"

	pb "github.com/darkphotonKN/seeyoulatte-app/common/api/proto/order"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service interface {
	Create(ctx context.Context, buyerID, listingID, sellerID uuid.UUID, quantity int, amount float64) (*Order, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Order, error)
	GetAll(ctx context.Context) ([]Order, error)
	GetPendingPaymentOrdersByUser(ctx context.Context, userID uuid.UUID) ([]*Order, error)
	Update(ctx context.Context, id, userID uuid.UUID, req *UpdateOrderRequest) (*Order, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
	TransitionState(ctx context.Context, orderID uuid.UUID, targetState, actor string) (*Order, error)
}

type Handler struct {
	pb.UnimplementedOrderServiceServer
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.Order, error) {
	buyerID, err := uuid.Parse(req.BuyerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid buyer_id: %v", err)
	}
	listingID, err := uuid.Parse(req.ListingId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid listing_id: %v", err)
	}
	sellerID, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid seller_id: %v", err)
	}
	if req.Quantity <= 0 {
		return nil, status.Error(codes.InvalidArgument, "quantity must be > 0")
	}
	if req.Amount <= 0 {
		return nil, status.Error(codes.InvalidArgument, "amount must be > 0")
	}

	o, err := h.service.Create(ctx, buyerID, listingID, sellerID, int(req.Quantity), req.Amount)
	if err != nil {
		return nil, mapError(err)
	}
	return orderToProto(o), nil
}

func (h *Handler) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.Order, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	o, err := h.service.GetByID(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return orderToProto(o), nil
}

func (h *Handler) ListOrders(ctx context.Context, _ *pb.ListOrdersRequest) (*pb.ListOrdersResponse, error) {
	orders, err := h.service.GetAll(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*pb.Order, 0, len(orders))
	for i := range orders {
		out = append(out, orderToProto(&orders[i]))
	}
	return &pb.ListOrdersResponse{Orders: out}, nil
}

func (h *Handler) GetPendingPaymentOrdersByUser(ctx context.Context, req *pb.GetPendingPaymentOrdersByUserRequest) (*pb.ListOrdersResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}
	orders, err := h.service.GetPendingPaymentOrdersByUser(ctx, userID)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*pb.Order, 0, len(orders))
	for _, o := range orders {
		out = append(out, orderToProto(o))
	}
	return &pb.ListOrdersResponse{Orders: out}, nil
}

func (h *Handler) UpdateOrder(ctx context.Context, req *pb.UpdateOrderRequest) (*pb.Order, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}
	in := &UpdateOrderRequest{State: req.State}
	if req.SellerRespondBy != nil {
		t := req.SellerRespondBy.AsTime()
		in.SellerRespondBy = &t
	}
	if req.ReviewEndsAt != nil {
		t := req.ReviewEndsAt.AsTime()
		in.ReviewEndsAt = &t
	}
	o, err := h.service.Update(ctx, id, userID, in)
	if err != nil {
		return nil, mapError(err)
	}
	return orderToProto(o), nil
}

func (h *Handler) DeleteOrder(ctx context.Context, req *pb.DeleteOrderRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id: %v", err)
	}
	if err := h.service.Delete(ctx, id, userID); err != nil {
		return nil, mapError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) TransitionState(ctx context.Context, req *pb.TransitionStateRequest) (*pb.Order, error) {
	id, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", err)
	}
	o, err := h.service.TransitionState(ctx, id, req.TargetState, req.Actor)
	if err != nil {
		return nil, mapError(err)
	}
	return orderToProto(o), nil
}

func orderToProto(o *Order) *pb.Order {
	out := &pb.Order{
		Id:        o.ID.String(),
		ListingId: o.ListingID.String(),
		BuyerId:   o.BuyerID.String(),
		SellerId:  o.SellerID.String(),
		Quantity:  int32(o.Quantity),
		Amount:    o.Amount,
		State:     o.State,
		CreatedAt: timestamppb.New(o.CreatedAt),
	}
	if o.SellerRespondBy != nil {
		out.SellerRespondBy = timestamppb.New(*o.SellerRespondBy)
	}
	if o.ReviewEndsAt != nil {
		out.ReviewEndsAt = timestamppb.New(*o.ReviewEndsAt)
	}
	return out
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case strings.Contains(err.Error(), "not found"):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, errBadInput):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

var errBadInput = errors.New("invalid input")
