package auth

import (
	"context"
	"errors"

	pb "github.com/darkphotonKN/seeyoulatte-app/common/api/proto/auth"
	commonconstants "github.com/darkphotonKN/seeyoulatte-app/common/constants"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service interface {
	SignUp(ctx context.Context, in *SignUpInput) (*AuthTokens, error)
	SignIn(ctx context.Context, in *SignInInput) (*AuthTokens, error)
	GoogleAuth(ctx context.Context, in *GoogleAuthInput) (*AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error)
	GetUser(ctx context.Context, id uuid.UUID) (*User, error)
	ListUsers(ctx context.Context) ([]*User, error)
	UpdateProfile(ctx context.Context, in *UpdateProfileInput) (*User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	VerifyUserNotFrozen(ctx context.Context, id uuid.UUID) error
	GetByStripeCustomerID(ctx context.Context, stripeCustomerID string) (*User, error)
	UpdateStripeCustomerID(ctx context.Context, userID uuid.UUID, stripeCustomerID string) error
}

type Handler struct {
	pb.UnimplementedAuthServiceServer
	service Service
}

func NewHandler(s Service) *Handler {
	return &Handler{service: s}
}

func (h *Handler) SignUp(ctx context.Context, req *pb.SignUpRequest) (*pb.AuthResponse, error) {
	tokens, err := h.service.SignUp(ctx, &SignUpInput{Email: req.Email, Password: req.Password, Name: req.Name})
	if err != nil {
		return nil, mapError(err)
	}
	return tokensToProto(tokens), nil
}

func (h *Handler) SignIn(ctx context.Context, req *pb.SignInRequest) (*pb.AuthResponse, error) {
	tokens, err := h.service.SignIn(ctx, &SignInInput{Email: req.Email, Password: req.Password})
	if err != nil {
		return nil, mapError(err)
	}
	return tokensToProto(tokens), nil
}

func (h *Handler) GoogleAuth(ctx context.Context, req *pb.GoogleAuthRequest) (*pb.AuthResponse, error) {
	tokens, err := h.service.GoogleAuth(ctx, &GoogleAuthInput{IDToken: req.IdToken})
	if err != nil {
		return nil, mapError(err)
	}
	return tokensToProto(tokens), nil
}

func (h *Handler) RefreshToken(ctx context.Context, req *pb.RefreshTokenRequest) (*pb.AuthResponse, error) {
	tokens, err := h.service.Refresh(ctx, req.RefreshToken)
	if err != nil {
		return nil, mapError(err)
	}
	return tokensToProto(tokens), nil
}

func (h *Handler) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	u, err := h.service.GetUser(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return userToProto(u), nil
}

func (h *Handler) ListUsers(ctx context.Context, _ *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
	users, err := h.service.ListUsers(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]*pb.User, 0, len(users))
	for _, u := range users {
		out = append(out, userToProto(u))
	}
	return &pb.ListUsersResponse{Users: out}, nil
}

func (h *Handler) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.User, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	u, err := h.service.UpdateProfile(ctx, &UpdateProfileInput{
		ID:                          id,
		Name:                        req.Name,
		Bio:                         req.Bio,
		LocationText:                req.LocationText,
		PreferredPickupInstructions: req.PreferredPickupInstructions,
	})
	if err != nil {
		return nil, mapError(err)
	}
	return userToProto(u), nil
}

func (h *Handler) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	if err := h.service.DeleteUser(ctx, id); err != nil {
		return nil, mapError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) VerifyUserNotFrozen(ctx context.Context, req *pb.VerifyUserNotFrozenRequest) (*emptypb.Empty, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}
	if err := h.service.VerifyUserNotFrozen(ctx, id); err != nil {
		return nil, mapError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *Handler) GetByStripeCustomerID(ctx context.Context, req *pb.GetByStripeCustomerIDRequest) (*pb.User, error) {
	u, err := h.service.GetByStripeCustomerID(ctx, req.StripeCustomerId)
	if err != nil {
		return nil, mapError(err)
	}
	return userToProto(u), nil
}

func (h *Handler) UpdateStripeCustomerID(ctx context.Context, req *pb.UpdateStripeCustomerIDRequest) (*emptypb.Empty, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id: %v", err)
	}
	if err := h.service.UpdateStripeCustomerID(ctx, userID, req.StripeCustomerId); err != nil {
		return nil, mapError(err)
	}
	return &emptypb.Empty{}, nil
}

func userToProto(u *User) *pb.User {
	out := &pb.User{
		Id:                          u.ID.String(),
		Email:                       u.Email,
		Name:                        u.Name,
		Bio:                         u.Bio,
		LocationText:                u.LocationText,
		IsFrozen:                    u.IsFrozen,
		AvatarUrl:                   u.AvatarURL,
		IsVerified:                  u.IsVerified,
		PreferredPickupInstructions: u.PreferredPickupInstructions,
		StripeCustomerId:            u.StripeCustomerID,
		CreatedAt:                   timestamppb.New(u.CreatedAt),
		UpdatedAt:                   timestamppb.New(u.UpdatedAt),
	}
	if u.LastLoginAt != nil {
		out.LastLoginAt = timestamppb.New(*u.LastLoginAt)
	}
	return out
}

func tokensToProto(t *AuthTokens) *pb.AuthResponse {
	return &pb.AuthResponse{
		User:             userToProto(t.User),
		AccessToken:      t.AccessToken,
		RefreshToken:     t.RefreshToken,
		AccessExpiresIn:  int64(t.AccessExpiresIn),
		RefreshExpiresIn: int64(t.RefreshExpiresIn),
	}
}

func mapError(err error) error {
	switch {
	case errors.Is(err, commonconstants.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, commonconstants.ErrDuplicateResource):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, commonconstants.ErrInvalidInput),
		errors.Is(err, commonconstants.ErrConstraintViolation):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, commonconstants.ErrUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, commonconstants.ErrForbidden):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
