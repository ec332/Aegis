package main

import (
    "context"
    "fmt"
    "net"
    "os"
    "time"

    "go.uber.org/zap"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
    "google.golang.org/grpc/reflection"
    "google.golang.org/grpc/status"
    "google.golang.org/protobuf/types/known/timestamppb"

    grpcserver "github.com/aegis/shared/grpc"
    settlement "github.com/aegis/proto/gen/settlement"
    wallet "github.com/aegis/proto/gen/wallet"
)

type SettlementGRPCServer struct {
    settlement.UnimplementedSettlementServiceServer
    settlements     map[string]*settlement.Settlement
    distributions   map[string][]*settlement.SettlementDistribution
    processedSettlements map[string]bool // Track processed settlements for idempotency
    logger          *zap.Logger
    walletClient    wallet.WalletServiceClient
}

func NewSettlementGRPCServer(logger *zap.Logger, walletClient wallet.WalletServiceClient) *SettlementGRPCServer {
    return &SettlementGRPCServer{
        settlements:        make(map[string]*settlement.Settlement),
        distributions:      make(map[string][]*settlement.SettlementDistribution),
        processedSettlements: make(map[string]bool),
        logger:             logger,
        walletClient:       walletClient,
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
    st, ok := s.settlements[req.SettlementId]
    if !ok {
        return nil, status.Error(codes.NotFound, "settlement not found")
    }
    if st.Status != "completed" {
        return nil, status.Error(codes.FailedPrecondition, "settlement not completed")
    }
    
    // Idempotency check: return success if already processed
    if s.processedSettlements[req.SettlementId] {
        s.logger.Info("settlement already processed, returning idempotent response", 
            zap.String("settlement_id", req.SettlementId))
        return &settlement.ProcessPayoutResponse{
            Success: true,
            TransactionId: req.SettlementId,
            Message: "payouts already processed (idempotent)",
        }, nil
    }
    
    // Lookup distributions for this settlement (stubbed in memory)
    dists, ok := s.distributions[req.SettlementId]
    if !ok {
        return nil, status.Error(codes.NotFound, "no distributions for settlement")
    }
    
    // Track processed distributions to ensure idempotency even if some fail
    processedCount := 0
    
    // Credit each user
    for _, d := range dists {
        if d.Status == "pending" {
            // Get wallet account for user (stubbed: assume user_id == wallet_id)
            walletReq := &wallet.CreditWalletRequest{
                AccountId:   d.UserId,
                Amount:      d.Amount,
                MarketId:    st.MarketId,
                ReferenceId: d.Id, // idempotent key for wallet service
            }
            _, err := s.walletClient.CreditWallet(ctx, walletReq)
            if err != nil {
                s.logger.Error("credit wallet failed", zap.String("user_id", d.UserId), zap.Error(err))
                // Continue processing other distributions even if one fails
                // This ensures partial failure doesn't break idempotency
                continue
            }
            d.Status = "completed"
            d.UpdatedAt = timestamppb.Now()
            processedCount++
        } else if d.Status == "completed" {
            // Already processed this distribution, count it
            processedCount++
        }
    }
    
    // Mark settlement as processed only if all distributions are completed
    if processedCount == len(dists) {
        s.processedSettlements[req.SettlementId] = true
        s.logger.Info("settlement fully processed", 
            zap.String("settlement_id", req.SettlementId),
            zap.Int("distributions_processed", processedCount))
        return &settlement.ProcessPayoutResponse{
            Success: true,
            TransactionId: req.SettlementId,
            Message: "payouts processed",
        }, nil
    }
    
    // Partial processing - some distributions failed
    s.logger.Warn("partial payout processing", 
        zap.String("settlement_id", req.SettlementId),
        zap.Int("processed", processedCount),
        zap.Int("total", len(dists)))
    
    return &settlement.ProcessPayoutResponse{
        Success: false,
        TransactionId: req.SettlementId,
        Message: fmt.Sprintf("partial processing: %d/%d distributions completed", processedCount, len(dists)),
    }, nil
}

// Distributions listing is not part of current proto; omitted

func main() {
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    // Connect to wallet service
    walletAddr := os.Getenv("WALLET_GRPC_ADDR")
    if walletAddr == "" {
        walletAddr = "wallet-service:50052"
    }
    walletConn, err := grpc.Dial(walletAddr, grpc.WithInsecure())
    if err != nil {
        logger.Fatal("Failed to connect to wallet service", zap.String("addr", walletAddr), zap.Error(err))
    }
    defer walletConn.Close()
    walletClient := wallet.NewWalletServiceClient(walletConn)

    grpcServer := grpc.NewServer(
        grpc.UnaryInterceptor(grpcserver.UnaryServerInterceptor(logger, nil)),
    )
    settlementServer := NewSettlementGRPCServer(logger, walletClient)
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
