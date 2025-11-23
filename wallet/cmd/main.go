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
	"aegis/wallet/internal/service"
	"aegis/wallet/internal/repository"
	"aegis/proto/wallet"
)

type WalletGRPCServer struct {
	wallet.UnimplementedWalletServiceServer
	service *service.WalletService
	logger  *zap.Logger
}

func NewWalletGRPCServer(walletService *service.WalletService, logger *zap.Logger) *WalletGRPCServer {
	return &WalletGRPCServer{
		service: walletService,
		logger:  logger,
	}
}

func (s *WalletGRPCServer) CreateWalletAccount(ctx context.Context, req *wallet.CreateWalletAccountRequest) (*wallet.CreateWalletAccountResponse, error) {
	s.logger.Info("Creating wallet account",
		zap.String("user_id", req.UserId),
		zap.String("currency", req.Currency),
	)

	account, err := s.service.CreateWalletAccount(ctx, req.UserId, req.Currency, req.InitialBalance)
	if err != nil {
		s.logger.Error("Failed to create wallet account", zap.Error(err))
		return nil, fmt.Errorf("failed to create wallet account: %w", err)
	}

	return &wallet.CreateWalletAccountResponse{
		Account: &wallet.WalletAccount{
			Id:        account.ID,
			UserId:    account.UserID,
			Currency:  account.Currency,
			Balance:   account.Balance,
			Status:    account.Status,
			CreatedAt: account.CreatedAt.Unix(),
			UpdatedAt: account.UpdatedAt.Unix(),
		},
	}, nil
}

func (s *WalletGRPCServer) GetWalletAccount(ctx context.Context, req *wallet.GetWalletAccountRequest) (*wallet.GetWalletAccountResponse, error) {
	s.logger.Info("Getting wallet account", zap.String("wallet_id", req.WalletId))

	account, err := s.service.GetWalletAccount(ctx, req.WalletId)
	if err != nil {
		s.logger.Error("Failed to get wallet account", zap.Error(err))
		return nil, fmt.Errorf("failed to get wallet account: %w", err)
	}

	return &wallet.GetWalletAccountResponse{
		Account: &wallet.WalletAccount{
			Id:        account.ID,
			UserId:    account.UserID,
			Currency:  account.Currency,
			Balance:   account.Balance,
			Status:    account.Status,
			CreatedAt: account.CreatedAt.Unix(),
			UpdatedAt: account.UpdatedAt.Unix(),
		},
	}, nil
}

func (s *WalletGRPCServer) Deposit(ctx context.Context, req *wallet.DepositRequest) (*wallet.DepositResponse, error) {
	s.logger.Info("Processing deposit",
		zap.String("wallet_id", req.WalletId),
		zap.Float64("amount", req.Amount),
	)

	transaction, err := s.service.Deposit(ctx, req.WalletId, req.Amount)
	if err != nil {
		s.logger.Error("Failed to process deposit", zap.Error(err))
		return nil, fmt.Errorf("failed to process deposit: %w", err)
	}

	return &wallet.DepositResponse{
		Transaction: &wallet.WalletTransaction{
			Id:              transaction.ID,
			WalletId:        transaction.WalletID,
			Type:            transaction.Type,
			Amount:          transaction.Amount,
			BalanceAfter:    transaction.BalanceAfter,
			Description:     transaction.Description,
			Status:          transaction.Status,
			CreatedAt:       transaction.CreatedAt.Unix(),
		},
	}, nil
}

func (s *WalletGRPCServer) Withdrawal(ctx context.Context, req *wallet.WithdrawalRequest) (*wallet.WithdrawalResponse, error) {
	s.logger.Info("Processing withdrawal",
		zap.String("wallet_id", req.WalletId),
		zap.Float64("amount", req.Amount),
	)

	transaction, err := s.service.Withdrawal(ctx, req.WalletId, req.Amount)
	if err != nil {
		s.logger.Error("Failed to process withdrawal", zap.Error(err))
		return nil, fmt.Errorf("failed to process withdrawal: %w", err)
	}

	return &wallet.WithdrawalResponse{
		Transaction: &wallet.WalletTransaction{
			Id:              transaction.ID,
			WalletId:        transaction.WalletID,
			Type:            transaction.Type,
			Amount:          transaction.Amount,
			BalanceAfter:    transaction.BalanceAfter,
			Description:     transaction.Description,
			Status:          transaction.Status,
			CreatedAt:       transaction.CreatedAt.Unix(),
		},
	}, nil
}

func (s *WalletGRPCServer) DebitWallet(ctx context.Context, req *wallet.DebitWalletRequest) (*wallet.DebitWalletResponse, error) {
	s.logger.Info("Processing wallet debit",
		zap.String("wallet_id", req.WalletId),
		zap.Float64("amount", req.Amount),
	)

	transaction, err := s.service.DebitWallet(ctx, req.WalletId, req.Amount, req.Description)
	if err != nil {
		s.logger.Error("Failed to debit wallet", zap.Error(err))
		return nil, fmt.Errorf("failed to debit wallet: %w", err)
	}

	return &wallet.DebitWalletResponse{
		Transaction: &wallet.WalletTransaction{
			Id:              transaction.ID,
			WalletId:        transaction.WalletID,
			Type:            transaction.Type,
			Amount:          transaction.Amount,
			BalanceAfter:    transaction.BalanceAfter,
			Description:     transaction.Description,
			Status:          transaction.Status,
			CreatedAt:       transaction.CreatedAt.Unix(),
		},
	}, nil
}

func (s *WalletGRPCServer) CreditWallet(ctx context.Context, req *wallet.CreditWalletRequest) (*wallet.CreditWalletResponse, error) {
	s.logger.Info("Processing wallet credit",
		zap.String("wallet_id", req.WalletId),
		zap.Float64("amount", req.Amount),
	)

	transaction, err := s.service.CreditWallet(ctx, req.WalletId, req.Amount, req.Description)
	if err != nil {
		s.logger.Error("Failed to credit wallet", zap.Error(err))
		return nil, fmt.Errorf("failed to credit wallet: %w", err)
	}

	return &wallet.CreditWalletResponse{
		Transaction: &wallet.WalletTransaction{
			Id:              transaction.ID,
			WalletId:        transaction.WalletID,
			Type:            transaction.Type,
			Amount:          transaction.Amount,
			BalanceAfter:    transaction.BalanceAfter,
			Description:     transaction.Description,
			Status:          transaction.Status,
			CreatedAt:       transaction.CreatedAt.Unix(),
		},
	}, nil
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Initialize dependencies
	metricsCollector := metrics.NewCollector()
	
	// Create repository (you'll need to implement this based on your existing wallet service)
	repo := repository.NewInMemoryWalletRepository()
	
	// Create service
	walletService := service.NewWalletService(repo, logger)
	
	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpc.UnaryServerInterceptor(metricsCollector, logger)),
	)
	
	// Register Wallet service
	walletServer := NewWalletGRPCServer(walletService, logger)
	wallet.RegisterWalletServiceServer(grpcServer, walletServer)
	
	// Register health check service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("wallet.WalletService", grpc_health_v1.HealthCheckResponse_SERVING)
	
	// Register reflection service for debugging
	reflection.Register(grpcServer)

	// Start listening
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		logger.Fatal("Failed to listen", zap.Error(err))
	}

	logger.Info("Starting Wallet gRPC server on :50052")
	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("Failed to serve", zap.Error(err))
	}
}