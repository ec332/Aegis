package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aegis/shared/circuitbreaker"
	"github.com/aegis/shared/grpc"
	"github.com/aegis/shared/kafka"
	"github.com/aegis/shared/metrics"
	"go.uber.org/zap"
)

// Example usage of the resilient gRPC client with circuit breaker and Kafka fallback
func main() {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Initialize metrics registry
	metricsRegistry := metrics.NewRegistry(logger)

	// Configure the resilient client for Market Service
	marketConfig := grpc.DefaultClientConfig("market", "localhost:50051")
	marketConfig.Timeout = 1 * time.Second // 1-second timeout as required
	marketConfig.MaxRetries = 3
	marketConfig.KafkaFallback = true
	marketConfig.KafkaConfig = kafka.Config{
		Brokers:      []string{"localhost:9092"},
		DialTimeout:  10 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Create market service client
	marketClient, err := grpc.NewResilientClient(marketConfig, logger, metricsRegistry)
	if err != nil {
		logger.Fatal("Failed to create market client", zap.Error(err))
	}
	defer marketClient.Close()

	// Example: Making a resilient gRPC call with circuit breaker and Kafka fallback
	ctx := context.Background()
	
	// Simulate a market request
	marketRequest := map[string]interface{}{
		"market_id": "market-123",
		"user_id":   "user-456",
	}
	
	marketResponse := map[string]interface{}{}
	
	// Invoke the gRPC method with automatic circuit breaker and retry logic
	err = marketClient.Invoke(ctx, "/market.MarketService/GetMarket", marketRequest, marketResponse)
	
	if err != nil {
		if err == grpc.ErrKafkaFallback {
			// Request was successfully queued to Kafka for asynchronous processing
			logger.Info("Market request queued to Kafka fallback due to gRPC timeout or circuit breaker open",
				zap.String("market_id", "market-123"),
				zap.String("method", "GetMarket"))
			fmt.Println("Request queued to Kafka - will be processed asynchronously")
		} else {
			logger.Error("Market request failed completely", zap.Error(err))
			fmt.Printf("Request failed: %v\n", err)
		}
	} else {
		logger.Info("Market request completed successfully",
			zap.String("market_id", "market-123"),
			zap.Any("response", marketResponse))
		fmt.Printf("Market response: %+v\n", marketResponse)
	}

	// Check circuit breaker status
	cb := marketClient.GetCircuitBreaker()
	logger.Info("Circuit breaker status",
		zap.String("state", cb.GetState().String()))

	// Check metrics
	serviceMetrics := marketClient.GetMetrics()
	logger.Info("Service metrics collected")

	// Configure wallet service client
	walletConfig := grpc.DefaultClientConfig("wallet", "localhost:50052")
	walletConfig.Timeout = 1 * time.Second
	walletConfig.KafkaFallback = true

	walletClient, err := grpc.NewResilientClient(walletConfig, logger, metricsRegistry)
	if err != nil {
		logger.Fatal("Failed to create wallet client", zap.Error(err))
	}
	defer walletClient.Close()

	// Example: Wallet operation with automatic retry and circuit breaker
	walletRequest := map[string]interface{}{
		"user_id": "user-456",
		"amount":  100.0,
	}
	
	walletResponse := map[string]interface{}{}
	
	err = walletClient.Invoke(ctx, "/wallet.WalletService/Deposit", walletRequest, walletResponse)
	
	if err != nil {
		if err == grpc.ErrKafkaFallback {
			logger.Info("Wallet deposit queued to Kafka fallback",
				zap.String("user_id", "user-456"),
				zap.Float64("amount", 100.0))
			fmt.Println("Wallet deposit queued to Kafka")
		} else {
			logger.Error("Wallet deposit failed", zap.Error(err))
		}
	} else {
		logger.Info("Wallet deposit completed successfully",
			zap.String("user_id", "user-456"),
			zap.Float64("amount", 100.0))
		fmt.Printf("Wallet response: %+v\n", walletResponse)
	}

	// Simulate circuit breaker scenario
	fmt.Println("\n--- Circuit Breaker Simulation ---")
	
	// Force multiple failures to open the circuit
	for i := 0; i < 5; i++ {
		// Simulate a failing service
		failingRequest := map[string]interface{}{"fail": true}
		failingResponse := map[string]interface{}{}
		
		err := marketClient.Invoke(ctx, "/market.MarketService/GetMarket", failingRequest, failingResponse)
		
		if err != nil {
			if err == grpc.ErrKafkaFallback {
				fmt.Printf("Request %d: Queued to Kafka fallback\n", i+1)
			} else if errors.Is(err, circuitbreaker.ErrCircuitOpen) {
				fmt.Printf("Request %d: Circuit breaker is OPEN - request rejected immediately\n", i+1)
			} else {
				fmt.Printf("Request %d: Failed with error: %v\n", i+1, err)
			}
		}
	}

	// Wait for circuit breaker timeout and test recovery
	fmt.Println("\nWaiting for circuit breaker timeout...")
	time.Sleep(2 * time.Second)
	
	// This should now go through (circuit in half-open state)
	successRequest := map[string]interface{}{"market_id": "market-recovery"}
	successResponse := map[string]interface{}{}
	
	err = marketClient.Invoke(ctx, "/market.MarketService/GetMarket", successRequest, successResponse)
	
	if err != nil {
		if err == grpc.ErrKafkaFallback {
			fmt.Println("Recovery test: Request queued to Kafka fallback")
		} else {
			fmt.Printf("Recovery test failed: %v\n", err)
		}
	} else {
		fmt.Println("Recovery test: Request succeeded - circuit breaker closed!")
	}

	fmt.Println("\n--- Example Complete ---")
	fmt.Println("The resilient gRPC client provides:")
	fmt.Println("✓ Automatic circuit breaker with 1-second timeout")
	fmt.Println("✓ Kafka fallback when gRPC calls timeout or fail")
	fmt.Println("✓ Exponential backoff retry mechanism")
	fmt.Println("✓ Comprehensive metrics and observability")
	fmt.Println("✓ Service-specific topic routing for Kafka messages")
}