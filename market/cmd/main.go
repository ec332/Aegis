package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	market "github.com/aegis/proto/gen/market"
	grpcserver "github.com/aegis/shared/grpc"
	"github.com/aegis/shared/metrics"
	"github.com/aegis/shared/utils"
	"github.com/ec332/aegis/market/internal/api"
	marketgrpc "github.com/ec332/aegis/market/internal/grpc"
	"github.com/ec332/aegis/market/internal/repository"
	"github.com/ec332/aegis/market/internal/service"
	"github.com/ec332/aegis/market/pkg/config"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	log.Println("Configuration loaded")

	// Initialize logger
	logger, err := utils.NewLogger("market-service")
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()
	logger.Info("Logger initialized")

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

	// Initialize service
	svc := service.New(repo, redisClient)
	logger.Info("Service initialized")

	// Create metrics registry
	metricsRegistry := metrics.NewRegistry(logger)

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcserver.UnaryServerInterceptor(logger, grpcserver.NewServerMetrics("market", metricsRegistry))),
		grpc.StreamInterceptor(grpcserver.StreamServerInterceptor(logger, grpcserver.NewServerMetrics("market", metricsRegistry))),
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

	// Setup HTTP server
	r := chi.NewRouter()

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Register routes
	r.Post("/markets", api.CreateMarket(svc))
	r.Get("/markets", api.ListMarkets(svc))
	r.Get("/markets/{marketId}", api.GetMarket(svc))
	r.Put("/markets/{marketId}", api.UpdateMarket(svc))
	r.Get("/markets/{marketId}/stream", api.StreamLiquidityUpdates(svc))

	// LSMR endpoints
	r.Get("/markets/{marketId}/prices", api.GetMarketPrices(svc))
	r.Post("/markets/{marketId}/options/{optionId}/cost/buy", api.CalculateBuyCost(svc))
	r.Post("/markets/{marketId}/options/{optionId}/cost/sell", api.CalculateSellCost(svc))

	// User endpoints
	r.Post("/users", api.CreateUser(svc))
	r.Get("/users/{userId}", api.GetUser(svc))
	r.Put("/users/{userId}", api.UpdateUser(svc))
	r.Get("/users/wallet/{walletAddress}", api.GetUserByWallet(svc))

	// Create HTTP listener
	httpAddr := fmt.Sprintf(":%s", cfg.HTTPPort)
	httpServer := &http.Server{
		Addr:    httpAddr,
		Handler: r,
	}

	// Start HTTP server
	go func() {
		logger.Info("Market HTTP service starting", zap.String("address", httpAddr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to serve HTTP", zap.Error(err))
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
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", zap.Error(err))
	}

	// Wait for ongoing RPCs to complete
	<-shutdownCtx.Done()

	logger.Info("Server exited")
}
