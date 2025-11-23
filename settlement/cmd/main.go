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

    grpcserver "aegis/shared/grpc"
    "aegis/shared/metrics"
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
        CreatedAt:       time.Now().Unix(),
        UpdatedAt:       time.Now().Unix(),
    }
    s.settlements[id] = st
    return &settlement.CreateSettlementResponse{Settlement: st}, nil
}

func (s *SettlementGRPCServer) GetSettlement(ctx context.Context, req *settlement.GetSettlementRequest) (*settlement.GetSettlementResponse, error) {
    st, ok := s.settlements[req.SettlementId]
    if !ok {
        return nil, fmt.Errorf("not found")
    }
    return &settlement.GetSettlementResponse{Settlement: st}, nil
}

func (s *SettlementGRPCServer) CompleteSettlement(ctx context.Context, req *settlement.CompleteSettlementRequest) (*settlement.CompleteSettlementResponse, error) {
    st, ok := s.settlements[req.SettlementId]
    if !ok {
        return nil, fmt.Errorf("not found")
    }
    st.Status = "completed"
    st.UpdatedAt = time.Now().Unix()
    return &settlement.CompleteSettlementResponse{Settlement: st}, nil
}

func (s *SettlementGRPCServer) ProcessPayout(ctx context.Context, req *settlement.ProcessPayoutRequest) (*settlement.ProcessPayoutResponse, error) {
    st, ok := s.settlements[req.SettlementId]
    if !ok {
        return nil, fmt.Errorf("not found")
    }
    d := &settlement.SettlementDistribution{
        Id:           fmt.Sprintf("d-%d", time.Now().UnixNano()),
        SettlementId: st.Id,
        UserId:       req.UserId,
        Amount:       req.Amount,
        Status:       "processed",
        ProcessedAt:  time.Now().Unix(),
    }
    s.distributions[st.Id] = append(s.distributions[st.Id], d)
    return &settlement.ProcessPayoutResponse{Distribution: d}, nil
}

func (s *SettlementGRPCServer) GetSettlementDistributions(ctx context.Context, req *settlement.GetSettlementDistributionsRequest) (*settlement.GetSettlementDistributionsResponse, error) {
    list := s.distributions[req.SettlementId]
    return &settlement.GetSettlementDistributionsResponse{Distributions: list}, nil
}

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