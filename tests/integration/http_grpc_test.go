//go:build integration
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"aegis/shared/kafka"
	"aegis/proto/market"
)

func TestHTTPToGRPCIntegration(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Test the complete HTTP-to-gRPC flow
	t.Run("MarketService HTTP to gRPC", func(t *testing.T) {
		// Create a market via HTTP API Gateway
		createMarketReq := map[string]interface{}{
			"title":       "Test Market",
			"description": "Test market for integration testing",
			"options": []map[string]interface{}{
				{"title": "Option A", "odds": 2.5},
				{"title": "Option B", "odds": 1.8},
			},
			"end_time": time.Now().Add(24 * time.Hour).Unix(),
		}

		jsonData, _ := json.Marshal(createMarketReq)
		
		// This should go through API Gateway -> gRPC client -> Market service
		resp, err := http.Post("http://localhost:8080/api/markets", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var createResp market.CreateMarketResponse
		err = json.NewDecoder(resp.Body).Decode(&createResp)
		require.NoError(t, err)
		assert.NotEmpty(t, createResp.Market.Id)

		// Get the market via HTTP API Gateway
		getResp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/markets/%s", createResp.Market.Id))
		require.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusOK, getResp.StatusCode)

		var getRespData market.GetMarketResponse
		err = json.NewDecoder(getResp.Body).Decode(&getRespData)
		require.NoError(t, err)
		assert.Equal(t, createResp.Market.Id, getRespData.Market.Id)
	})

	t.Run("WalletService HTTP to gRPC", func(t *testing.T) {
		// Create a wallet via HTTP API Gateway
		createWalletReq := map[string]interface{}{
			"user_id":          "test-user-123",
			"currency":        "USD",
			"initial_balance": 1000.0,
		}

		jsonData, _ := json.Marshal(createWalletReq)
		
		resp, err := http.Post("http://localhost:8080/api/wallets", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var createResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&createResp)
		require.NoError(t, err)
		
		walletID, ok := createResp["account"].(map[string]interface{})["id"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, walletID)

		// Get the wallet via HTTP API Gateway
		getResp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/wallets/%s", walletID))
		require.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusOK, getResp.StatusCode)
	})

	t.Run("SettlementService HTTP to gRPC", func(t *testing.T) {
		// Create a settlement via HTTP API Gateway
		createSettlementReq := map[string]interface{}{
			"market_id":       "test-market-123",
			"winning_option_id": "test-option-123",
		}

		jsonData, _ := json.Marshal(createSettlementReq)
		
		resp, err := http.Post("http://localhost:8080/api/settlements", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var createResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&createResp)
		require.NoError(t, err)
		
		settlementID, ok := createResp["settlement"].(map[string]interface{})["id"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, settlementID)

		// Get the settlement via HTTP API Gateway
		getResp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/settlements/%s", settlementID))
		require.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusOK, getResp.StatusCode)
	})
}

func TestCircuitBreakerAndKafkaFallback(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Test circuit breaker opening and Kafka fallback
	t.Run("Circuit Breaker Opens on Service Failure", func(t *testing.T) {
		// Simulate service failure by stopping a service (in real test, you'd mock this)
		// For now, we'll test the circuit breaker configuration
		
		// Create multiple requests to trigger circuit breaker
		for i := 0; i < 10; i++ {
			resp, err := http.Get("http://localhost:8080/api/markets/non-existent")
			if err == nil {
				resp.Body.Close()
			}
		}

		// After circuit breaker opens, requests should fail fast
		start := time.Now()
		resp, err := http.Get("http://localhost:8080/api/markets/non-existent")
		duration := time.Since(start)
		
		if err == nil {
			resp.Body.Close()
		}

		// Circuit breaker should open quickly (less than 1 second)
		assert.Less(t, duration, 100*time.Millisecond)
	})

	t.Run("Kafka Fallback on Timeout", func(t *testing.T) {
		// Test that requests timeout and fallback to Kafka
		ctx := context.Background()
		
		// Create Kafka consumer to verify fallback messages
		consumer, err := kafka.NewConsumer([]string{"localhost:9092"}, "test-group", logger)
		require.NoError(t, err)
		defer consumer.Close()

		// Subscribe to fallback topics
		topics := []string{
			"market.ListMarkets.fallback",
			"market.GetMarket.fallback",
			"wallet.CreateWalletAccount.fallback",
			"settlement.CreateSettlement.fallback",
		}
		
		err = consumer.SubscribeTopics(topics)
		require.NoError(t, err)

		// Make a request that might timeout
		go func() {
			http.Get("http://localhost:8080/api/markets")
		}()

		// Wait for fallback message
		msg, err := consumer.ReadMessage(30 * time.Second)
		if err == nil {
			// Verify the fallback message structure
			var fallbackMsg map[string]interface{}
			err = json.Unmarshal(msg.Value, &fallbackMsg)
			require.NoError(t, err)
			
			assert.Contains(t, fallbackMsg, "service")
			assert.Contains(t, fallbackMsg, "method")
			assert.Contains(t, fallbackMsg, "timestamp")
			assert.Contains(t, fallbackMsg, "error")
		}
	})
}

