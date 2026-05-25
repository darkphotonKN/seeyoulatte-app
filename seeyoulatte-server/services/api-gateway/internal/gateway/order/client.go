package ordergw

import (
	"context"
	"sync"

	pb "github.com/darkphotonKN/seeyoulatte-app/common/api/proto/order"
	"github.com/darkphotonKN/seeyoulatte-app/common/discovery"
	"github.com/darkphotonKN/seeyoulatte-app/services/api-gateway/internal/listing"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const targetService = "order"

// Client wraps a long-lived gRPC connection to order-service.
type Client struct {
	registry       discovery.Registry
	listingService ListingLookup
	mu             sync.Mutex
	conn           *grpc.ClientConn
}

// ListingLookup is the slice of the listing service the order gateway needs.
// CreateOrder accepts only listing_id + quantity from the HTTP client; the
// gateway resolves the listing locally (since listing still lives in the
// api-gateway monolith) and passes seller_id + amount into the gRPC call.
type ListingLookup interface {
	GetByID(ctx context.Context, id uuid.UUID) (*listing.Listing, error)
}

func NewClient(registry discovery.Registry, listingService ListingLookup) *Client {
	return &Client{registry: registry, listingService: listingService}
}

func (c *Client) connClient(ctx context.Context) (pb.OrderServiceClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && c.conn.GetState() != connectivity.Shutdown {
		return pb.NewOrderServiceClient(c.conn), nil
	}

	conn, err := discovery.ServiceConnection(ctx, targetService, c.registry)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	return pb.NewOrderServiceClient(conn), nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}

// CreateOrder looks up the listing locally, validates quantity, then forwards
// the pre-validated data to order-service. The listing freeze-check on the
// seller can't be done here once the user table moves to auth-service — the
// order-service is expected to do its own buyer-frozen check via auth-service.
func (c *Client) CreateOrder(ctx context.Context, buyerID uuid.UUID, req *CreateOrderRequest) (*Order, error) {
	l, err := c.listingService.GetByID(ctx, req.ListingID)
	if err != nil {
		return nil, err
	}
	amount := l.Price * float64(req.Quantity)

	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
	o, err := client.CreateOrder(ctx, &pb.CreateOrderRequest{
		BuyerId:   buyerID.String(),
		ListingId: req.ListingID.String(),
		SellerId:  l.SellerID.String(),
		Quantity:  int32(req.Quantity),
		Amount:    amount,
	})
	if err != nil {
		return nil, err
	}
	return orderFromProto(o), nil
}

func (c *Client) GetByID(ctx context.Context, id uuid.UUID) (*Order, error) {
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
	o, err := client.GetOrder(ctx, &pb.GetOrderRequest{Id: id.String()})
	if err != nil {
		return nil, err
	}
	return orderFromProto(o), nil
}

func (c *Client) ListOrders(ctx context.Context) ([]*Order, error) {
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.ListOrders(ctx, &pb.ListOrdersRequest{})
	if err != nil {
		return nil, err
	}
	out := make([]*Order, 0, len(resp.Orders))
	for _, o := range resp.Orders {
		out = append(out, orderFromProto(o))
	}
	return out, nil
}

func (c *Client) GetPendingPaymentOrdersByUser(ctx context.Context, userID uuid.UUID) ([]*Order, error) {
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GetPendingPaymentOrdersByUser(ctx, &pb.GetPendingPaymentOrdersByUserRequest{
		UserId: userID.String(),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*Order, 0, len(resp.Orders))
	for _, o := range resp.Orders {
		out = append(out, orderFromProto(o))
	}
	return out, nil
}

func (c *Client) UpdateOrder(ctx context.Context, id uuid.UUID, userID uuid.UUID, req *UpdateOrderRequest) (*Order, error) {
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
	pbReq := &pb.UpdateOrderRequest{
		Id:     id.String(),
		UserId: userID.String(),
		State:  req.State,
	}
	if req.SellerRespondBy != nil {
		pbReq.SellerRespondBy = timestamppb.New(*req.SellerRespondBy)
	}
	if req.ReviewEndsAt != nil {
		pbReq.ReviewEndsAt = timestamppb.New(*req.ReviewEndsAt)
	}
	o, err := client.UpdateOrder(ctx, pbReq)
	if err != nil {
		return nil, err
	}
	return orderFromProto(o), nil
}

func (c *Client) DeleteOrder(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	client, err := c.connClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.DeleteOrder(ctx, &pb.DeleteOrderRequest{Id: id.String(), UserId: userID.String()})
	return err
}

func (c *Client) TransitionState(ctx context.Context, orderID uuid.UUID, targetState, actor string) error {
	client, err := c.connClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.TransitionState(ctx, &pb.TransitionStateRequest{
		OrderId:     orderID.String(),
		TargetState: targetState,
		Actor:       actor,
	})
	return err
}

func orderFromProto(o *pb.Order) *Order {
	id, _ := uuid.Parse(o.Id)
	listingID, _ := uuid.Parse(o.ListingId)
	buyerID, _ := uuid.Parse(o.BuyerId)
	sellerID, _ := uuid.Parse(o.SellerId)
	out := &Order{
		ID:        id,
		ListingID: listingID,
		BuyerID:   buyerID,
		SellerID:  sellerID,
		Quantity:  int(o.Quantity),
		Amount:    o.Amount,
		State:     o.State,
		CreatedAt: o.CreatedAt.AsTime(),
	}
	if o.SellerRespondBy != nil {
		t := o.SellerRespondBy.AsTime()
		out.SellerRespondBy = &t
	}
	if o.ReviewEndsAt != nil {
		t := o.ReviewEndsAt.AsTime()
		out.ReviewEndsAt = &t
	}
	return out
}
