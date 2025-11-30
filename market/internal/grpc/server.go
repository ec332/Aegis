package grpc

import (
    "context"
    "time"

    market "github.com/aegis/proto/gen/market"
    "github.com/ec332/aegis/market/internal/service"
    "github.com/ec332/aegis/market/pkg/models"
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
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "id is required")
    }

    marketModel, err := s.service.GetMarket(ctx, req.Id)
	if err != nil {
        s.logger.Error("failed to get market", zap.String("id", req.Id), zap.Error(err))
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
    if req.Question == "" {
        return nil, status.Error(codes.InvalidArgument, "question is required")
    }
    if req.Description == "" {
        return nil, status.Error(codes.InvalidArgument, "description is required")
    }

	s.logger.Info("Api Gateway received CreateMarket request",
		zap.Any("request", req))

    // Convert protobuf request to internal model
    var endPtr *time.Time
    if req.EndTime != nil {
        t := req.EndTime.AsTime()
        endPtr = &t
    }
    createReq := models.CreateMarketRequest{
        Title:              req.Question,
        Description:        req.Description,
        Options:            req.Options,
        ResolutionDatetime: endPtr,
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
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "id is required")
    }

	// Convert protobuf request to internal model
	updateReq := models.UpdateMarketRequest{}
	
    if req.Status != "" {
        statusVal := models.MarketStatus(req.Status)
        updateReq.Status = &statusVal
    }
    if req.EndTime != nil {
        t := req.EndTime.AsTime()
        updateReq.ResolutionDatetime = &t
    }
    if req.Outcome != "" {
        updateReq.WinningOptionID = &req.Outcome
    }

    marketModel, err := s.service.UpdateMarket(ctx, req.Id, updateReq)
	if err != nil {
        s.logger.Error("failed to update market", 
            zap.String("id", req.Id), 
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
    if req.Status != "" {
        status := models.MarketStatus(req.Status)
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

// GetMarketOptions retrieves options for a market with current LMSR prices
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

	// Get current prices using LMSR
	prices, err := s.service.GetMarketPrices(ctx, req.MarketId)
	if err != nil {
		s.logger.Error("failed to get market prices", 
			zap.String("market_id", req.MarketId), 
			zap.Error(err))
		return nil, status.Error(codes.Internal, "failed to get market prices")
	}

	protoOptions := make([]*market.Option, len(marketModel.Options))
	for i, option := range marketModel.Options {
		protoOpt := convertOptionToProto(&option)
		protoOpt.CurrentPrice = prices[option.ID]
		protoOptions[i] = protoOpt
	}

	return &market.GetMarketOptionsResponse{
		Options: protoOptions,
	}, nil
}

// User management methods

// GetUser retrieves a user by ID
func (s *Server) GetUser(ctx context.Context, req *market.GetUserRequest) (*market.GetUserResponse, error) {
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
    if req.Id == "" {
        return nil, status.Error(codes.InvalidArgument, "id is required")
    }

	updateReq := models.UpdateUserRequest{}
    if req.WalletAddress != "" {
        // Not persisted in UpdateUserRequest model; ignore
    }
    // Balance and Role are not optional in proto; apply when non-zero/non-empty
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

	return &market.UpdateUserResponse{
		User: convertUserToProto(user),
	}, nil
}

// Helper functions to convert between models and protobuf types

func convertMarketToProto(marketModel *models.Market) *market.Market {
    protoMarket := &market.Market{
        Id:          marketModel.ID,
        Question:    marketModel.Title,
        Description: marketModel.Description,
        Status:      string(marketModel.Status),
        CreatedAt:   timestamppb.New(marketModel.CreatedAt),
        UpdatedAt:   timestamppb.New(marketModel.UpdatedAt),
    }

    if marketModel.ResolutionDatetime != nil {
        protoMarket.EndTime = timestamppb.New(*marketModel.ResolutionDatetime)
    }

    // Options are returned via GetMarketOptions

    // Liquidity pools are not part of current proto

	return protoMarket
}

func convertOptionToProto(optionModel *models.Option) *market.Option {
    return &market.Option{
        Id:           optionModel.ID,
        MarketId:     optionModel.MarketID,
        OptionText:   optionModel.Title,
        CurrentPrice: 0,
        Volume:       0,
        CreatedAt:    timestamppb.New(optionModel.CreatedAt),
        UpdatedAt:    timestamppb.New(optionModel.CreatedAt),
    }
}

// Liquidity pool conversion removed; not defined in current proto

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