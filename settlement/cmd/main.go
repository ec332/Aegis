package main

import (
    "context"
    "fmt"
    "net"
    "time"

    "go.uber.org/zap"
    "google.golang.org/grpc"
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
    "google.golang.org/grpc/reflection"
    "google.golang.org/protobuf/types/known/timestamppb"

    grpcserver "github.com/aegis/shared/grpc"
    "github.com/aegis/shared/metrics"
    settlement "github.com/aegis/proto/gen/settlement"
)

type SettlementGRPCServer struct {
    settlement.UnimplementedSettlementServiceServer
    settlements     map[string]*settlement.Settlement
    distributions   map[string][]*settlement.SettlementDistribution
    logger          *zap.Logger
}

func NewSettlementGRPCServer(logger *zap.Logger) *SettlementGRPCServer {
    return &SettlementGRPCServer{
        settlements:   make(map[string]*settlement.Settlement),
        distributions: make(map[string][]*settlement.SettlementDistribution),
        logger:        logger,
    }
}

func (s *SettlementGRPCServer) CreateSettlement(ctx context.Context, req *settlement.CreateSettlementRequest) (*settlement.CreateSettlementResponse, error) {
    id := fmt.Sprintf("s-%d", time.Now().UnixNano())
    st := &settlement.Settlement{
        Id:              id,
        MarketId:        req.MarketId,
        WinningOptionId: req.WinningOptionId,
        Status:          "pending",
        CreatedAt:       timestamppb.Now(),
        UpdatedAt:       timestamppb.Now(),
    }
    s.settlements[id] = st
    return &settlement.CreateSettlementResponse{Settlement: st}, nil
}

func (s *SettlementGRPCServer) GetSettlement(ctx context.Context, req *settlement.GetSettlementRequest) (*settlement.GetSettlementResponse, error) {
    st, ok := s.settlements[req.Id]
    if !ok {
        return nil, fmt.Errorf("not found")
    }
    return &settlement.GetSettlementResponse{Settlement: st}, nil
}

func (s *SettlementGRPCServer) CompleteSettlement(ctx context.Context, req *settlement.CompleteSettlementRequest) (*settlement.CompleteSettlementResponse, error) {
    st, ok := s.settlements[req.Id]
    if !ok {
        return nil, fmt.Errorf("not found")
    }
    st.Status = "completed"
    st.UpdatedAt = timestamppb.Now()
    return &settlement.CompleteSettlementResponse{Settlement: st}, nil
}

func (s *SettlementGRPCServer) ProcessPayout(ctx context.Context, req *settlement.ProcessPayoutRequest) (*settlement.ProcessPayoutResponse, error) {
    if _, ok := s.settlements[req.SettlementId]; !ok {
        return nil, fmt.Errorf("not found")
    }
    txID := fmt.Sprintf("tx-%d", time.Now().UnixNano())
    return &settlement.ProcessPayoutResponse{Success: true, TransactionId: txID, Message: "payout processed"}, nil
}

// Distributions listing is not part of current proto; omitted

func main() {
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    metricsRegistry := metrics.NewRegistry(logger)
    grpcServer := grpc.NewServer(
        grpc.UnaryInterceptor(grpcserver.UnaryServerInterceptor(logger, grpcserver.NewServerMetrics("settlement", metricsRegistry))),
    )
    settlementServer := NewSettlementGRPCServer(logger)
    settlement.RegisterSettlementServiceServer(grpcServer, settlementServer)
    healthServer := health.NewServer()
    grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
    healthServer.SetServingStatus("settlement.SettlementService", grpc_health_v1.HealthCheckResponse_SERVING)
    reflection.Register(grpcServer)
    lis, err := net.Listen("tcp", ":50053")
    if err != nil {
        logger.Fatal("Failed to listen", zap.Error(err))
    }
    logger.Info("Starting Settlement gRPC server on :50053")
    if err := grpcServer.Serve(lis); err != nil {
        logger.Fatal("Failed to serve", zap.Error(err))
    }
}