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
    "github.com/aegis/proto/gen/wallet"
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
    viper.SetDefault("MARKET_SERVICE_GRPC_ADDR", "market-service:50051")
    viper.SetDefault("WALLET_GRPC_ADDR", "wallet-service:50052")
    viper.SetDefault("WALLET_SERVICE_GRPC_ADDR", "wallet-service:50052")
    viper.SetDefault("WALLET_DEFAULT_CURRENCY", "USD")

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
    marketAddr := firstNonEmpty(
        viper.GetString("MARKET_SERVICE_GRPC_ADDR"),
        viper.GetString("MARKET_GRPC_ADDR"),
    )
    marketConn, err := grpc.Dial(marketAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        logger.Fatal("failed to connect to market service", zap.String("addr", marketAddr), zap.Error(err))
    }
    defer marketConn.Close()
    
    marketClient := market.NewMarketServiceClient(marketConn)
    logger.Info("connected to market service", zap.String("addr", marketAddr))

    // Create wallet gRPC client
    walletAddr := firstNonEmpty(
        viper.GetString("WALLET_SERVICE_GRPC_ADDR"),
        viper.GetString("WALLET_GRPC_ADDR"),
    )
    walletConn, err := grpc.Dial(walletAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
    if err != nil {
        logger.Fatal("failed to connect to wallet service", zap.String("addr", walletAddr), zap.Error(err))
    }
    defer walletConn.Close()
    walletClient := wallet.NewWalletServiceClient(walletConn)
    logger.Info("connected to wallet service", zap.String("addr", walletAddr))

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
    defaultCurrency := viper.GetString("WALLET_DEFAULT_CURRENCY")
    pb.RegisterTransactionServiceServer(s, grpcServer.NewTransactionGRPCServer(svc, logger, marketClient, walletClient, defaultCurrency))

    logger.Info("starting gRPC server", zap.Int("port", grpcPort))
    if err := s.Serve(lis); err != nil {
        logger.Fatal("failed to serve", zap.Error(err))
    }
    _ = os.Stderr
}

func firstNonEmpty(values ...string) string {
    for _, v := range values {
        if v != "" {
            return v
        }
    }
    return ""
}
