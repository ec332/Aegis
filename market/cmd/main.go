package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "net"
    "os"
    "os/signal"
    "syscall"
    "time"

    market "github.com/aegis/proto/gen/market"
    settlement "github.com/aegis/proto/gen/settlement"
    grpcserver "github.com/aegis/shared/grpc"
    "github.com/aegis/shared/utils"
    marketgrpc "github.com/ec332/aegis/market/internal/grpc"
    "github.com/ec332/aegis/market/internal/repository"
    "github.com/ec332/aegis/market/internal/service"
    "github.com/ec332/aegis/market/pkg/config"
    _ "github.com/lib/pq"
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
    "google.golang.org/grpc"
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	// Initialize logger first
	logger, err := utils.NewLogger("market-service")
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()
	logger.Info("Logger initialized")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}
	logger.Info("Configuration loaded")

	// Initialize database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		logger.Fatal("Failed to ping database", zap.Error(err))
	}
	logger.Info("PostgreSQL connected")

	// Initialize schema
	repo := repository.New(db)
	ctx := context.Background()
	if err := repo.InitSchema(ctx); err != nil {
		logger.Fatal("Failed to initialize schema", zap.Error(err))
	}
	logger.Info("Database schema initialized")

	// Initialize Redis client
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.Fatal("Failed to parse Redis URL", zap.Error(err))
	}
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	logger.Info("Redis connected")

    settlementConn, err := grpc.Dial(cfg.SettlementGRPCAddr, grpc.WithInsecure())
    if err != nil {
        logger.Fatal("Failed to connect to settlement service", zap.Error(err))
    }
    defer settlementConn.Close()
    settlementClient := settlement.NewSettlementServiceClient(settlementConn)

    svc := service.New(repo, redisClient, logger, settlementClient)
    logger.Info("Service initialized")

    // Create gRPC server
    grpcServer := grpc.NewServer(
        grpc.UnaryInterceptor(grpcserver.UnaryServerInterceptor(logger, nil)),
        grpc.StreamInterceptor(grpcserver.StreamServerInterceptor(logger, nil)),
    )

	// Register market service
	marketServer := marketgrpc.NewServer(svc, logger)
	market.RegisterMarketServiceServer(grpcServer, marketServer)

	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// Set health status
	healthServer.SetServingStatus("market.MarketService", grpc_health_v1.HealthCheckResponse_SERVING)

	// Create listener
	addr := fmt.Sprintf(":%s", cfg.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatal("Failed to create listener", zap.Error(err))
	}

    // Start gRPC server
    go func() {
        logger.Info("Market gRPC service starting", zap.String("address", addr))
        if err := grpcServer.Serve(lis); err != nil {
            logger.Fatal("Failed to serve gRPC", zap.Error(err))
        }
    }()

    // Run an immediate settlement sweep on startup to catch backlog
    go func() {
        time.Sleep(3 * time.Second)
        ctxInit, cancel := context.WithTimeout(context.Background(), 20*time.Second)
        now := time.Now()
        _ = svc.TriggerSettlementsForExpiredMarkets(ctxInit, now.Add(-24*time.Hour), now)
        cancel()
    }()

    settleTicker := time.NewTicker(1 * time.Minute)
    go func() {
        for range settleTicker.C {
            ctxTick, cancel := context.WithTimeout(context.Background(), 20*time.Second)
            since := time.Now().Add(-1 * time.Minute)
            until := time.Now()
            _ = svc.TriggerSettlementsForExpiredMarkets(ctxTick, since, until)
            cancel()
        }
    }()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Set health status to not serving
	healthServer.SetServingStatus("market.MarketService", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

    // Stop accepting new connections
    grpcServer.GracefulStop()

    settleTicker.Stop()

	// Wait for ongoing RPCs to complete
	<-shutdownCtx.Done()

	logger.Info("Server exited")
}