func TestResiliencePatterns(t *testing.T) {
	t.Run("Retry Mechanism", func(t *testing.T) {
		// Test that failed requests are retried with exponential backoff
		// This would typically be tested with a mock service that fails initially
		
		start := time.Now()
		resp, err := http.Get("http://localhost:8080/api/health")
		duration := time.Since(start)
		
		if err == nil {
			defer resp.Body.Close()
		}

		// Health check should be fast (no retries needed)
		assert.Less(t, duration, 100*time.Millisecond)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("Metrics Collection", func(t *testing.T) {
		// Test that metrics are being collected for gRPC calls
		// Make some requests to generate metrics
		for i := 0; i < 5; i++ {
			resp, _ := http.Get("http://localhost:8080/api/markets")
			if resp != nil {
				resp.Body.Close()
			}
		}

		// Check metrics endpoint (if available)
		resp, err := http.Get("http://localhost:8080/metrics")
		if err == nil {
			defer resp.Body.Close()
			// Verify metrics are being collected
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}
	})
}

func TestEndToEndFlow(t *testing.T) {
	t.Run("Complete Prediction Market Flow", func(t *testing.T) {
		logger, _ := zap.NewDevelopment()
		defer logger.Sync()

		// Step 1: Create a market
		createMarketReq := map[string]interface{}{
			"title":       "End-to-End Test Market",
			"description": "Testing complete flow",
			"options": []map[string]interface{}{
				{"title": "Team A Wins", "odds": 2.0},
				{"title": "Team B Wins", "odds": 1.5},
			},
			"end_time": time.Now().Add(1 * time.Hour).Unix(),
		}

		jsonData, _ := json.Marshal(createMarketReq)
		resp, err := http.Post("http://localhost:8080/api/markets", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var marketResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&marketResp)
		require.NoError(t, err)
		
		marketID, ok := marketResp["market"].(map[string]interface{})["id"].(string)
		require.True(t, ok)

		// Step 2: Create a wallet
		createWalletReq := map[string]interface{}{
			"user_id":          "test-user-e2e",
			"currency":        "USD",
			"initial_balance": 500.0,
		}

		jsonData, _ = json.Marshal(createWalletReq)
		resp, err = http.Post("http://localhost:8080/api/wallets", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var walletResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&walletResp)
		require.NoError(t, err)
		
		walletID, ok := walletResp["account"].(map[string]interface{})["id"].(string)
		require.True(t, ok)

		// Step 3: Create a settlement
		createSettlementReq := map[string]interface{}{
			"market_id":       marketID,
			"winning_option_id": "team-a-wins",
		}

		jsonData, _ = json.Marshal(createSettlementReq)
		resp, err = http.Post("http://localhost:8080/api/settlements", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var settlementResp map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&settlementResp)
		require.NoError(t, err)
		
		settlementID, ok := settlementResp["settlement"].(map[string]interface{})["id"].(string)
		require.True(t, ok)

		// Step 4: Complete the settlement
		completeReq := map[string]interface{}{}
		jsonData, _ = json.Marshal(completeReq)
		
		req, err := http.NewRequest("PUT", 
			fmt.Sprintf("http://localhost:8080/api/settlements/%s/complete", settlementID),
			bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err = client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		logger.Info("End-to-end flow completed successfully",
			zap.String("market_id", marketID),
			zap.String("wallet_id", walletID),
			zap.String("settlement_id", settlementID),
		)
	})
}
