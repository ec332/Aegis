package main

import (
<<<<<<< Updated upstream
    "context"
    "database/sql"
    "net"
    "os"
    "os/signal"
    "syscall"
=======
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"time"
>>>>>>> Stashed changes

    wallet "github.com/aegis/proto/gen/wallet"
    grpcserver "github.com/aegis/shared/grpc"
    "github.com/aegis/shared/metrics"
    "go.uber.org/zap"
    "google.golang.org/grpc"
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
    "google.golang.org/grpc/reflection"
    _ "github.com/lib/pq"

    wgrpc "aegis/wallet/internal/grpc"
    "aegis/wallet/internal/repository"
    "aegis/wallet/internal/service"
    "aegis/wallet/internal/auth"
)

<<<<<<< Updated upstream
=======
type WalletGRPCServer struct {
	wallet.UnimplementedWalletServiceServer
	db     *sql.DB
	logger *zap.Logger
}

func NewWalletGRPCServer(db *sql.DB, logger *zap.Logger) *WalletGRPCServer {
	return &WalletGRPCServer{
		db:     db,
		logger: logger,
	}
}

func (s *WalletGRPCServer) CreateWalletAccount(ctx context.Context, req *wallet.CreateWalletAccountRequest) (*wallet.CreateWalletAccountResponse, error) {
	// Check if wallet already exists for user
	queryCheck := `
		SELECT id, user_id, currency, balance, status, created_at, updated_at
		FROM wallet_accounts
		WHERE user_id = $1 AND currency = $2
	`
	var existingAcc wallet.WalletAccount
	var existingBalance float64
	var existingCreatedAt, existingUpdatedAt time.Time

	err := s.db.QueryRowContext(ctx, queryCheck, req.UserId, req.Currency).Scan(
		&existingAcc.Id, &existingAcc.UserId, &existingAcc.Currency, &existingBalance, &existingAcc.Status, &existingCreatedAt, &existingUpdatedAt,
	)

	if err == nil {
		s.logger.Info("Wallet already exists for user", zap.String("user_id", req.UserId), zap.String("wallet_id", existingAcc.Id))
		existingAcc.TotalBalance = existingBalance
		existingAcc.AvailableBalance = existingBalance
		existingAcc.CreatedAt = timestamppb.New(existingCreatedAt)
		existingAcc.UpdatedAt = timestamppb.New(existingUpdatedAt)
		return &wallet.CreateWalletAccountResponse{Account: &existingAcc}, nil
	} else if err != sql.ErrNoRows {
		s.logger.Error("Failed to check existing wallet", zap.Error(err))
		return nil, fmt.Errorf("failed to check existing wallet: %v", err)
	}

	// Create new wallet if none exists
	id := fmt.Sprintf("w-%d", time.Now().UnixNano())
	now := time.Now()

	queryInsert := `
		INSERT INTO wallet_accounts (id, user_id, currency, balance, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, currency, balance, status, created_at, updated_at
	`

	var acc wallet.WalletAccount
	var balance float64
	var createdAt, updatedAt time.Time

	err = s.db.QueryRowContext(ctx, queryInsert,
		id, req.UserId, req.Currency, 0.0, "active", now, now,
	).Scan(
		&acc.Id, &acc.UserId, &acc.Currency, &balance, &acc.Status, &createdAt, &updatedAt,
	)

	if err != nil {
		s.logger.Error("Failed to create wallet account", zap.Error(err))
		return nil, fmt.Errorf("failed to create wallet account: %v", err)
	}

	acc.TotalBalance = balance
	acc.AvailableBalance = balance
	acc.CreatedAt = timestamppb.New(createdAt)
	acc.UpdatedAt = timestamppb.New(updatedAt)

	s.logger.Info("Created new wallet", zap.String("user_id", req.UserId), zap.String("wallet_id", id))
	return &wallet.CreateWalletAccountResponse{Account: &acc}, nil
}

func (s *WalletGRPCServer) GetWalletAccount(ctx context.Context, req *wallet.GetWalletAccountRequest) (*wallet.GetWalletAccountResponse, error) {
	query := `
		SELECT id, user_id, currency, balance, status, created_at, updated_at
		FROM wallet_accounts
		WHERE id = $1
	`

	var acc wallet.WalletAccount
	var balance float64
	var createdAt, updatedAt time.Time

	err := s.db.QueryRowContext(ctx, query, req.Id).Scan(
		&acc.Id, &acc.UserId, &acc.Currency, &balance, &acc.Status, &createdAt, &updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("not found")
	} else if err != nil {
		s.logger.Error("Failed to get wallet account", zap.Error(err))
		return nil, fmt.Errorf("failed to get wallet account: %v", err)
	}

	acc.TotalBalance = balance
	acc.AvailableBalance = balance
	acc.CreatedAt = timestamppb.New(createdAt)
	acc.UpdatedAt = timestamppb.New(updatedAt)

	return &wallet.GetWalletAccountResponse{Account: &acc}, nil
}

