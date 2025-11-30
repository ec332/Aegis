package grpc

import (
    "context"
    "time"

    "aegis/wallet/internal/service"
    "aegis/wallet/pkg/models"
    "aegis/wallet/internal/auth"

    wallet "github.com/aegis/proto/gen/wallet"
    "go.uber.org/zap"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/timestamppb"
)

// Server implements the WalletService gRPC server
type Server struct {
    wallet.UnimplementedWalletServiceServer
    service *service.Service
    logger  *zap.Logger
    tm      *auth.TokenManager
}

// NewServer creates a new gRPC server instance
func NewServer(service *service.Service, logger *zap.Logger, tm *auth.TokenManager) *Server {
    return &Server{
        service: service,
        logger:  logger,
        tm:      tm,
    }
}

// User management methods

// GetUser retrieves a user by ID
func (s *Server) GetUser(ctx context.Context, req *wallet.GetUserRequest) (*wallet.GetUserResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	user, err := s.service.GetUser(ctx, req.Id)
	if err != nil {
		s.logger.Error("failed to get user",
			zap.String("id", req.Id),
			zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &wallet.GetUserResponse{
		User: convertUserToProto(user),
	}, nil
}

// GetUserByWallet retrieves a user by wallet address
func (s *Server) GetUserByWallet(ctx context.Context, req *wallet.GetUserByWalletRequest) (*wallet.GetUserResponse, error) {
	if req.WalletAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "wallet_address is required")
	}

	user, err := s.service.GetUserByWalletAddress(ctx, req.WalletAddress)
	if err != nil {
		s.logger.Error("failed to get user by wallet",
			zap.String("wallet_address", req.WalletAddress),
			zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get user by wallet")
	}

	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &wallet.GetUserResponse{
		User: convertUserToProto(user),
	}, nil
}

// CreateUser creates a new user
func (s *Server) CreateUser(ctx context.Context, req *wallet.CreateUserRequest) (*wallet.CreateUserResponse, error) {
	if req.WalletAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "wallet_address is required")
	}

	createReq := models.CreateUserRequest{
		WalletAddress: req.WalletAddress,
		Balance:       req.Balance,
	}

	user, err := s.service.CreateUser(ctx, createReq)
	if err != nil {
		s.logger.Error("failed to create user",
			zap.String("wallet_address", req.WalletAddress),
			zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create user")
	}

	return &wallet.CreateUserResponse{
		User: convertUserToProto(user),
	}, nil
}

// UpdateUser updates a user
func (s *Server) UpdateUser(ctx context.Context, req *wallet.UpdateUserRequest) (*wallet.UpdateUserResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	updateReq := models.UpdateUserRequest{}
	if req.Balance != 0 {
		b := req.Balance
		updateReq.Balance = &b
	}
	if req.Role != "" {
		role := models.UserRole(req.Role)
		updateReq.Role = &role
	}

	user, err := s.service.UpdateUser(ctx, req.Id, updateReq)
	if err != nil {
		s.logger.Error("failed to update user",
			zap.String("id", req.Id),
			zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update user")
	}

	return &wallet.UpdateUserResponse{
		User: convertUserToProto(user),
	}, nil
}

func (s *Server) RequestNonce(ctx context.Context, req *wallet.RequestNonceRequest) (*wallet.RequestNonceResponse, error) {
    if req.Wallet == "" {
        return nil, status.Error(codes.InvalidArgument, "wallet is required")
    }
    nonce, err := s.service.RequestNonce(ctx, req.Wallet)
    if err != nil {
        s.logger.Error("request nonce failed", zap.Error(err), zap.String("wallet", req.Wallet))
        return nil, status.Error(codes.Internal, "request nonce failed")
    }
    return &wallet.RequestNonceResponse{Nonce: nonce}, nil
}

func (s *Server) VerifySignature(ctx context.Context, req *wallet.VerifySignatureRequest) (*wallet.VerifySignatureResponse, error) {
    if req.Wallet == "" || req.Signature == "" {
        return nil, status.Error(codes.InvalidArgument, "wallet and signature are required")
    }
    token, err := s.service.VerifySignature(ctx, req.Wallet, req.Signature, func(wallet string) (string, error) {
        return s.tm.Generate(wallet, 7*24*time.Hour)
    })
    if err != nil {
        s.logger.Error("verify signature failed", zap.Error(err), zap.String("wallet", req.Wallet))
        return nil, status.Error(codes.Unauthenticated, "verification failed")
    }
    return &wallet.VerifySignatureResponse{Token: token}, nil
}

// Helper functions to convert between models and protobuf types

func convertUserToProto(userModel *models.User) *wallet.User {
    return &wallet.User{
        Id:            userModel.ID,
        WalletAddress: userModel.WalletAddress,
        Balance:       userModel.Balance,
        Nonce:         userModel.Nonce,
        Role:          string(userModel.Role),
        CreatedAt:     timestamppb.New(userModel.CreatedAt),
        UpdatedAt:     timestamppb.New(userModel.UpdatedAt),
        LastLogin:     func() *timestamppb.Timestamp { if userModel.LastLogin != nil { return timestamppb.New(*userModel.LastLogin) } ; return nil }(),
    }
}
