package config

import (
	"fmt"
	"os"
	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DatabaseURL string
	RedisURL    string
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

	return &Config{
		Port:        port,
		DatabaseURL: databaseURL,
		RedisURL:    redisURL,
	}, nil
}

// Fallback if no env variable is set
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