func (s *WalletGRPCServer) updateBalance(ctx context.Context, walletID string, amount float64, txType string, refID string) (*wallet.WalletTransaction, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Lock the account and get current balance
	var currentBalance float64
	err = tx.QueryRowContext(ctx, "SELECT balance FROM wallet_accounts WHERE id = $1 FOR UPDATE", walletID).Scan(&currentBalance)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("not found")
	} else if err != nil {
		return nil, err
	}

	newBalance := currentBalance + amount
	if newBalance < 0 {
		return nil, fmt.Errorf("insufficient funds")
	}

	// Update balance
	_, err = tx.ExecContext(ctx, "UPDATE wallet_accounts SET balance = $1, updated_at = $2 WHERE id = $3", newBalance, time.Now(), walletID)
	if err != nil {
		return nil, err
	}

	// Create transaction record
	txID := fmt.Sprintf("t-%d", time.Now().UnixNano())
	now := time.Now()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO wallet_transactions (id, wallet_id, type, amount, balance_after, description, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, txID, walletID, txType, amount, newBalance, refID, "completed", now)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return &wallet.WalletTransaction{
		Id:          txID,
		WalletId:    walletID,
		Type:        txType,
		Amount:      amount,
		Status:      "completed",
		ReferenceId: refID,
		CreatedAt:   timestamppb.New(now),
		UpdatedAt:   timestamppb.New(now),
	}, nil
}

func (s *WalletGRPCServer) Deposit(ctx context.Context, req *wallet.DepositRequest) (*wallet.DepositResponse, error) {
	tx, err := s.updateBalance(ctx, req.AccountId, req.Amount, "deposit", req.ReferenceId)
	if err != nil {
		s.logger.Error("Deposit failed", zap.Error(err))
		return nil, err
	}
	return &wallet.DepositResponse{Transaction: tx}, nil
}

func (s *WalletGRPCServer) Withdrawal(ctx context.Context, req *wallet.WithdrawalRequest) (*wallet.WithdrawalResponse, error) {
	tx, err := s.updateBalance(ctx, req.AccountId, -req.Amount, "withdrawal", req.ReferenceId)
	if err != nil {
		s.logger.Error("Withdrawal failed", zap.Error(err))
		return nil, err
	}
	// Ensure returned transaction amount is positive for withdrawal
	tx.Amount = req.Amount
	return &wallet.WithdrawalResponse{Transaction: tx}, nil
}

func (s *WalletGRPCServer) DebitWallet(ctx context.Context, req *wallet.DebitWalletRequest) (*wallet.DebitWalletResponse, error) {
	tx, err := s.updateBalance(ctx, req.AccountId, -req.Amount, "debit", req.ReferenceId)
	if err != nil {
		s.logger.Error("Debit failed", zap.Error(err))
		return nil, err
	}
	tx.Amount = req.Amount
	return &wallet.DebitWalletResponse{Transaction: tx}, nil
}

func (s *WalletGRPCServer) CreditWallet(ctx context.Context, req *wallet.CreditWalletRequest) (*wallet.CreditWalletResponse, error) {
	tx, err := s.updateBalance(ctx, req.AccountId, req.Amount, "credit", req.ReferenceId)
	if err != nil {
		s.logger.Error("Credit failed", zap.Error(err))
		return nil, err
	}
	return &wallet.CreditWalletResponse{Transaction: tx}, nil
}

>>>>>>> Stashed changes
func main() {
    logger, _ := zap.NewProduction()
    defer logger.Sync()

<<<<<<< Updated upstream
    dbURL := getEnv("WALLET_DATABASE_URL", "postgres://postgres:postgres@localhost:5432/aegis?sslmode=disable")
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        logger.Fatal("db open failed", zap.Error(err))
    }
    defer db.Close()
    if err := db.Ping(); err != nil {
        logger.Fatal("db ping failed", zap.Error(err))
    }
    repo := repository.New(db)
    if err := repo.InitSchema(context.Background()); err != nil {
        logger.Fatal("schema init failed", zap.Error(err))
    }
=======
	// Database connection
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}

	metricsRegistry := metrics.NewRegistry(logger)
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcserver.UnaryServerInterceptor(logger, grpcserver.NewServerMetrics("wallet", metricsRegistry))),
	)
	walletServer := NewWalletGRPCServer(db, logger)
	wallet.RegisterWalletServiceServer(grpcServer, walletServer)
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("wallet.WalletService", grpc_health_v1.HealthCheckResponse_SERVING)
	reflection.Register(grpcServer)
>>>>>>> Stashed changes

    svc := service.New(repo, logger)
    tm := auth.NewTokenManager()

    metricsRegistry := metrics.NewRegistry(logger)
    grpcServer := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            wgrpc.AuthUnaryInterceptor(logger, tm),
            grpcserver.UnaryServerInterceptor(logger, grpcserver.NewServerMetrics("wallet", metricsRegistry)),
            grpcserver.RecoveryInterceptor(logger),
        ),
        grpc.ChainStreamInterceptor(
            grpcserver.StreamServerInterceptor(logger, grpcserver.NewServerMetrics("wallet", metricsRegistry)),
            grpcserver.StreamRecoveryInterceptor(logger),
        ),
    )

    server := wgrpc.NewServer(svc, logger, tm)
    wallet.RegisterWalletServiceServer(grpcServer, server)

    healthServer := health.NewServer()
    grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
    healthServer.SetServingStatus("wallet.WalletService", grpc_health_v1.HealthCheckResponse_SERVING)
    reflection.Register(grpcServer)

    port := getEnv("WALLET_SERVICE_PORT", "50052")
    lis, err := net.Listen("tcp", ":"+port)
    if err != nil {
        logger.Fatal("listen failed", zap.Error(err))
    }

    go func() {
        if err := grpcServer.Serve(lis); err != nil {
            logger.Fatal("serve failed", zap.Error(err))
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    healthServer.SetServingStatus("wallet.WalletService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
    grpcServer.GracefulStop()
}

func getEnv(k, d string) string {
    v := os.Getenv(k)
    if v == "" {
        return d
    }
    return v
}
