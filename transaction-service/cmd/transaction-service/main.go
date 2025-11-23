package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/spf13/viper"
    "go.uber.org/zap"

    httpHandlers "transaction-service/internal/http"
    "transaction-service/internal/log"
)

func main() {
    viper.SetEnvPrefix("APP")
    viper.AutomaticEnv()
    viper.SetDefault("HTTP_PORT", 5555)
    viper.SetDefault("DB_HOST", "localhost")
    viper.SetDefault("DB_PORT", 5432)
    viper.SetDefault("DB_NAME", "transaction")
    viper.SetDefault("DB_USER", "postgres")
    viper.SetDefault("DB_PASSWORD", "postgres")

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

    r := chi.NewRouter()
    r.Use(middleware.RequestID)
    r.Use(middleware.RealIP)
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(middleware.Timeout(60 * time.Second))

    r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("{\"status\":\"healthy\"}"))
    })

    handler := httpHandlers.New(pool, logger)
    handler.RegisterRoutes(r)

    port := viper.GetInt("HTTP_PORT")
    addr := fmt.Sprintf(":%d", port)
    logger.Info("starting server", zap.String("addr", addr))
    if err := http.ListenAndServe(addr, r); err != nil && err != http.ErrServerClosed {
        logger.Fatal("server error", zap.Error(err))
    }
    _ = os.Stderr
}