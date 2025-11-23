package main

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"aegis/shared/grpc"
	"aegis/shared/metrics"
	"aegis/settlement/internal/service"
	"aegis/settlement/internal/repository"
	"aegis/proto/settlement"
)

type SettlementGRPCServer struct {
	settlement.UnimplementedSettlementServiceServer
	service *service.SettlementService
	logger  *zap.Logger
}

func NewSettlementGRPCServer(settlementService *service.SettlementService, logger *zap.Logger) *SettlementGRPCServer {
	return &SettlementGRPCServer{
		service: settlementService,
		logger:  logger,
	}
}

func (s *SettlementGRPCServer) CreateSettlement(ctx context.Context, req *settlement.CreateSettlementRequest) (*settlement.CreateSettlementResponse, error) {
	s.logger.Info("Creating settlement",
		zap.String("market_id", req.MarketId),
		zap.String("winning_option_id", req.WinningOptionId),
	)

	settlement, err := s.service.CreateSettlement(ctx, req.MarketId, req.WinningOptionId)
	if err != nil {
		s.logger.Error("Failed to create settlement", zap.Error(err))
		return nil, fmt.Errorf("failed to create settlement: %w", err)
	}

	return &settlement.CreateSettlementResponse{
		Settlement: &settlement.Settlement{
			Id:              settlement.ID,
			MarketId:        settlement.MarketID,
			WinningOptionId: settlement.WinningOptionID,
			Status:          settlement.Status,
			CreatedAt:       settlement.CreatedAt.Unix(),
			UpdatedAt:       settlement.UpdatedAt.Unix(),
		},
	}, nil
}

func (s *SettlementGRPCServer) GetSettlement(ctx context.Context, req *settlement.GetSettlementRequest) (*settlement.GetSettlementResponse, error) {
	s.logger.Info("Getting settlement", zap.String("settlement_id", req.SettlementId))

	settlement, err := s.service.GetSettlement(ctx, req.SettlementId)
	if err != nil {
		s.logger.Error("Failed to get settlement", zap.Error(err))
		return nil, fmt.Errorf("failed to get settlement: %w", err)
	}

	return &settlement.GetSettlementResponse{
		Settlement: &settlement.Settlement{
			Id:              settlement.ID,
			MarketId:        settlement.MarketID,
			WinningOptionId: settlement.WinningOptionID,
			Status:          settlement.Status,
			CreatedAt:       settlement.CreatedAt.Unix(),
			UpdatedAt:       settlement.UpdatedAt.Unix(),
		},
	}, nil
}

func (s *SettlementGRPCServer) CompleteSettlement(ctx context.Context, req *settlement.CompleteSettlementRequest) (*settlement.CompleteSettlementResponse, error) {
	s.logger.Info("Completing settlement", zap.String("settlement_id", req.SettlementId))

	settlement, err := s.service.CompleteSettlement(ctx, req.SettlementId)
	if err != nil {
		s.logger.Error("Failed to complete settlement", zap.Error(err))
		return nil, fmt.Errorf("failed to complete settlement: %w", err)
	}

	return &settlement.CompleteSettlementResponse{
		Settlement: &settlement.Settlement{
			Id:              settlement.ID,
			MarketId:        settlement.MarketID,
			WinningOptionId: settlement.WinningOptionID,
			Status:          settlement.Status,
			CreatedAt:       settlement.CreatedAt.Unix(),
			UpdatedAt:       settlement.UpdatedAt.Unix(),
		},
	}, nil
}

func (s *SettlementGRPCServer) ProcessPayout(ctx context.Context, req *settlement.ProcessPayoutRequest) (*settlement.ProcessPayoutResponse, error) {
	s.logger.Info("Processing payout",
		zap.String("settlement_id", req.SettlementId),
		zap.String("user_id", req.UserId),
		zap.Float64("amount", req.Amount),
	)

	distribution, err := s.service.ProcessPayout(ctx, req.SettlementId, req.UserId, req.Amount)
	if err != nil {
		s.logger.Error("Failed to process payout", zap.Error(err))
		return nil, fmt.Errorf("failed to process payout: %w", err)
	}

	return &settlement.ProcessPayoutResponse{
		Distribution: &settlement.SettlementDistribution{
			Id:           distribution.ID,
			SettlementId: distribution.SettlementID,
			UserId:       distribution.UserID,
			Amount:       distribution.Amount,
			Status:       distribution.Status,
			ProcessedAt:  distribution.ProcessedAt.Unix(),
		},
	}, nil
}

func (s *SettlementGRPCServer) GetSettlementDistributions(ctx context.Context, req *settlement.GetSettlementDistributionsRequest) (*settlement.GetSettlementDistributionsResponse, error) {
	s.logger.Info("Getting settlement distributions", zap.String("settlement_id", req.SettlementId))

	distributions, err := s.service.GetSettlementDistributions(ctx, req.SettlementId)
	if err != nil {
		s.logger.Error("Failed to get settlement distributions", zap.Error(err))
		return nil, fmt.Errorf("failed to get settlement distributions: %w", err)
	}

	protoDistributions := make([]*settlement.SettlementDistribution, len(distributions))
	for i, dist := range distributions {
		protoDistributions[i] = &settlement.SettlementDistribution{
			Id:           dist.ID,
			SettlementId: dist.SettlementID,
			UserId:       dist.UserID,
			Amount:       dist.Amount,
			Status:       dist.Status,
			ProcessedAt:  dist.ProcessedAt.Unix(),
		}
	}

	return &settlement.GetSettlementDistributionsResponse{
		Distributions: protoDistributions,
	}, nil
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Initialize dependencies
	metricsCollector := metrics.NewCollector()
	
	// Create repository (you'll need to implement this based on your existing settlement service)
	repo := repository.NewInMemorySettlementRepository()
	
	// Create service
	settlementService := service.NewSettlementService(repo, logger)
	
	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpc.UnaryServerInterceptor(metricsCollector, logger)),
	)
	
	// Register Settlement service
	settlementServer := NewSettlementGRPCServer(settlementService, logger)
	settlement.RegisterSettlementServiceServer(grpcServer, settlementServer)
	
	// Register health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("settlement.SettlementService", grpc_health_v1.HealthCheckResponse_SERVING)
	
	// Register reflection service for debugging
	reflection.Register(grpcServer)

	// Start listening
	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}

	logger.Info("Starting Settlement gRPC server on :50053")
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("Failed to serve", zap.Error(err))
	}
}