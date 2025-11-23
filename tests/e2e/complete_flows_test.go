package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// End-to-End Test Suite for Complete Business Flows
func TestCompletePredictionMarketFlow(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	t.Run("Complete Market Lifecycle", func(t *testing.T) {
		// Step 1: Create a market
		logger.Info("Creating prediction market...")
		
		createMarketReq := map[string]interface{}{
			"title":       "E2E Test: Sports Match Outcome",
			"description": "End-to-end test for complete market lifecycle",
			"options": []map[string]interface{}{
				{"title": "Team A Wins", "odds": 2.5},
				{"title": "Team B Wins", "odds": 1.8},
				{"title": "Draw", "odds": 3.2},
			},
			"end_time": time.Now().Add(2 * time.Hour).Unix(),
		}

		jsonData, _ := json.Marshal(createMarketReq)
		
		marketResp, err := http.Post("http://localhost:8080/api/markets", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer marketResp.Body.Close()

		assert.Equal(t, http.StatusCreated, marketResp.StatusCode)

		var marketRespData map[string]interface{}
		err = json.NewDecoder(marketResp.Body).Decode(&marketRespData)
		require.NoError(t, err)
		
		marketID := marketRespData["market"].(map[string]interface{})["id"].(string)
		logger.Info("Market created", zap.String("market_id", marketID))

		// Step 2: Create multiple user wallets
		logger.Info("Creating user wallets...")
		
		users := []struct {
			userID   string
			currency string
			balance  float64
		}{
			{"user-e2e-1", "USD", 1000.0},
			{"user-e2e-2", "USD", 1500.0},
			{"user-e2e-3", "USD", 800.0},
		}

		walletIDs := make([]string, len(users))
		for i, user := range users {
			createWalletReq := map[string]interface{}{
				"user_id":          user.userID,
				"currency":        user.currency,
				"initial_balance": user.balance,
			}

			jsonData, _ := json.Marshal(createWalletReq)
			
			walletResp, err := http.Post("http://localhost:8080/api/wallets", "application/json", bytes.NewBuffer(jsonData))
			require.NoError(t, err)
			defer walletResp.Body.Close()

			assert.Equal(t, http.StatusCreated, walletResp.StatusCode)

			var walletRespData map[string]interface{}
			err = json.NewDecoder(walletResp.Body).Decode(&walletRespData)
			require.NoError(t, err)
			
			walletIDs[i] = walletRespData["account"].(map[string]interface{})["id"].(string)
			logger.Info("Wallet created", zap.String("user_id", user.userID), zap.String("wallet_id", walletIDs[i]))
		}

		// Step 3: Simulate betting activity (deposit/withdraw operations)
		logger.Info("Simulating betting activity...")
		
		// User 1 deposits more funds
		depositReq := map[string]interface{}{"amount": 500.0}
		jsonData, _ = json.Marshal(depositReq)
		
		depositResp, err := http.Post(
			fmt.Sprintf("http://localhost:8080/api/wallets/%s/deposit", walletIDs[0]),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		require.NoError(t, err)
		defer depositResp.Body.Close()
		assert.Equal(t, http.StatusOK, depositResp.StatusCode)

		// User 2 withdraws some funds
		withdrawReq := map[string]interface{}{"amount": 200.0}
		jsonData, _ = json.Marshal(withdrawReq)
		
		withdrawResp, err := http.Post(
			fmt.Sprintf("http://localhost:8080/api/wallets/%s/withdraw", walletIDs[1]),
			"application/json",
			bytes.NewBuffer(jsonData),
		)
		require.NoError(t, err)
		defer withdrawResp.Body.Close()
		assert.Equal(t, http.StatusOK, withdrawResp.StatusCode)

		// Step 4: Create settlement for the market
		logger.Info("Creating market settlement...")
		
		createSettlementReq := map[string]interface{}{
			"market_id":         marketID,
			"winning_option_id": "team-a-wins", // Simulate Team A winning
		}

		jsonData, _ = json.Marshal(createSettlementReq)
		
		settlementResp, err := http.Post("http://localhost:8080/api/settlements", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer settlementResp.Body.Close()

		assert.Equal(t, http.StatusCreated, settlementResp.StatusCode)

		var settlementRespData map[string]interface{}
		err = json.NewDecoder(settlementResp.Body).Decode(&settlementRespData)
		require.NoError(t, err)
		
		settlementID := settlementRespData["settlement"].(map[string]interface{})["id"].(string)
		logger.Info("Settlement created", zap.String("settlement_id", settlementID))

		// Step 5: Complete the settlement
		logger.Info("Completing settlement...")
		
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
		logger.Info("Settlement completed successfully")

		// Step 6: Verify final state
		logger.Info("Verifying final state...")
		
		// Check market status
		finalMarketResp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/markets/%s", marketID))
		require.NoError(t, err)
		defer finalMarketResp.Body.Close()
		assert.Equal(t, http.StatusOK, finalMarketResp.StatusCode)

		// Check settlement status
		finalSettlementResp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/settlements/%s", settlementID))
		require.NoError(t, err)
		defer finalSettlementResp.Body.Close()
		assert.Equal(t, http.StatusOK, finalSettlementResp.StatusCode)

		// Check wallet balances
		for i, walletID := range walletIDs {
			walletResp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/wallets/%s", walletID))
			require.NoError(t, err)
			defer walletResp.Body.Close()
			assert.Equal(t, http.StatusOK, walletResp.StatusCode)

			var walletData map[string]interface{}
			err = json.NewDecoder(walletResp.Body).Decode(&walletData)
			require.NoError(t, err)
			
			finalBalance := walletData["account"].(map[string]interface{})["balance"].(float64)
			logger.Info("Final wallet balance", 
				zap.String("user_id", users[i].userID),
				zap.Float64("initial_balance", users[i].balance),
				zap.Float64("final_balance", finalBalance))
		}

		logger.Info("Complete prediction market flow test passed successfully")
	})
}

func TestConcurrentMarketOperations(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	t.Run("Multiple Markets Creation", func(t *testing.T) {
		concurrency := 10
		done := make(chan struct {
			marketID string
			err      error
		}, concurrency)

		// Create multiple markets concurrently
		for i := 0; i < concurrency; i++ {
			go func(index int) {
				createMarketReq := map[string]interface{}{
					"title":       fmt.Sprintf("Concurrent Market %d", index),
					"description": fmt.Sprintf("Testing concurrent market creation %d", index),
					"options": []map[string]interface{}{
						{"title": "Option A", "odds": 2.0},
						{"title": "Option B", "odds": 1.8},
					},
					"end_time": time.Now().Add(time.Duration(index+1) * time.Hour).Unix(),
				}

				jsonData, _ := json.Marshal(createMarketReq)
				
				resp, err := http.Post("http://localhost:8080/api/markets", "application/json", bytes.NewBuffer(jsonData))
				if err != nil {
					done <- struct {
						marketID string
						err      error
					}{"", err}
					return
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusCreated {
					done <- struct {
						marketID string
						err      error
					}{"", fmt.Errorf("unexpected status code: %d", resp.StatusCode)}
					return
				}

				var respData map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&respData)
				if err != nil {
					done <- struct {
						marketID string
						err      error
					}{"", err}
					return
				}

				marketID := respData["market"].(map[string]interface{})["id"].(string)
				done <- struct {
					marketID string
					err      error
				}{marketID, nil}
			}(i)
		}

		// Collect results
		marketIDs := make([]string, 0, concurrency)
		successCount := 0
		
		for i := 0; i < concurrency; i++ {
			result := <-done
			if result.err == nil {
				successCount++
				marketIDs = append(marketIDs, result.marketID)
				logger.Info("Concurrent market created", zap.String("market_id", result.marketID))
			} else {
				t.Logf("Concurrent market creation failed: %v", result.err)
			}
		}

		assert.Equal(t, concurrency, successCount, "All concurrent market creations should succeed")
		assert.Len(t, marketIDs, concurrency, "Should have created all markets")

		// Verify all markets can be retrieved
		for _, marketID := range marketIDs {
			resp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/markets/%s", marketID))
			require.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}

		logger.Info("Concurrent market creation test completed", zap.Int("created_count", successCount))
	})

	t.Run("Concurrent Wallet Operations", func(t *testing.T) {
		// Create a wallet first
		createWalletReq := map[string]interface{}{
			"user_id":          "concurrent-wallet-user",
			"currency":        "USD",
			"initial_balance": 10000.0, // High balance for multiple operations
		}

		jsonData, _ := json.Marshal(createWalletReq)
		
		resp, err := http.Post("http://localhost:8080/api/wallets", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var walletRespData map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&walletRespData)
		require.NoError(t, err)
		
		walletID := walletRespData["account"].(map[string]interface{})["id"].(string)

		// Perform concurrent operations
		concurrency := 20
		done := make(chan struct {
			operation string
			amount    float64
			err       error
		}, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(index int) {
				var operationResp *http.Response
				var err error
				var amount float64
				var operation string

				if index%2 == 0 {
					// Deposit
					amount = 100.0 + float64(index)
					operation = "deposit"
					
					depositReq := map[string]interface{}{"amount": amount}
					jsonData, _ := json.Marshal(depositReq)
					
					operationResp, err = http.Post(
						fmt.Sprintf("http://localhost:8080/api/wallets/%s/deposit", walletID),
						"application/json",
						bytes.NewBuffer(jsonData),
					)
				} else {
					// Withdraw
					amount = 50.0 + float64(index)
					operation = "withdraw"
					
					withdrawReq := map[string]interface{}{"amount": amount}
					jsonData, _ := json.Marshal(withdrawReq)
					
					operationResp, err = http.Post(
						fmt.Sprintf("http://localhost:8080/api/wallets/%s/withdraw", walletID),
						"application/json",
						bytes.NewBuffer(jsonData),
					)
				}

				if err != nil {
					done <- struct {
						operation string
						amount    float64
						err       error
					}{operation, amount, err}
					return
				}
				defer operationResp.Body.Close()

				if operationResp.StatusCode != http.StatusOK {
					done <- struct {
						operation string
						amount    float64
						err       error
					}{operation, amount, fmt.Errorf("unexpected status code: %d", operationResp.StatusCode)}
					return
				}

				done <- struct {
					operation string
					amount    float64
					err       error
				}{operation, amount, nil}
			}(i)
		}

		// Collect results
		successCount := 0
		totalDeposits := 0.0
		totalWithdrawals := 0.0
		
		for i := 0; i < concurrency; i++ {
			result := <-done
			if result.err == nil {
				successCount++
				if result.operation == "deposit" {
					totalDeposits += result.amount
				} else {
					totalWithdrawals += result.amount
				}
				logger.Info("Concurrent wallet operation completed", 
					zap.String("operation", result.operation),
					zap.Float64("amount", result.amount))
			} else {
				t.Logf("Concurrent wallet operation failed: %v", result.err)
			}
		}

		assert.Equal(t, concurrency, successCount, "All concurrent wallet operations should succeed")

		// Verify final balance
		finalWalletResp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/wallets/%s", walletID))
		require.NoError(t, err)
		defer finalWalletResp.Body.Close()

		assert.Equal(t, http.StatusOK, finalWalletResp.StatusCode)

		var finalWalletData map[string]interface{}
		err = json.NewDecoder(finalWalletResp.Body).Decode(&finalWalletData)
		require.NoError(t, err)
		
		finalBalance := finalWalletData["account"].(map[string]interface{})["balance"].(float64)
		expectedBalance := 10000.0 + totalDeposits - totalWithdrawals
		
		assert.InDelta(t, expectedBalance, finalBalance, 0.01, "Final balance should match expected balance")

		logger.Info("Concurrent wallet operations test completed", 
			zap.Int("operations", successCount),
			zap.Float64("total_deposits", totalDeposits),
			zap.Float64("total_withdrawals", totalWithdrawals),
			zap.Float64("final_balance", finalBalance))
	})
}

func TestErrorRecoveryAndResilience(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	t.Run("Service Recovery After Errors", func(t *testing.T) {
		// Test that the system can recover from various error conditions
		
		// Create a market
		createMarketReq := map[string]interface{}{
			"title":       "Error Recovery Test Market",
			"description": "Testing system resilience and error recovery",
			"options": []map[string]interface{}{
				{"title": "Option A", "odds": 2.0},
				{"title": "Option B", "odds": 1.5},
			},
			"end_time": time.Now().Add(1 * time.Hour).Unix(),
		}

		jsonData, _ := json.Marshal(createMarketReq)
		
		resp, err := http.Post("http://localhost:8080/api/markets", "application/json", bytes.NewBuffer(jsonData))
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var marketRespData map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&marketRespData)
		require.NoError(t, err)
		
		marketID := marketRespData["market"].(map[string]interface{})["id"].(string)

		// Test invalid operations that should fail gracefully
		invalidOperations := []struct {
			name   string
			method string
			url    string
			body   interface{}
		}{
			{
				name:   "Invalid market ID",
				method: "GET",
				url:    "http://localhost:8080/api/markets/invalid-market-id",
				body:   nil,
			},
			{
				name:   "Invalid wallet creation",
				method: "POST",
				url:    "http://localhost:8080/api/wallets",
				body:   map[string]interface{}{"user_id": "", "currency": "INVALID", "initial_balance": -100},
			},
			{
				name:   "Invalid settlement creation",
				method: "POST",
				url:    "http://localhost:8080/api/settlements",
				body:   map[string]interface{}{"market_id": "", "winning_option_id": ""},
			},
		}

		for _, op := range invalidOperations {
			t.Run(op.name, func(t *testing.T) {
				var resp *http.Response
				var err error

				if op.method == "GET" {
					resp, err = http.Get(op.url)
				} else {
					jsonData, _ := json.Marshal(op.body)
					resp, err = http.Post(op.url, "application/json", bytes.NewBuffer(jsonData))
				}

				require.NoError(t, err)
				defer resp.Body.Close()

				// Should return appropriate error status
				assert.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError}, resp.StatusCode)
			})
		}

		// Verify that valid operations still work after errors
		validMarketResp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/markets/%s", marketID))
		require.NoError(t, err)
		defer validMarketResp.Body.Close()
		assert.Equal(t, http.StatusOK, validMarketResp.StatusCode)

		logger.Info("Error recovery test completed successfully")
	})
}

