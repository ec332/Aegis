package main

import (
    "context"
    "database/sql"
    "net"
    "os"
    "os/signal"
    "syscall"

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

func main() {
    logger, _ := zap.NewProduction()
    defer logger.Sync()

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
