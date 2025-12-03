package main

import (
    "context"
    "crypto/rand"
    "fmt"
    "net"
    "os"
    "strconv"
    "strings"
    "time"

    "go.uber.org/zap"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
    "google.golang.org/grpc/reflection"
    "google.golang.org/grpc/status"
    "google.golang.org/grpc/metadata"
    "google.golang.org/protobuf/types/known/timestamppb"

    "github.com/golang-jwt/jwt/v5"

    grpcserver "github.com/aegis/shared/grpc"
    market "github.com/aegis/proto/gen/market"
    settlement "github.com/aegis/proto/gen/settlement"
    wallet "github.com/aegis/proto/gen/wallet"
    transaction "github.com/aegis/proto/gen/transaction"
)

type SettlementGRPCServer struct {
    settlement.UnimplementedSettlementServiceServer
    settlements     map[string]*settlement.Settlement
    distributions   map[string][]*settlement.SettlementDistribution
    processedSettlements map[string]bool // Track processed settlements for idempotency
    logger          *zap.Logger
    walletClient    wallet.WalletServiceClient
    marketClient    market.MarketServiceClient
    transactionClient transaction.TransactionServiceClient
}

func NewSettlementGRPCServer(logger *zap.Logger, walletClient wallet.WalletServiceClient, marketClient market.MarketServiceClient, txnClient transaction.TransactionServiceClient) *SettlementGRPCServer {
    return &SettlementGRPCServer{
        settlements:        make(map[string]*settlement.Settlement),
        distributions:      make(map[string][]*settlement.SettlementDistribution),
        processedSettlements: make(map[string]bool),
        logger:             logger,
        walletClient:       walletClient,
        marketClient:       marketClient,
        transactionClient:  txnClient,
    }
}

