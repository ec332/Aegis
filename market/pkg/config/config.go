package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
    Port        string
    HTTPPort    string
    DatabaseURL string
    RedisURL    string
    SettlementGRPCAddr string
}

// Load loads configuration from env variables
func Load() (*Config, error) {
    _ = godotenv.Load()

    port := getEnv("PORT", "8080")
    databaseURL := getEnv("DATABASE_URL", "")
    if databaseURL == "" {
        return nil, fmt.Errorf("DATABASE_URL is required")
    }

    redisURL := getEnv("REDIS_URL", "redis://localhost:6379")

    httpPort := getEnv("HTTP_PORT", "8081")
    settlementAddr := getEnv("SETTLEMENT_SERVICE_GRPC_ADDR", "settlement-service:50053")

    return &Config{
        Port:        port,
        HTTPPort:    httpPort,
        DatabaseURL: databaseURL,
        RedisURL:    redisURL,
        SettlementGRPCAddr: settlementAddr,
    }, nil
}

// Fallback if no env variable is set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