func TestSystemHealthAndMonitoring(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	t.Run("Health Check Endpoints", func(t *testing.T) {
		healthEndpoints := []struct {
			name string
			url  string
		}{
			{"API Gateway", "http://localhost:8080/health"},
			{"Transaction Service", "http://localhost:8081/health"},
		}

		for _, endpoint := range healthEndpoints {
			t.Run(endpoint.name, func(t *testing.T) {
				resp, err := http.Get(endpoint.url)
				require.NoError(t, err)
				defer resp.Body.Close()

				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var healthData map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&healthData)
				require.NoError(t, err)
				assert.Equal(t, "healthy", healthData["status"])

				logger.Info("Health check passed", zap.String("service", endpoint.name))
			})
		}
	})

	t.Run("API Endpoints Availability", func(t *testing.T) {
		endpoints := []struct {
			name   string
			method string
			url    string
		}{
			{"List Markets", "GET", "http://localhost:8080/api/markets"},
			{"Create Market", "POST", "http://localhost:8080/api/markets"},
			{"List Wallets", "GET", "http://localhost:8080/api/wallets"},
			{"Create Wallet", "POST", "http://localhost:8080/api/wallets"},
			{"List Settlements", "GET", "http://localhost:8080/api/settlements"},
			{"Create Settlement", "POST", "http://localhost:8080/api/settlements"},
		}

		for _, endpoint := range endpoints {
			t.Run(endpoint.name, func(t *testing.T) {
				var resp *http.Response
				var err error

				if endpoint.method == "GET" {
					resp, err = http.Get(endpoint.url)
				} else {
					resp, err = http.Post(endpoint.url, "application/json", bytes.NewBuffer([]byte("{}")))
				}

				require.NoError(t, err)
				defer resp.Body.Close()

				// Should not return 503 Service Unavailable or connection errors
				assert.NotEqual(t, http.StatusServiceUnavailable, resp.StatusCode)
				assert.Greater(t, resp.StatusCode, 0)

				logger.Info("API endpoint available", zap.String("endpoint", endpoint.name))
			})
		}
	})
}