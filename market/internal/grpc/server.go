package grpc

import (
	"context"
	"fmt"

	market "github.com/aegis/proto/gen/market"
	"github.com/aegis/shared/utils"
	"github.com/ec332/aegis/market/internal/service"
	"github.com/ec332/aegis/market/pkg/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server implements the MarketService gRPC server
type Server struct {
	market.UnimplementedMarketServiceServer
	service *service.Service
	logger  *zap.Logger
}

// NewServer creates a new gRPC server instance
func NewServer(service *service.Service, logger *zap.Logger) *Server {
	return &Server{
		service: service,
		logger:  logger,
	}
}

// GetMarket retrieves a market by ID
func (s *Server) GetMarket(ctx context.Context, req *market.GetMarketRequest) (*market.GetMarketResponse, error) {
	if req.MarketId == "" {
		return nil, status.Error(codes.InvalidArgument, "market_id is required")
	}

	marketModel, err := s.service.GetMarket(ctx, req.MarketId)
	if err != nil {
		s.logger.Error("failed to get market", zap.String("market_id", req.MarketId), zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get market")
	}

	if marketModel == nil {
		return nil, status.Error(codes.NotFound, "market not found")
	}

	return &market.GetMarketResponse{
		Market: convertMarketToProto(marketModel),
	}, nil
}

// CreateMarket creates a new market
func (s *Server) CreateMarket(ctx context.Context, req *market.CreateMarketRequest) (*market.CreateMarketResponse, error) {
	// Validate request
	if req.Title == "" {
		return nil, status.Error(codes.InvalidArgument, "title is required")
	}
	if req.Description == "" {
		return nil, status.Error(codes.InvalidArgument, "description is required")
	}
	if len(req.Options) < 2 {
		return nil, status.Error(codes.InvalidArgument, "at least 2 options are required")
	}

	// Convert protobuf request to internal model
	createReq := models.CreateMarketRequest{
		Title:              req.Title,
		Description:        req.Description,
		Options:            req.Options,
		ResolutionDatetime: req.ResolutionDatetime.AsTime(),
	}

	marketModel, err := s.service.CreateMarket(ctx, createReq)
	if err != nil {
		s.logger.Error("failed to create market", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to create market")
	}

	return &market.CreateMarketResponse{
		Market: convertMarketToProto(marketModel),
	}, nil
}

// UpdateMarket updates a market's details
func (s *Server) UpdateMarket(ctx context.Context, req *market.UpdateMarketRequest) (*market.UpdateMarketResponse, error) {
	if req.MarketId == "" {
		return nil, status.Error(codes.InvalidArgument, "market_id is required")
	}

	// Convert protobuf request to internal model
	updateReq := models.UpdateMarketRequest{}
	
	if req.Title != nil {
		updateReq.Title = req.Title
	}
	if req.Description != nil {
		updateReq.Description = req.Description
	}
	if req.Status != nil {
		status := models.MarketStatus(*req.Status)
		updateReq.Status = &status
	}
	if req.ResolutionDatetime != nil {
		updateReq.ResolutionDatetime = req.ResolutionDatetime
	}
	if req.WinningOptionId != nil {
		updateReq.WinningOptionID = req.WinningOptionId
	}

	marketModel, err := s.service.UpdateMarket(ctx, req.MarketId, updateReq)
	if err != nil {
		s.logger.Error("failed to update market", 
			zap.String("market_id", req.MarketId), 
			zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update market")
	}

	return &market.UpdateMarketResponse{
		Market: convertMarketToProto(marketModel),
	}, nil
}

// ListMarkets retrieves markets
func (s *Server) ListMarkets(ctx context.Context, req *market.ListMarketsRequest) (*market.ListMarketsResponse, error) {
	var statusFilter *models.MarketStatus
	if req.Status != nil {
		status := models.MarketStatus(*req.Status)
		statusFilter = &status
	}

	markets, err := s.service.ListMarkets(ctx, statusFilter)
	if err != nil {
		s.logger.Error("failed to list markets", zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to list markets")
	}

	protoMarkets := make([]*market.Market, len(markets))
	for i, marketModel := range markets {
		protoMarkets[i] = convertMarketToProto(&marketModel)
	}

	return &market.ListMarketsResponse{
		Markets: protoMarkets,
	}, nil
}

// GetMarketOptions retrieves options for a market
func (s *Server) GetMarketOptions(ctx context.Context, req *market.GetMarketOptionsRequest) (*market.GetMarketOptionsResponse, error) {
	if req.MarketId == "" {
		return nil, status.Error(codes.InvalidArgument, "market_id is required")
	}

	marketModel, err := s.service.GetMarket(ctx, req.MarketId)
	if err != nil {
		s.logger.Error("failed to get market options", 
			zap.String("market_id", req.MarketId), 
			zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get market options")
	}

	if marketModel == nil {
		return nil, status.Error(codes.NotFound, "market not found")
	}

	protoOptions := make([]*market.Option, len(marketModel.Options))
	for i, option := range marketModel.Options {
		protoOptions[i] = convertOptionToProto(&option)
	}

	return &market.GetMarketOptionsResponse{
		Options: protoOptions,
	}, nil
}

// User management methods

// GetUser retrieves a user by ID
func (s *Server) GetUser(ctx context.Context, req *market.GetUserRequest) (*market.GetUserResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	user, err := s.service.GetUser(ctx, req.UserId)
	if err != nil {
		s.logger.Error("failed to get user", 
			zap.String("user_id", req.UserId), 
			zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get user")
	}

	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return &market.GetUserResponse{
		User: convertUserToProto(user),
	}, nil
}

// GetUserByWallet retrieves a user by wallet address
func (s *Server) GetUserByWallet(ctx context.Context, req *market.GetUserByWalletRequest) (*market.GetUserResponse, error) {
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

	return &market.GetUserResponse{
		User: convertUserToProto(user),
	}, nil
}

// CreateUser creates a new user
func (s *Server) CreateUser(ctx context.Context, req *market.CreateUserRequest) (*market.CreateUserResponse, error) {
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

	return &market.CreateUserResponse{
		User: convertUserToProto(user),
	}, nil
}

// UpdateUser updates a user
func (s *Server) UpdateUser(ctx context.Context, req *market.UpdateUserRequest) (*market.UpdateUserResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	updateReq := models.UpdateUserRequest{}
	if req.Balance != nil {
		updateReq.Balance = req.Balance
	}
	if req.Nonce != nil {
		updateReq.Nonce = req.Nonce
	}
	if req.Role != nil {
		role := models.UserRole(*req.Role)
		updateReq.Role = &role
	}

	user, err := s.service.UpdateUser(ctx, req.UserId, updateReq)
	if err != nil {
		s.logger.Error("failed to update user", 
			zap.String("user_id", req.UserId), 
			zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to update user")
	}

	return &market.UpdateUserResponse{
		User: convertUserToProto(user),
	}, nil
}

// Helper functions to convert between models and protobuf types

func convertMarketToProto(marketModel *models.Market) *market.Market {
	protoMarket := &market.Market{
		Id:                 marketModel.ID,
		Title:              marketModel.Title,
		Description:        marketModel.Description,
		Status:             string(marketModel.Status),
		ResolutionDatetime: timestamppb.New(marketModel.ResolutionDatetime),
		CreatedAt:          timestamppb.New(marketModel.CreatedAt),
		UpdatedAt:          timestamppb.New(marketModel.UpdatedAt),
	}

	if marketModel.WinningOptionID != nil {
		protoMarket.WinningOptionId = *marketModel.WinningOptionID
	}

	if marketModel.Options != nil {
		protoMarket.Options = make([]*market.Option, len(marketModel.Options))
		for i, option := range marketModel.Options {
			protoMarket.Options[i] = convertOptionToProto(&option)
		}
	}

	if marketModel.LiquidityPools != nil {
		protoMarket.LiquidityPools = make([]*market.LiquidityPool, len(marketModel.LiquidityPools))
		for i, pool := range marketModel.LiquidityPools {
			protoMarket.LiquidityPools[i] = convertLiquidityPoolToProto(&pool)
		}
	}

	return protoMarket
}

func convertOptionToProto(optionModel *models.Option) *market.Option {
	return &market.Option{
		Id:        optionModel.ID,
		MarketId:  optionModel.MarketID,
		Title:     optionModel.Title,
		CreatedAt: timestamppb.New(optionModel.CreatedAt),
	}
}

func convertLiquidityPoolToProto(poolModel *models.LiquidityPool) *market.LiquidityPool {
	return &market.LiquidityPool{
		Id:        poolModel.ID,
		MarketId:  poolModel.MarketID,
		OptionId:  poolModel.OptionID,
		PoolValue: poolModel.PoolValue,
		UpdatedAt: timestamppb.New(poolModel.UpdatedAt),
	}
}

func convertUserToProto(userModel *models.User) *market.User {
	return &market.User{
		Id:            userModel.ID,
		WalletAddress: userModel.WalletAddress,
		Balance:       userModel.Balance,
		Nonce:         userModel.Nonce,
		Role:          string(userModel.Role),
		CreatedAt:     timestamppb.New(userModel.CreatedAt),
		UpdatedAt:     timestamppb.New(userModel.UpdatedAt),
	}
}