func (s *SettlementGRPCServer) CreateSettlement(ctx context.Context, req *settlement.CreateSettlementRequest) (*settlement.CreateSettlementResponse, error) {
    id := fmt.Sprintf("s-%d", time.Now().UnixNano())
    st := &settlement.Settlement{
        Id:              id,
        MarketId:        req.MarketId,
        WinningOptionId: req.WinningOptionId,
        Status:          "resolving",
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
    // Resolve outcome via mocked oracle (random YES/NO)
    outcome, err := s.resolveOutcomeFromOracle(ctx, st.MarketId)
    if err != nil {
        return nil, status.Error(codes.Internal, "oracle resolution failed")
    }

    // Map outcome to actual option ID using Market service
    optID, err := s.mapOutcomeToOptionID(ctx, st.MarketId, outcome)
    if err != nil {
        s.logger.Warn("failed to map outcome to option id, falling back to existing", zap.String("market_id", st.MarketId), zap.Error(err))
    } else {
        st.WinningOptionId = optID
    }

    st.Status = "completed"
    st.UpdatedAt = timestamppb.Now()

    // Persist winner to Market service immediately (idempotent update)
    if st.WinningOptionId != "" {
        _, err := s.marketClient.UpdateMarket(ctx, &market.UpdateMarketRequest{Id: st.MarketId, Status: "resolved", Outcome: st.WinningOptionId})
        if err != nil {
            s.logger.Warn("failed to update market with winner", zap.String("market_id", st.MarketId), zap.Error(err))
        }
    }
    if _, exists := s.distributions[st.Id]; !exists {
        req := &transaction.GetTransactionsRequest{MarketId: &st.MarketId, Limit: 100000, Offset: 0}
        txResp, terr := s.transactionClient.GetTransactions(ctx, req)
        if terr != nil {
            s.logger.Warn("failed to fetch transactions for distribution", zap.String("market_id", st.MarketId), zap.Error(terr))
        } else {
            holdings := make(map[string]int64)
            for _, t := range txResp.Transactions {
                if strings.TrimSpace(t.OptionId) != strings.TrimSpace(st.WinningOptionId) {
                    continue
                }
                uid := strings.TrimSpace(t.UserId)
                if uid == "" { continue }
                shares := int64(t.NumberOfShares)
                if t.TransactionType == transaction.TransactionType_SELL {
                    holdings[uid] -= shares
                } else {
                    holdings[uid] += shares
                }
            }
            perShare := 1.0
            if v := strings.TrimSpace(os.Getenv("SETTLEMENT_PAYOUT_PER_SHARE")); v != "" {
                if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
                    perShare = f
                }
            }
            dists := make([]*settlement.SettlementDistribution, 0, len(holdings))
            for uid, sh := range holdings {
                if sh <= 0 { continue }
                did := fmt.Sprintf("d-%d", time.Now().UnixNano())
                amt := float64(sh) * perShare
                d := &settlement.SettlementDistribution{
                    Id:           did,
                    SettlementId: st.Id,
                    UserId:       uid,
                    Amount:       amt,
                    Status:       "pending",
                    CreatedAt:    timestamppb.Now(),
                    UpdatedAt:    timestamppb.Now(),
                }
                dists = append(dists, d)
            }
            if len(dists) > 0 {
                s.distributions[st.Id] = dists
            }
        }
    }
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
    
    dists, ok := s.distributions[req.SettlementId]
    if !ok {
        return nil, status.Error(codes.NotFound, "no distributions for settlement")
    }
    
    // Track processed distributions to ensure idempotency even if some fail
    processedCount := 0
    
    // Credit each user
    for _, d := range dists {
        if d.Status == "pending" {
            cur := strings.TrimSpace(os.Getenv("WALLET_DEFAULT_CURRENCY"))
            if cur == "" {
                cur = "USD"
            }
            ctxAuth := s.withAuth(ctx)
            accResp, aerr := s.walletClient.GetWalletAccountByUserID(ctxAuth, &wallet.GetWalletAccountByUserIDRequest{UserId: d.UserId, Currency: cur})
            if aerr != nil || accResp == nil || accResp.Account == nil {
                s.logger.Error("resolve wallet account failed", zap.String("user_id", d.UserId), zap.Error(aerr))
                continue
            }
            accID := accResp.Account.Id
            _, err := s.walletClient.Deposit(ctxAuth, &wallet.DepositRequest{AccountId: accID, Amount: d.Amount, ReferenceId: d.Id})
            if err != nil {
                s.logger.Error("deposit failed", zap.String("account_id", accID), zap.Error(err))
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

// resolveOutcomeFromOracle mocks a decentralized oracle by randomly returning YES or NO
func (s *SettlementGRPCServer) resolveOutcomeFromOracle(ctx context.Context, marketID string) (string, error) {
    b := make([]byte, 1)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    if int(b[0])%2 == 0 {
        return "YES", nil
    }
    return "NO", nil
}

// mapOutcomeToOptionID queries Market service to translate outcome text to the option ID
func (s *SettlementGRPCServer) mapOutcomeToOptionID(ctx context.Context, marketID string, outcome string) (string, error) {
    resp, err := s.marketClient.GetMarketOptions(ctx, &market.GetMarketOptionsRequest{MarketId: marketID})
    if err != nil {
        return "", err
    }
    target := strings.ToUpper(outcome)
    for _, opt := range resp.Options {
        if strings.ToUpper(opt.GetOptionText()) == target {
            return opt.GetId(), nil
        }
    }
    // Fallback: pick a random option if outcome text doesn't match
    if len(resp.Options) > 0 {
        b := make([]byte, 1)
        if _, err := rand.Read(b); err == nil {
            idx := int(b[0]) % len(resp.Options)
            return resp.Options[idx].GetId(), nil
        }
        // If randomness fails, choose the first option
        return resp.Options[0].GetId(), nil
    }
    return "", fmt.Errorf("no options available for market %s", marketID)
}

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

    // Connect to market service for outcome mapping
    marketAddr := os.Getenv("MARKET_GRPC_ADDR")
    if marketAddr == "" {
        marketAddr = os.Getenv("MARKET_SERVICE_GRPC_ADDR")
    }
    if marketAddr == "" {
        marketAddr = "market-service:50051"
    }
    marketConn, err := grpc.Dial(marketAddr, grpc.WithInsecure())
    if err != nil {
        logger.Fatal("Failed to connect to market service", zap.String("addr", marketAddr), zap.Error(err))
    }
    defer marketConn.Close()
    marketClient := market.NewMarketServiceClient(marketConn)

    // Connect to transaction service
    txnAddr := os.Getenv("TRANSACTION_GRPC_ADDR")
    if txnAddr == "" {
        txnAddr = os.Getenv("TRANSACTION_SERVICE_GRPC_ADDR")
    }
    if txnAddr == "" {
        txnAddr = "transaction-service:50052"
    }
    txnConn, err := grpc.Dial(txnAddr, grpc.WithInsecure())
    if err != nil {
        logger.Fatal("Failed to connect to transaction service", zap.String("addr", txnAddr), zap.Error(err))
    }
    defer txnConn.Close()
    txnClient := transaction.NewTransactionServiceClient(txnConn)

    grpcServer := grpc.NewServer(
        grpc.UnaryInterceptor(grpcserver.UnaryServerInterceptor(logger, nil)),
    )
    settlementServer := NewSettlementGRPCServer(logger, walletClient, marketClient, txnClient)
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
func (s *SettlementGRPCServer) withAuth(ctx context.Context) context.Context {
    if md, ok := metadata.FromIncomingContext(ctx); ok {
        if len(md.Get("authorization")) > 0 {
            return metadata.NewOutgoingContext(ctx, md)
        }
    }

    tok := strings.TrimSpace(os.Getenv("WALLET_INTERNAL_BEARER_TOKEN"))
    if tok == "" {
        tok = strings.TrimSpace(os.Getenv("WALLET_SERVICE_TOKEN"))
    }
    if tok != "" {
        low := strings.ToLower(tok)
        if !strings.HasPrefix(low, "bearer ") {
            tok = fmt.Sprintf("Bearer %s", tok)
        }
        return metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", tok))
    }

    secret := strings.TrimSpace(os.Getenv("AUTH_JWT_SECRET"))
    if secret == "" {
        secret = "dev-secret"
    }
    claims := jwt.MapClaims{
        "wallet": "internal-service",
        "exp":    time.Now().Add(10 * time.Minute).Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    signed, err := token.SignedString([]byte(secret))
    if err == nil {
        return metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", fmt.Sprintf("Bearer %s", signed)))
    }
    return ctx
}
