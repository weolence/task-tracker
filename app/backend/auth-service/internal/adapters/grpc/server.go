package grpcadapter

import (
	"context"

	authsvcv1 "auth-service/api/proto/authsvcv1"
	userv1 "auth-service/api/proto/userv1"
	"auth-service/internal/core/domain"
	"auth-service/internal/core/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthServer struct {
	authsvcv1.UnimplementedAuthServiceServer
	auth *usecase.AuthUseCase
}

func NewAuthServer(auth *usecase.AuthUseCase) *AuthServer {
	return &AuthServer{auth: auth}
}

func (s *AuthServer) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.OperationResponse, error) {
	err := s.auth.Register(ctx, domain.User{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Surname:  req.Surname,
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "register: %v", err)
	}
	return &userv1.OperationResponse{Message: "user registered"}, nil
}

func (s *AuthServer) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	token, err := s.auth.Login(ctx, req.Email, req.Password)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid credentials")
	}
	return &userv1.LoginResponse{Token: token}, nil
}

func (s *AuthServer) ValidateToken(ctx context.Context, req *userv1.ValidateTokenRequest) (*userv1.ValidateTokenResponse, error) {
	payload, err := s.auth.ValidateToken(ctx, req.Token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}
	return &userv1.ValidateTokenResponse{
		UserId: int32(payload.UserID),
		Role:   payload.Role,
	}, nil
}

func (s *AuthServer) GetUserInfo(ctx context.Context, req *userv1.GetUserRequest) (*userv1.User, error) {
	return s.getUser(ctx, req)
}

func (s *AuthServer) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.User, error) {
	return s.getUser(ctx, req)
}

func (s *AuthServer) getUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.User, error) {
	var (
		u   *domain.User
		err error
	)
	switch {
	case req.UserId != nil:
		u, err = s.auth.GetUserByID(ctx, int(*req.UserId))
	case req.Email != nil && *req.Email != "":
		u, err = s.auth.GetUserByEmail(ctx, *req.Email)
	default:
		return nil, status.Error(codes.InvalidArgument, "user_id or email required")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get user: %v", err)
	}
	if u == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return &userv1.User{
		Id:      u.ID,
		Email:   u.Email,
		Name:    u.Name,
		Surname: u.Surname,
		Role:    u.Role,
	}, nil
}

func (s *AuthServer) UpdateUser(ctx context.Context, req *userv1.UpdateUserRequest) (*userv1.OperationResponse, error) {
	if req.User == nil {
		return nil, status.Error(codes.InvalidArgument, "user is required")
	}
	u := domain.User{
		ID:      req.User.Id,
		Email:   req.User.Email,
		Name:    req.User.Name,
		Surname: req.User.Surname,
		Role:    req.User.Role,
	}
	var pwPtr *string
	if req.Password != nil {
		pwPtr = req.Password
	}
	if err := s.auth.UpdateUser(ctx, u, pwPtr); err != nil {
		return nil, status.Errorf(codes.Internal, "update user: %v", err)
	}
	return &userv1.OperationResponse{Message: "user updated"}, nil
}

func (s *AuthServer) DeleteUser(ctx context.Context, req *userv1.DeleteUserRequest) (*userv1.OperationResponse, error) {
	user, err := s.auth.GetUserByID(ctx, int(req.UserId))
	if err != nil || user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err := s.auth.DeleteUser(ctx, user.Email); err != nil {
		return nil, status.Errorf(codes.Internal, "delete user: %v", err)
	}
	return &userv1.OperationResponse{Message: "user deleted"}, nil
}
