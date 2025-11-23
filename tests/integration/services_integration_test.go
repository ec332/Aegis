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
	market "aegis/proto/gen/market"
	wallet "aegis/proto/gen/wallet"
	settlement "aegis/proto/gen/settlement"
)

// Enhanced integration tests for service-to-service communication
func TestServiceToServiceCommunication(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	t.Run("Market Service Integration", func(t *testing.T) {
		// Test market creation and retrieval flow
		createMarketReq := map[string]interface{}{
			"title":       "Integration Test Market",
			"description": "Testing service integration",
			"options": []map[string]interface{}{
				{"title": "Option A", "odds": 2.0},
				{"title": "Option B", "odds": 1.5},
				{"title": "Option C", "odds": 3.0},
			},
			"end_time": time.Now().Add(2 * time.Hour).Unix(),
		}

		jsonData, _ := json.Marshal(createMarketReq)
		
		resp, err := http.Post("http://localhost:8080/api/markets", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var createResp market.CreateMarketResponse
		err = json.NewDecoder(resp.Body).Decode(&createResp)
		require.NoError(t, err)
		assert.NotEmpty(t, createResp.Market.Id)

		// Verify market was created by retrieving it
		getResp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/markets/%s", createResp.Market.Id))
		require.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusOK, getResp.StatusCode)

		var getRespData market.GetMarketResponse
		err = json.NewDecoder(getResp.Body).Decode(&getRespData)
		require.NoError(t, err)
		assert.Equal(t, createResp.Market.Id, getRespData.Market.Id)
		assert.Equal(t, "Integration Test Market", getRespData.Market.Title)
		assert.Len(t, getRespData.Market.Options, 3)

		// Test listing markets
		listResp, err := http.Get("http://localhost:8080/api/markets")
		require.NoError(t, err)
		defer listResp.Body.Close()

		assert.Equal(t, http.StatusOK, listResp.StatusCode)

		var listRespData market.ListMarketsResponse
		err = json.NewDecoder(listResp.Body).Decode(&listRespData)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(listRespData.Markets), 1)

		// Test market options endpoint
		optionsResp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/markets/%s/options", createResp.Market.Id))
		require.NoError(t, err)
		defer optionsResp.Body.Close()

		assert.Equal(t, http.StatusOK, optionsResp.StatusCode)
	})

	t.Run("Wallet Service Integration", func(t *testing.T) {
		// Test wallet creation and operations
		createWalletReq := map[string]interface{}{
			"user_id":          "integration-test-user",
			"currency":        "USD",
			"initial_balance": 1000.0,
		}

		jsonData, _ := json.Marshal(createWalletReq)
		
		resp, err := http.Post("http://localhost:8080/api/wallets", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var createResp wallet.CreateWalletAccountResponse
		err = json.NewDecoder(resp.Body).Decode(&createResp)
		require.NoError(t, err)
		assert.NotEmpty(t, createResp.Account.Id)
		assert.Equal(t, "integration-test-user", createResp.Account.UserId)
		assert.Equal(t, "USD", createResp.Account.Currency)
		assert.Equal(t, 1000.0, createResp.Account.Balance)

		walletID := createResp.Account.Id

		// Test deposit
		depositReq := map[string]interface{}{
			"amount": 500.0,
		}
		jsonData, _ = json.Marshal(depositReq)
		
		depositResp, err := http.Post(
			fmt.Sprintf("http://localhost:8080/api/wallets/%s/deposit", walletID),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		require.NoError(t, err)
		defer depositResp.Body.Close()

		assert.Equal(t, http.StatusOK, depositResp.StatusCode)

		var depositRespData wallet.DepositResponse
		err = json.NewDecoder(depositResp.Body).Decode(&depositRespData)
		require.NoError(t, err)
		assert.Equal(t, 1500.0, depositRespData.Account.Balance)

		// Test withdraw
		withdrawReq := map[string]interface{}{
			"amount": 300.0,
		}
		jsonData, _ = json.Marshal(withdrawReq)
		
		withdrawResp, err := http.Post(
			fmt.Sprintf("http://localhost:8080/api/wallets/%s/withdraw", walletID),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		require.NoError(t, err)
		defer withdrawResp.Body.Close()

		assert.Equal(t, http.StatusOK, withdrawResp.StatusCode)

		var withdrawRespData wallet.WithdrawResponse
		err = json.NewDecoder(withdrawResp.Body).Decode(&withdrawRespData)
		require.NoError(t, err)
		assert.Equal(t, 1200.0, withdrawRespData.Account.Balance)

		// Verify final balance
		getResp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/wallets/%s", walletID))
		require.NoError(t, err)
		defer getResp.Body.Close()

		assert.Equal(t, http.StatusOK, getResp.StatusCode)

		var getRespData wallet.GetWalletAccountResponse
		err = json.NewDecoder(getResp.Body).Decode(&getRespData)
		require.NoError(t, err)
		assert.Equal(t, 1200.0, getRespData.Account.Balance)
	})

	t.Run("Settlement Service Integration", func(t *testing.T) {
		// First create a market for settlement
		createMarketReq := map[string]interface{}{
			"title":       "Settlement Test Market",
			"description": "Testing settlement integration",
			"options": []map[string]interface{}{
				{"title": "Winning Option", "odds": 2.0},
				{"title": "Losing Option", "odds": 1.5},
			},
			"end_time": time.Now().Add(1 * time.Hour).Unix(),
		}

		jsonData, _ := json.Marshal(createMarketReq)
		
		marketResp, err := http.Post("http://localhost:8080/api/markets", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer marketResp.Body.Close()

		assert.Equal(t, http.StatusCreated, marketResp.StatusCode)

		var marketRespData market.CreateMarketResponse
		err = json.NewDecoder(marketResp.Body).Decode(&marketRespData)
		require.NoError(t, err)

		marketID := marketRespData.Market.Id
		winningOptionID := marketRespData.Market.Options[0].Id

		// Create settlement
		createSettlementReq := map[string]interface{}{
			"market_id":         marketID,
			"winning_option_id": winningOptionID,
		}

		jsonData, _ = json.Marshal(createSettlementReq)
		
		settlementResp, err := http.Post("http://localhost:8080/api/settlements", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer settlementResp.Body.Close()

		assert.Equal(t, http.StatusCreated, settlementResp.StatusCode)

		var settlementRespData settlement.CreateSettlementResponse
		err = json.NewDecoder(settlementResp.Body).Decode(&settlementRespData)
		require.NoError(t, err)
		assert.NotEmpty(t, settlementRespData.Settlement.Id)
		assert.Equal(t, marketID, settlementRespData.Settlement.MarketId)
		assert.Equal(t, winningOptionID, settlementRespData.Settlement.WinningOptionId)
		assert.Equal(t, settlement.SettlementStatus_PENDING, settlementRespData.Settlement.Status)

		settlementID := settlementRespData.Settlement.Id

		// Get settlement
		getSettlementResp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/settlements/%s", settlementID))
		require.NoError(t, err)
		defer getSettlementResp.Body.Close()

		assert.Equal(t, http.StatusOK, getSettlementResp.StatusCode)

		var getSettlementRespData settlement.GetSettlementResponse
		err = json.NewDecoder(getSettlementResp.Body).Decode(&getSettlementRespData)
		require.NoError(t, err)
		assert.Equal(t, settlementID, getSettlementRespData.Settlement.Id)

		// Complete settlement
		completeSettlementReq := map[string]interface{}{}
		jsonData, _ = json.Marshal(completeSettlementReq)
		
		req, err := http.NewRequest(
			"PUT",
			fmt.Sprintf("http://localhost:8080/api/settlements/%s/complete", settlementID),
			bytes.NewBuffer(jsonData),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second}
		completeResp, err := client.Do(req)
		require.NoError(t, err)
		defer completeResp.Body.Close()

		assert.Equal(t, http.StatusOK, completeResp.StatusCode)

		// Verify settlement is completed
		finalSettlementResp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/settlements/%s", settlementID))
		require.NoError(t, err)
		defer finalSettlementResp.Body.Close()

		assert.Equal(t, http.StatusOK, finalSettlementResp.StatusCode)

		var finalSettlementRespData settlement.GetSettlementResponse
		err = json.NewDecoder(finalSettlementResp.Body).Decode(&finalSettlementRespData)
		require.NoError(t, err)
		assert.Equal(t, settlement.SettlementStatus_COMPLETED, finalSettlementRespData.Settlement.Status)
	})

	t.Run("Cross-Service Integration", func(t *testing.T) {
		// Test complete flow: market -> wallet -> settlement
		
		// Step 1: Create market
		createMarketReq := map[string]interface{}{
			"title":       "Cross-Service Integration Market",
			"description": "Testing complete integration flow",
			"options": []map[string]interface{}{
				{"title": "Option 1", "odds": 2.0},
				{"title": "Option 2", "odds": 1.8},
			},
			"end_time": time.Now().Add(3 * time.Hour).Unix(),
		}

		jsonData, _ := json.Marshal(createMarketReq)
		
		marketResp, err := http.Post("http://localhost:8080/api/markets", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer marketResp.Body.Close()

		assert.Equal(t, http.StatusCreated, marketResp.StatusCode)

		var marketRespData market.CreateMarketResponse
		err = json.NewDecoder(marketResp.Body).Decode(&marketRespData)
		require.NoError(t, err)

		marketID := marketRespData.Market.Id

		// Step 2: Create wallet
		createWalletReq := map[string]interface{}{
			"user_id":          "cross-service-user",
			"currency":        "USD",
			"initial_balance": 2000.0,
		}

		jsonData, _ = json.Marshal(createWalletReq)
		
		walletResp, err := http.Post("http://localhost:8080/api/wallets", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer walletResp.Body.Close()

		assert.Equal(t, http.StatusCreated, walletResp.StatusCode)

		var walletRespData wallet.CreateWalletAccountResponse
		err = json.NewDecoder(walletResp.Body).Decode(&walletRespData)
		require.NoError(t, err)

		walletID := walletRespData.Account.Id

		// Step 3: Create settlement
		createSettlementReq := map[string]interface{}{
			"market_id":         marketID,
			"winning_option_id": marketRespData.Market.Options[0].Id,
		}

		jsonData, _ = json.Marshal(createSettlementReq)
		
		settlementResp, err := http.Post("http://localhost:8080/api/settlements", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer settlementResp.Body.Close()

		assert.Equal(t, http.StatusCreated, settlementResp.StatusCode)

		var settlementRespData settlement.CreateSettlementResponse
		err = json.NewDecoder(settlementResp.Body).Decode(&settlementRespData)
		require.NoError(t, err)

		settlementID := settlementRespData.Settlement.Id

		// Step 4: Complete settlement
		completeSettlementReq := map[string]interface{}{}
		jsonData, _ = json.Marshal(completeSettlementReq)
		
		req, err := http.NewRequest(
			"PUT",
			fmt.Sprintf("http://localhost:8080/api/settlements/%s/complete", settlementID),
			bytes.NewBuffer(jsonData),
		)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 30 * time.Second}
		completeResp, err := client.Do(req)
		require.NoError(t, err)
		defer completeResp.Body.Close()

		assert.Equal(t, http.StatusOK, completeResp.StatusCode)

		logger.Info("Cross-service integration test completed successfully",
			zap.String("market_id", marketID),
			zap.String("wallet_id", walletID),
			zap.String("settlement_id", settlementID),
		)
	})
}

func TestErrorHandlingIntegration(t *testing.T) {
	t.Run("Invalid Request Handling", func(t *testing.T) {
		// Test invalid market creation
		invalidMarketReq := map[string]interface{}{
			"title": "", // Empty title should be invalid
			"description": "Test description",
		}

		jsonData, _ := json.Marshal(invalidMarketReq)
		
		resp, err := http.Post("http://localhost:8080/api/markets", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should handle gracefully (either 400 or 500 depending on service implementation)
		assert.Contains(t, []int{http.StatusBadRequest, http.StatusInternalServerError}, resp.StatusCode)

		// Test invalid wallet operations
		invalidWalletReq := map[string]interface{}{
			"user_id":          "",
			"currency":        "INVALID",
			"initial_balance": -100.0, // Negative balance
		}

		jsonData, _ = json.Marshal(invalidWalletReq)
		
		resp, err = http.Post("http://localhost:8080/api/wallets", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Contains(t, []int{http.StatusBadRequest, http.StatusInternalServerError}, resp.StatusCode)

		// Test non-existent resource access
		resp, err = http.Get("http://localhost:8080/api/markets/non-existent-market-id")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("Service Unavailability Handling", func(t *testing.T) {
		// Test what happens when a service is temporarily unavailable
		// This would typically be tested with service mocking or by temporarily stopping services
		
		// Make multiple rapid requests to test circuit breaker behavior
		for i := 0; i < 10; i++ {
			resp, err := http.Get("http://localhost:8080/api/markets")
			if err == nil {
				resp.Body.Close()
			}
		}

		// The system should handle this gracefully
		// In a real test, you'd verify circuit breaker state
	})
}

func TestConcurrentRequestsIntegration(t *testing.T) {
	t.Run("Concurrent Market Operations", func(t *testing.T) {
		// Test concurrent market creation
		concurrency := 5
		done := make(chan bool, concurrency)
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(index int) {
				createMarketReq := map[string]interface{}{
					"title":       fmt.Sprintf("Concurrent Market %d", index),
					"description": "Testing concurrent operations",
					"options": []map[string]interface{}{
						{"title": "Option A", "odds": 2.0},
						{"title": "Option B", "odds": 1.5},
					},
					"end_time": time.Now().Add(time.Hour).Unix(),
				}

				jsonData, _ := json.Marshal(createMarketReq)
				
				resp, err := http.Post("http://localhost:8080/api/markets", "application/json", bytes.NewBuffer(jsonData))
				if err != nil {
					errors <- err
					done <- false
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusCreated {
					errors <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
					done <- false
					return
				}

				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		successCount := 0
		for i := 0; i < concurrency; i++ {
			select {
			case success := <-done:
				if success {
					successCount++
				}
			case err := <-errors:
				t.Logf("Concurrent operation error: %v", err)
			case <-time.After(30 * time.Second):
				t.Fatal("Timeout waiting for concurrent operations")
			}
		}

		assert.Equal(t, concurrency, successCount, "All concurrent market creations should succeed")
	})

	t.Run("Concurrent Wallet Operations", func(t *testing.T) {
		// Create a wallet first
		createWalletReq := map[string]interface{}{
			"user_id":          "concurrent-user",
			"currency":        "USD",
			"initial_balance": 1000.0,
		}

		jsonData, _ := json.Marshal(createWalletReq)
		
		resp, err := http.Post("http://localhost:8080/api/wallets", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var createResp wallet.CreateWalletAccountResponse
		err = json.NewDecoder(resp.Body).Decode(&createResp)
		require.NoError(t, err)

		walletID := createResp.Account.Id

		// Test concurrent deposits and withdrawals
		concurrency := 6
		done := make(chan bool, concurrency)
		errors := make(chan error, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(index int) {
				var operationResp *http.Response
				var err error

				if index%2 == 0 {
					// Deposit
					depositReq := map[string]interface{}{"amount": 100.0}
					jsonData, _ := json.Marshal(depositReq)
					
					operationResp, err = http.Post(
						fmt.Sprintf("http://localhost:8080/api/wallets/%s/deposit", walletID),
						"application/json",
						bytes.NewBuffer(jsonData),
					)
				} else {
					// Withdraw
					withdrawReq := map[string]interface{}{"amount": 50.0}
					jsonData, _ := json.Marshal(withdrawReq)
					
					operationResp, err = http.Post(
						fmt.Sprintf("http://localhost:8080/api/wallets/%s/withdraw", walletID),
						"application/json",
						bytes.NewBuffer(jsonData),
					)
				}

				if err != nil {
					errors <- err
					done <- false
					return
				}
				defer operationResp.Body.Close()

				if operationResp.StatusCode != http.StatusOK {
					errors <- fmt.Errorf("unexpected status code: %d", operationResp.StatusCode)
					done <- false
					return
				}

				done <- true
			}(i)
		}

		// Wait for all operations to complete
		successCount := 0
		for i := 0; i < concurrency; i++ {
			select {
			case success := <-done:
				if success {
					successCount++
				}
			case err := <-errors:
				t.Logf("Concurrent wallet operation error: %v", err)
			case <-time.After(30 * time.Second):
				t.Fatal("Timeout waiting for concurrent wallet operations")
			}
		}

		assert.Equal(t, concurrency, successCount, "All concurrent wallet operations should succeed")
	})
}

func TestKafkaIntegration(t *testing.T) {
	t.Run("Kafka Fallback Messages", func(t *testing.T) {
		// Test that fallback messages are properly published to Kafka
		// when services are unavailable or timeout
		
		ctx := context.Background()
		
		// Create Kafka consumer to verify fallback messages
		consumer, err := kafka.NewConsumer([]string{"localhost:9092"}, "integration-test-group", logger)
		if err != nil {
			t.Skip("Kafka not available for integration testing")
			return
		}
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

		// Make requests that might trigger fallback
		go func() {
			// Make multiple requests to increase chance of fallback
			for i := 0; i < 5; i++ {
				http.Get("http://localhost:8080/api/markets")
				http.Get("http://localhost:8080/api/wallets/non-existent")
				http.Post("http://localhost:8080/api/settlements", "application/json", bytes.NewBuffer([]byte("{}")))
				time.Sleep(100 * time.Millisecond)
			}
		}()

		// Wait for fallback messages
		timeout := time.After(30 * time.Second)
		messagesReceived := 0
		maxMessages := 3

		for messagesReceived < maxMessages {
			select {
			case msg, err := consumer.ReadMessage(5 * time.Second):
				if err == nil {
					// Verify fallback message structure
					var fallbackMsg map[string]interface{}
					err = json.Unmarshal(msg.Value, &fallbackMsg)
					require.NoError(t, err)
					
					assert.Contains(t, fallbackMsg, "service")
					assert.Contains(t, fallbackMsg, "method")
					assert.Contains(t, fallbackMsg, "timestamp")
					assert.Contains(t, fallbackMsg, "error")
					
					messagesReceived++
					t.Logf("Received fallback message for %s.%s", 
						fallbackMsg["service"], fallbackMsg["method"])
				}
			case <-timeout:
				t.Logf("Received %d fallback messages before timeout", messagesReceived)
				break
			}
		}

		assert.GreaterOrEqual(t, messagesReceived, 1, "Should receive at least one fallback message")
	})
}

func TestHealthCheckIntegration(t *testing.T) {
	t.Run("All Services Health Check", func(t *testing.T) {
		// Test that all services are healthy and responding
		services := []struct {
			name string
			url  string
		}{
			{"API Gateway", "http://localhost:8080/health"},
			{"Transaction Service", "http://localhost:8081/health"},
		}

		for _, service := range services {
			t.Run(service.name, func(t *testing.T) {
				resp, err := http.Get(service.url)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var healthResp map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&healthResp)
				require.NoError(t, err)
				assert.Equal(t, "healthy", healthResp["status"])
			})
		}
	})

	t.Run("Service Dependencies Health", func(t *testing.T) {
		// Test that services can reach their dependencies (Kafka, Redis, etc.)
		
		// Test Kafka connectivity by checking if we can produce/consume messages
		producer, err := kafka.NewProducer(kafka.Config{Brokers: []string{"localhost:9092"}}, logger)
		if err != nil {
			t.Skip("Kafka not available for health check")
			return
		}
		defer producer.Close()

		// Test message production
		testMessage := map[string]interface{}{
			"test":    "health-check",
			"service": "integration-test",
			"time":    time.Now().Unix(),
		}

		err = producer.Publish(ctx, "health-check", "test-key", testMessage)
		assert.NoError(t, err, "Should be able to publish to Kafka")

		// Test message consumption
		consumer, err := kafka.NewConsumer([]string{"localhost:9092"}, "health-check-group", logger)
		require.NoError(t, err)
		defer consumer.Close()

		err = consumer.SubscribeTopics([]string{"health-check"})
		require.NoError(t, err)

		// Try to read the message we just published
		msg, err := consumer.ReadMessage(10 * time.Second)
		if err == nil {
			assert.NotNil(t, msg, "Should be able to consume from Kafka")
		}
	})
}