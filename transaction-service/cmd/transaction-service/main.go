package main

import (
    "context"
    "fmt"
    "net"
    "os"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/spf13/viper"
    "go.uber.org/zap"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    "github.com/aegis/proto/gen/market"
    pb "github.com/aegis/proto/gen/transaction"
    grpcServer "transaction-service/internal/grpc"
    "transaction-service/internal/log"
    "transaction-service/internal/service"
    "transaction-service/internal/store/postgres"
)

func main() {
    viper.SetEnvPrefix("APP")
    viper.AutomaticEnv()
    viper.SetDefault("GRPC_PORT", 50052)
    viper.SetDefault("DB_HOST", "localhost")
    viper.SetDefault("DB_PORT", 5432)
    viper.SetDefault("DB_NAME", "transaction")
    viper.SetDefault("DB_USER", "postgres")
    viper.SetDefault("DB_PASSWORD", "postgres")
    viper.SetDefault("MARKET_GRPC_ADDR", "localhost:50051")

    logger := log.New()
    defer logger.Sync()

    dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
        viper.GetString("DB_USER"),
        viper.GetString("DB_PASSWORD"),
        viper.GetString("DB_HOST"),
        viper.GetInt("DB_PORT"),
        viper.GetString("DB_NAME"),
    )

    cfg, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        logger.Fatal("failed to parse db config", zap.Error(err))
    }
    cfg.MaxConns = 10
    pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
    if err != nil {
        logger.Fatal("failed to connect to db", zap.Error(err))
    }
    defer pool.Close()

    // Create market gRPC client
    marketAddr := viper.GetString("MARKET_GRPC_ADDR")
    marketConn, err := grpc.Dial(marketAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        logger.Fatal("failed to connect to market service", zap.String("addr", marketAddr), zap.Error(err))
    }
    defer marketConn.Close()
    
    marketClient := market.NewMarketServiceClient(marketConn)
    logger.Info("connected to market service", zap.String("addr", marketAddr))

    // Create transaction service
    repo := postgres.New(pool)
    svc := service.NewTransactionService(repo)

    // Create gRPC server
    grpcPort := viper.GetInt("GRPC_PORT")
    lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
    if err != nil {
        logger.Fatal("failed to listen", zap.Error(err))
    }

    s := grpc.NewServer()
    pb.RegisterTransactionServiceServer(s, grpcServer.NewTransactionGRPCServer(svc, logger, marketClient))

    logger.Info("starting gRPC server", zap.Int("port", grpcPort))
    if err := s.Serve(lis); err != nil {
        logger.Fatal("failed to serve", zap.Error(err))
    }
    _ = os.Stderr
}