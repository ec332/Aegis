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
    wallet "github.com/aegis/proto/gen/wallet"
)

type WalletGRPCServer struct {
    wallet.UnimplementedWalletServiceServer
    accounts    map[string]*wallet.WalletAccount
    transactions map[string][]*wallet.WalletTransaction
    logger      *zap.Logger
}

func NewWalletGRPCServer(logger *zap.Logger) *WalletGRPCServer {
    return &WalletGRPCServer{
        accounts:     make(map[string]*wallet.WalletAccount),
        transactions: make(map[string][]*wallet.WalletTransaction),
        logger:       logger,
    }
}

func (s *WalletGRPCServer) CreateWalletAccount(ctx context.Context, req *wallet.CreateWalletAccountRequest) (*wallet.CreateWalletAccountResponse, error) {
    id := fmt.Sprintf("w-%d", time.Now().UnixNano())
    acc := &wallet.WalletAccount{
        Id:        id,
        UserId:    req.UserId,
        Currency:  req.Currency,
        Balance:   req.InitialBalance,
        Status:    "active",
        CreatedAt: time.Now().Unix(),
        UpdatedAt: time.Now().Unix(),
    }
    s.accounts[id] = acc
    return &wallet.CreateWalletAccountResponse{Account: acc}, nil
}

func (s *WalletGRPCServer) GetWalletAccount(ctx context.Context, req *wallet.GetWalletAccountRequest) (*wallet.GetWalletAccountResponse, error) {
    acc, ok := s.accounts[req.WalletId]
    if !ok {
        return nil, fmt.Errorf("not found")
    }
    return &wallet.GetWalletAccountResponse{Account: acc}, nil
}

func (s *WalletGRPCServer) Deposit(ctx context.Context, req *wallet.DepositRequest) (*wallet.DepositResponse, error) {
    acc, ok := s.accounts[req.WalletId]
    if !ok {
        return nil, fmt.Errorf("not found")
    }
    acc.Balance += req.Amount
    acc.UpdatedAt = time.Now().Unix()
    tx := &wallet.WalletTransaction{
        Id:           fmt.Sprintf("t-%d", time.Now().UnixNano()),
        WalletId:     req.WalletId,
        Type:         "deposit",
        Amount:       req.Amount,
        BalanceAfter: acc.Balance,
        Description:  "",
        Status:       "completed",
        CreatedAt:    time.Now().Unix(),
    }
    s.transactions[req.WalletId] = append(s.transactions[req.WalletId], tx)
    return &wallet.DepositResponse{Transaction: tx}, nil
}

func (s *WalletGRPCServer) Withdrawal(ctx context.Context, req *wallet.WithdrawalRequest) (*wallet.WithdrawalResponse, error) {
    acc, ok := s.accounts[req.WalletId]
    if !ok {
        return nil, fmt.Errorf("not found")
    }
    if acc.Balance < req.Amount {
        return nil, fmt.Errorf("insufficient funds")
    }
    acc.Balance -= req.Amount
    acc.UpdatedAt = time.Now().Unix()
    tx := &wallet.WalletTransaction{
        Id:           fmt.Sprintf("t-%d", time.Now().UnixNano()),
        WalletId:     req.WalletId,
        Type:         "withdrawal",
        Amount:       req.Amount,
        BalanceAfter: acc.Balance,
        Description:  "",
        Status:       "completed",
        CreatedAt:    time.Now().Unix(),
    }
    s.transactions[req.WalletId] = append(s.transactions[req.WalletId], tx)
    return &wallet.WithdrawalResponse{Transaction: tx}, nil
}

func (s *WalletGRPCServer) DebitWallet(ctx context.Context, req *wallet.DebitWalletRequest) (*wallet.DebitWalletResponse, error) {
    acc, ok := s.accounts[req.WalletId]
    if !ok {
        return nil, fmt.Errorf("not found")
    }
    if acc.Balance < req.Amount {
        return nil, fmt.Errorf("insufficient funds")
    }
    acc.Balance -= req.Amount
    acc.UpdatedAt = time.Now().Unix()
    tx := &wallet.WalletTransaction{
        Id:           fmt.Sprintf("t-%d", time.Now().UnixNano()),
        WalletId:     req.WalletId,
        Type:         "debit",
        Amount:       req.Amount,
        BalanceAfter: acc.Balance,
        Description:  req.Description,
        Status:       "completed",
        CreatedAt:    time.Now().Unix(),
    }
    s.transactions[req.WalletId] = append(s.transactions[req.WalletId], tx)
    return &wallet.DebitWalletResponse{Transaction: tx}, nil
}

func (s *WalletGRPCServer) CreditWallet(ctx context.Context, req *wallet.CreditWalletRequest) (*wallet.CreditWalletResponse, error) {
    acc, ok := s.accounts[req.WalletId]
    if !ok {
        return nil, fmt.Errorf("not found")
    }
    acc.Balance += req.Amount
    acc.UpdatedAt = time.Now().Unix()
    tx := &wallet.WalletTransaction{
        Id:           fmt.Sprintf("t-%d", time.Now().UnixNano()),
        WalletId:     req.WalletId,
        Type:         "credit",
        Amount:       req.Amount,
        BalanceAfter: acc.Balance,
        Description:  req.Description,
        Status:       "completed",
        CreatedAt:    time.Now().Unix(),
    }
    s.transactions[req.WalletId] = append(s.transactions[req.WalletId], tx)
    return &wallet.CreditWalletResponse{Transaction: tx}, nil
}

func main() {
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    metricsRegistry := metrics.NewRegistry(logger)
    grpcServer := grpc.NewServer(
        grpc.UnaryInterceptor(grpcserver.UnaryServerInterceptor(logger, grpcserver.NewServerMetrics("wallet", metricsRegistry))),
    )
    walletServer := NewWalletGRPCServer(logger)
    wallet.RegisterWalletServiceServer(grpcServer, walletServer)
    healthServer := health.NewServer()
    grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
    healthServer.SetServingStatus("wallet.WalletService", grpc_health_v1.HealthCheckResponse_SERVING)
    reflection.Register(grpcServer)

    lis, err := net.Listen("tcp", ":50052")
    if err != nil {
        logger.Fatal("Failed to listen", zap.Error(err))
    }

    logger.Info("Starting Wallet gRPC server on :50052")
    if err := grpcServer.Serve(lis); err != nil {
        logger.Fatal("Failed to serve", zap.Error(err))
    }
}