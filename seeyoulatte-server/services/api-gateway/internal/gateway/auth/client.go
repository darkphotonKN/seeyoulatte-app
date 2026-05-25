package authgw

import (
	"context"
	"sync"

	pb "github.com/darkphotonKN/seeyoulatte-app/common/api/proto/auth"
	"github.com/darkphotonKN/seeyoulatte-app/common/discovery"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

const targetService = "auth"

// Client wraps a long-lived gRPC connection to auth-service. The connection is
// established lazily on first use and reused across all calls — gRPC's HTTP/2
// layer multiplexes streams over a single connection, so concurrent RPCs are
// cheap. Opening a new conn per RPC serializes badly under load.
type Client struct {
	registry discovery.Registry
	mu       sync.Mutex
	conn     *grpc.ClientConn
}

func NewClient(registry discovery.Registry) *Client {
	return &Client{registry: registry}
}

func (c *Client) connClient(ctx context.Context) (pb.AuthServiceClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && c.conn.GetState() != connectivity.Shutdown {
		return pb.NewAuthServiceClient(c.conn), nil
	}

	conn, err := discovery.ServiceConnection(ctx, targetService, c.registry)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	return pb.NewAuthServiceClient(conn), nil
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

func (c *Client) SignUp(ctx context.Context, req *SignUpRequest) (*AuthResponse, error) {
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.SignUp(ctx, &pb.SignUpRequest{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		return nil, err
	}
	return authRespToHTTP(resp), nil
}

func (c *Client) SignIn(ctx context.Context, req *SignInRequest) (*AuthResponse, error) {
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.SignIn(ctx, &pb.SignInRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return nil, err
	}
	return authRespToHTTP(resp), nil
}

func (c *Client) GoogleAuth(ctx context.Context, idToken string) (*AuthResponse, error) {
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.GoogleAuth(ctx, &pb.GoogleAuthRequest{IdToken: idToken})
	if err != nil {
		return nil, err
	}
	return authRespToHTTP(resp), nil
}

func (c *Client) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
	u, err := client.GetUser(ctx, &pb.GetUserRequest{Id: id.String()})
	if err != nil {
		return nil, err
	}
	return userFromProto(u), nil
}

func (c *Client) UpdateProfile(ctx context.Context, id uuid.UUID, req UpdateProfileRequest) (*User, error) {
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
	u, err := client.UpdateProfile(ctx, &pb.UpdateProfileRequest{
		Id:                          id.String(),
		Name:                        req.Name,
		Bio:                         req.Bio,
		LocationText:                req.LocationText,
		PreferredPickupInstructions: req.PreferredPickupInstructions,
	})
	if err != nil {
		return nil, err
	}
	return userFromProto(u), nil
}

func (c *Client) ListUsers(ctx context.Context) ([]*User, error) {
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := client.ListUsers(ctx, &pb.ListUsersRequest{})
	if err != nil {
		return nil, err
	}
	users := make([]*User, 0, len(resp.Users))
	for _, u := range resp.Users {
		users = append(users, userFromProto(u))
	}
	return users, nil
}

func (c *Client) GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*User, error) {
	client, err := c.connClient(ctx)
	if err != nil {
		return nil, err
	}
	u, err := client.GetByStripeCustomerID(ctx, &pb.GetByStripeCustomerIDRequest{StripeCustomerId: stripeCustomerID})
	if err != nil {
		return nil, err
	}
	return userFromProto(u), nil
}

func (c *Client) GetUserIDByStripeCustomerID(ctx context.Context, stripeCustomerID string) (uuid.UUID, error) {
	u, err := c.GetByStripeCustomerID(ctx, stripeCustomerID)
	if err != nil {
		return uuid.Nil, err
	}
	return u.ID, nil
}

func (c *Client) UpdateStripeCustomerID(ctx context.Context, userID uuid.UUID, stripeCustomerID string) error {
	client, err := c.connClient(ctx)
	if err != nil {
		return err
	}
	_, err = client.UpdateStripeCustomerID(ctx, &pb.UpdateStripeCustomerIDRequest{
		UserId:           userID.String(),
		StripeCustomerId: stripeCustomerID,
	})
	return err
}

func userFromProto(u *pb.User) *User {
	id, _ := uuid.Parse(u.Id)
	out := &User{
		ID:                          id,
		Email:                       u.Email,
		Name:                        u.Name,
		Bio:                         u.Bio,
		LocationText:                u.LocationText,
		IsFrozen:                    u.IsFrozen,
		AvatarURL:                   u.AvatarUrl,
		IsVerified:                  u.IsVerified,
		PreferredPickupInstructions: u.PreferredPickupInstructions,
		StripeCustomerID:            u.StripeCustomerId,
		CreatedAt:                   u.CreatedAt.AsTime(),
		UpdatedAt:                   u.UpdatedAt.AsTime(),
	}
	if u.LastLoginAt != nil {
		t := u.LastLoginAt.AsTime()
		out.LastLoginAt = &t
	}
	return out
}

func authRespToHTTP(r *pb.AuthResponse) *AuthResponse {
	return &AuthResponse{
		User:             userFromProto(r.User),
		AccessToken:      r.AccessToken,
		RefreshToken:     r.RefreshToken,
		AccessExpiresIn:  r.AccessExpiresIn,
		RefreshExpiresIn: r.RefreshExpiresIn,
	}
}
