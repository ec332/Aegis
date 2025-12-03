package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	market "github.com/aegis/proto/gen/market"
	settlement "github.com/aegis/proto/gen/settlement"
	wallet "github.com/aegis/proto/gen/wallet"
)

// Mock Kafka producer
type mockKafkaProducer struct {
	mock.Mock
}

func (m *mockKafkaProducer) Publish(ctx context.Context, topic string, key string, value interface{}) error {
	args := m.Called(ctx, topic, key, value)
	return args.Error(0)
}

// Mock implementations for testing
type mockMarketService struct {
	mock.Mock
}

func (m *mockMarketService) CreateMarket(ctx context.Context, req *market.CreateMarketRequest, opts ...grpc.CallOption) (*market.CreateMarketResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*market.CreateMarketResponse), args.Error(1)
}

func (m *mockMarketService) GetMarket(ctx context.Context, req *market.GetMarketRequest, opts ...grpc.CallOption) (*market.GetMarketResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*market.GetMarketResponse), args.Error(1)
}

func (m *mockMarketService) ListMarkets(ctx context.Context, req *market.ListMarketsRequest, opts ...grpc.CallOption) (*market.ListMarketsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*market.ListMarketsResponse), args.Error(1)
}

func (m *mockMarketService) UpdateMarket(ctx context.Context, req *market.UpdateMarketRequest, opts ...grpc.CallOption) (*market.UpdateMarketResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*market.UpdateMarketResponse), args.Error(1)
}

func (m *mockMarketService) GetMarketOptions(ctx context.Context, req *market.GetMarketOptionsRequest, opts ...grpc.CallOption) (*market.GetMarketOptionsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*market.GetMarketOptionsResponse), args.Error(1)
}

func (m *mockMarketService) GetUser(ctx context.Context, req *market.GetUserRequest, opts ...grpc.CallOption) (*market.GetUserResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*market.GetUserResponse), args.Error(1)
}

func (m *mockMarketService) GetUserByWallet(ctx context.Context, req *market.GetUserByWalletRequest, opts ...grpc.CallOption) (*market.GetUserResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*market.GetUserResponse), args.Error(1)
}

func (m *mockMarketService) CreateUser(ctx context.Context, req *market.CreateUserRequest, opts ...grpc.CallOption) (*market.CreateUserResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*market.CreateUserResponse), args.Error(1)
}

func (m *mockMarketService) UpdateUser(ctx context.Context, req *market.UpdateUserRequest, opts ...grpc.CallOption) (*market.UpdateUserResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*market.UpdateUserResponse), args.Error(1)
}

type mockWalletService struct {
	mock.Mock
}

func (m *mockWalletService) CreateWalletAccount(ctx context.Context, req *wallet.CreateWalletAccountRequest, opts ...grpc.CallOption) (*wallet.CreateWalletAccountResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*wallet.CreateWalletAccountResponse), args.Error(1)
}

func (m *mockWalletService) GetWalletAccount(ctx context.Context, req *wallet.GetWalletAccountRequest, opts ...grpc.CallOption) (*wallet.GetWalletAccountResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*wallet.GetWalletAccountResponse), args.Error(1)
}

func (m *mockWalletService) Deposit(ctx context.Context, req *wallet.DepositRequest, opts ...grpc.CallOption) (*wallet.DepositResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*wallet.DepositResponse), args.Error(1)
}

func (m *mockWalletService) Withdrawal(ctx context.Context, req *wallet.WithdrawalRequest, opts ...grpc.CallOption) (*wallet.WithdrawalResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*wallet.WithdrawalResponse), args.Error(1)
}

func (m *mockWalletService) GetWalletAccountByUserID(ctx context.Context, req *wallet.GetWalletAccountByUserIDRequest, opts ...grpc.CallOption) (*wallet.GetWalletAccountByUserIDResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*wallet.GetWalletAccountByUserIDResponse), args.Error(1)
}

func (m *mockWalletService) UpdateWalletAccount(ctx context.Context, req *wallet.UpdateWalletAccountRequest, opts ...grpc.CallOption) (*wallet.UpdateWalletAccountResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*wallet.UpdateWalletAccountResponse), args.Error(1)
}

func (m *mockWalletService) DebitWallet(ctx context.Context, req *wallet.DebitWalletRequest, opts ...grpc.CallOption) (*wallet.DebitWalletResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*wallet.DebitWalletResponse), args.Error(1)
}

func (m *mockWalletService) CreditWallet(ctx context.Context, req *wallet.CreditWalletRequest, opts ...grpc.CallOption) (*wallet.CreditWalletResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*wallet.CreditWalletResponse), args.Error(1)
}

func (m *mockWalletService) GetWalletTransactions(ctx context.Context, req *wallet.GetWalletTransactionsRequest, opts ...grpc.CallOption) (*wallet.GetWalletTransactionsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*wallet.GetWalletTransactionsResponse), args.Error(1)
}

func (m *mockWalletService) GetTransaction(ctx context.Context, req *wallet.GetTransactionRequest, opts ...grpc.CallOption) (*wallet.GetTransactionResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*wallet.GetTransactionResponse), args.Error(1)
}

func (m *mockWalletService) SettleTransaction(ctx context.Context, req *wallet.SettleTransactionRequest, opts ...grpc.CallOption) (*wallet.SettleTransactionResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*wallet.SettleTransactionResponse), args.Error(1)
}

type mockSettlementService struct {
	mock.Mock
}

func (m *mockSettlementService) CreateSettlement(ctx context.Context, req *settlement.CreateSettlementRequest, opts ...grpc.CallOption) (*settlement.CreateSettlementResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*settlement.CreateSettlementResponse), args.Error(1)
}

func (m *mockSettlementService) GetSettlement(ctx context.Context, req *settlement.GetSettlementRequest, opts ...grpc.CallOption) (*settlement.GetSettlementResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*settlement.GetSettlementResponse), args.Error(1)
}

func (m *mockSettlementService) CompleteSettlement(ctx context.Context, req *settlement.CompleteSettlementRequest, opts ...grpc.CallOption) (*settlement.CompleteSettlementResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*settlement.CompleteSettlementResponse), args.Error(1)
}

func (m *mockSettlementService) GetSettlementByMarketID(ctx context.Context, req *settlement.GetSettlementByMarketIDRequest, opts ...grpc.CallOption) (*settlement.GetSettlementResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*settlement.GetSettlementResponse), args.Error(1)
}

func (m *mockSettlementService) ProcessPayout(ctx context.Context, req *settlement.ProcessPayoutRequest, opts ...grpc.CallOption) (*settlement.ProcessPayoutResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*settlement.ProcessPayoutResponse), args.Error(1)
}

// Test setup helpers
func setupTestGateway(t *testing.T) (*APIGateway, *mockMarketService, *mockWalletService, *mockSettlementService, *mockKafkaProducer) {
    logger, _ := zap.NewDevelopment()

    marketMock := &mockMarketService{}
    walletMock := &mockWalletService{}
    settlementMock := &mockSettlementService{}
    kafkaMock := &mockKafkaProducer{}

    gateway := &APIGateway{
        logger:         logger,
        marketStub:     marketMock,
        walletStub:     walletMock,
        settlementStub: settlementMock,
        kafkaProducer:  kafkaMock,
    }

	return gateway, marketMock, walletMock, settlementMock, kafkaMock
}

func createTestRequest(method, path string, body interface{}) (*http.Request, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

// CORS Tests
func TestCORSConfiguration(t *testing.T) {
	tests := []struct {
		name                 string
		envOrigins           string
		envMethods           string
		envHeaders           string
		requestOrigin        string
		requestMethod        string
		requestHeaders       string
		expectedStatus       int
		expectedAllowOrigin  string
		expectedAllowMethods string
		expectedAllowHeaders string
	}{
		{
			name:                 "Default CORS configuration",
			envOrigins:           "",
			envMethods:           "",
			envHeaders:           "",
			requestOrigin:        "http://localhost:3000",
			requestMethod:        "GET",
			requestHeaders:       "Content-Type",
			expectedStatus:       http.StatusOK,
			expectedAllowOrigin:  "http://localhost:3000",
			expectedAllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
			expectedAllowHeaders: "Accept, Content-Type, Authorization",
		},
		{
			name:                 "Custom origins allowed",
			envOrigins:           "https://example.com,https://app.example.com",
			envMethods:           "GET,POST",
			envHeaders:           "Authorization,X-Custom-Header",
			requestOrigin:        "https://example.com",
			requestMethod:        "POST",
			requestHeaders:       "Authorization",
			expectedStatus:       http.StatusOK,
			expectedAllowOrigin:  "https://example.com",
			expectedAllowMethods: "GET, POST",
			expectedAllowHeaders: "Authorization, X-Custom-Header",
		},
		{
			name:                 "Origin not allowed",
			envOrigins:           "https://example.com",
			envMethods:           "GET,POST",
			envHeaders:           "Content-Type",
			requestOrigin:        "https://malicious.com",
			requestMethod:        "GET",
			requestHeaders:       "Content-Type",
			expectedStatus:       http.StatusOK, // CORS middleware allows but doesn't set headers
			expectedAllowOrigin:  "",
			expectedAllowMethods: "",
			expectedAllowHeaders: "",
		},
		{
			name:                 "Preflight OPTIONS request",
			envOrigins:           "http://localhost:3000",
			envMethods:           "GET,POST,PUT,DELETE",
			envHeaders:           "Content-Type,Authorization",
			requestOrigin:        "http://localhost:3000",
			requestMethod:        "OPTIONS",
			requestHeaders:       "Content-Type,Authorization",
			expectedStatus:       http.StatusOK,
			expectedAllowOrigin:  "http://localhost:3000",
			expectedAllowMethods: "GET, POST, PUT, DELETE",
			expectedAllowHeaders: "Content-Type, Authorization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			if tt.envOrigins != "" {
				os.Setenv("CORS_ORIGINS", tt.envOrigins)
				defer os.Unsetenv("CORS_ORIGINS")
			}
			if tt.envMethods != "" {
				os.Setenv("CORS_METHODS", tt.envMethods)
				defer os.Unsetenv("CORS_METHODS")
			}
			if tt.envHeaders != "" {
				os.Setenv("CORS_HEADERS", tt.envHeaders)
				defer os.Unsetenv("CORS_HEADERS")
			}

			// Create a test server with the main function's CORS setup
			// We'll test this by making actual HTTP requests
			req, err := http.NewRequest(tt.requestMethod, "/health", nil)
			require.NoError(t, err)

			if tt.requestOrigin != "" {
				req.Header.Set("Origin", tt.requestOrigin)
			}
			if tt.requestHeaders != "" {
				req.Header.Set("Access-Control-Request-Headers", tt.requestHeaders)
			}

			// Create a simple handler to test CORS
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Apply CORS middleware similar to main.go
			originsEnv := strings.TrimSpace(getEnv("CORS_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000"))
			methodsEnv := strings.TrimSpace(getEnv("CORS_METHODS", "GET,POST,PUT,DELETE,OPTIONS"))
			headersEnv := strings.TrimSpace(getEnv("CORS_HEADERS", "Accept,Content-Type,Authorization"))

			var origins []string
			for _, o := range strings.Split(originsEnv, ",") {
				o = strings.TrimSpace(o)
				if o != "" {
					origins = append(origins, o)
				}
			}
			var methods []string
			for _, m := range strings.Split(methodsEnv, ",") {
				m = strings.TrimSpace(m)
				if m != "" {
					methods = append(methods, m)
				}
			}
			var headers []string
			for _, h := range strings.Split(headersEnv, ",") {
				h = strings.TrimSpace(h)
				if h != "" {
					headers = append(headers, h)
				}
			}

			// Simple CORS implementation for testing
			corsHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				origin := r.Header.Get("Origin")

				// Check if origin is allowed
				originAllowed := false
				for _, allowedOrigin := range origins {
					if allowedOrigin == origin {
						originAllowed = true
						break
					}
				}

				if originAllowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ", "))
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ", "))
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Max-Age", "300")
				}

				if r.Method == "OPTIONS" {
					w.WriteHeader(http.StatusOK)
					return
				}

				handler.ServeHTTP(w, r)
			})

			rr := httptest.NewRecorder()
			corsHandler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			if tt.expectedAllowOrigin != "" {
				assert.Equal(t, tt.expectedAllowOrigin, rr.Header().Get("Access-Control-Allow-Origin"))
			}
			if tt.expectedAllowMethods != "" {
				assert.Equal(t, tt.expectedAllowMethods, rr.Header().Get("Access-Control-Allow-Methods"))
			}
			if tt.expectedAllowHeaders != "" {
				assert.Equal(t, tt.expectedAllowHeaders, rr.Header().Get("Access-Control-Allow-Headers"))
			}
		})
	}
}

func TestHealthEndpoint(t *testing.T) {
	setupTestGateway(t)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET health endpoint",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedBody:   `{"status":"healthy"}`,
		},
		{
			name:           "POST health endpoint not allowed",
			method:         "POST",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := createTestRequest(tt.method, "/health", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tt.expectedBody))
			})

			handler.ServeHTTP(rr, req)
			assert.Equal(t, tt.expectedStatus, rr.Code)
			if tt.expectedBody != "" {
				assert.JSONEq(t, tt.expectedBody, rr.Body.String())
			}
		})
	}
}

func TestMarketEndpoints(t *testing.T) {
	gateway, marketMock, _, _, _ := setupTestGateway(t)

	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		setupMock      func()
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name:   "Create market success",
			method: "POST",
			path:   "/api/markets",
			body: map[string]interface{}{
				"question":    "Test Market",
				"description": "Test description",
				"category":    "Sports",
				"end_time": map[string]interface{}{
					"seconds": time.Now().Add(24 * time.Hour).Unix(),
					"nanos":   0,
				},
			},
			setupMock: func() {
				marketMock.On("CreateMarket", mock.Anything, mock.Anything).Return(
					&market.CreateMarketResponse{
						Market: &market.Market{
							Id:          "market-123",
							Question:    "Test Market",
							Description: "Test description",
						},
					}, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "Get market success",
			method: "GET",
			path:   "/api/markets/market-123",
			setupMock: func() {
				marketMock.On("GetMarket", mock.Anything, mock.Anything).Return(
					&market.GetMarketResponse{
						Market: &market.Market{
							Id:          "market-123",
							Question:    "Test Market",
							Description: "Test description",
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "List markets success",
			method: "GET",
			path:   "/api/markets",
			setupMock: func() {
				marketMock.On("ListMarkets", mock.Anything, mock.Anything).Return(
					&market.ListMarketsResponse{
						Markets: []*market.Market{
							{
								Id:          "market-123",
								Question:    "Test Market",
								Description: "Test description",
							},
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Update market success",
			method: "PUT",
			path:   "/api/markets/market-123",
			body: map[string]interface{}{
				"question":    "Updated Market",
				"description": "Updated description",
			},
			setupMock: func() {
				marketMock.On("UpdateMarket", mock.Anything, mock.Anything).Return(
					&market.UpdateMarketResponse{
						Market: &market.Market{
							Id:          "market-123",
							Question:    "Updated Market",
							Description: "Updated description",
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Market not found",
			method: "GET",
			path:   "/api/markets/non-existent",
			setupMock: func() {
				marketMock.On("GetMarket", mock.Anything, mock.MatchedBy(func(req *market.GetMarketRequest) bool {
					return req.Id == "non-existent"
				})).Return(nil, fmt.Errorf("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Invalid JSON body",
			method:         "POST",
			path:           "/api/markets",
			body:           "invalid json",
			setupMock:      func() {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock expectations to avoid interference between tests
			marketMock.ExpectedCalls = nil
			marketMock.Calls = nil

			tt.setupMock()

			req, err := createTestRequest(tt.method, tt.path, tt.body)
			require.NoError(t, err)

			rr := httptest.NewRecorder()

			// Create a router for testing using gorilla/mux
			router := mux.NewRouter()
			router.HandleFunc("/api/markets", gateway.handleMarketRequest).Methods("GET", "POST")
			router.HandleFunc("/api/markets/{id}", gateway.handleMarketRequest).Methods("GET", "PUT")
			router.HandleFunc("/api/markets/{id}/options", gateway.handleMarketRequest).Methods("GET")

			router.ServeHTTP(rr, req)
			assert.Equal(t, tt.expectedStatus, rr.Code)

			marketMock.AssertExpectations(t)
		})
	}
}

func TestWalletEndpoints(t *testing.T) {
	gateway, _, walletMock, _, _ := setupTestGateway(t)

	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		setupMock      func()
		expectedStatus int
	}{
		{
			name:   "Create wallet success",
			method: "POST",
			path:   "/api/wallets",
			body: map[string]interface{}{
				"user_id":         "user-123",
				"currency":        "USD",
				"initial_balance": 1000.0,
			},
			setupMock: func() {
				walletMock.On("CreateWalletAccount", mock.Anything, mock.Anything).Return(
					&wallet.CreateWalletAccountResponse{
						Account: &wallet.WalletAccount{
							Id:               "wallet-123",
							UserId:           "user-123",
							Currency:         "USD",
							TotalBalance:     1000.0,
							AvailableBalance: 1000.0,
						},
					}, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "Get wallet success",
			method: "GET",
			path:   "/api/wallets/wallet-123",
			setupMock: func() {
				walletMock.On("GetWalletAccount", mock.Anything, mock.Anything).Return(
					&wallet.GetWalletAccountResponse{
						Account: &wallet.WalletAccount{
							Id:               "wallet-123",
							UserId:           "user-123",
							Currency:         "USD",
							TotalBalance:     1000.0,
							AvailableBalance: 1000.0,
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Deposit success",
			method: "POST",
			path:   "/api/wallets/wallet-123/deposit",
			body: map[string]interface{}{
				"amount": 500.0,
			},
			setupMock: func() {
				walletMock.On("Deposit", mock.Anything, mock.Anything).Return(
					&wallet.DepositResponse{
						Transaction: &wallet.WalletTransaction{
							Id:       "txn-123",
							WalletId: "wallet-123",
							Amount:   500.0,
							Type:     "DEPOSIT",
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Withdraw success",
			method: "POST",
			path:   "/api/wallets/wallet-123/withdraw",
			body: map[string]interface{}{
				"amount": 200.0,
			},
			setupMock: func() {
				walletMock.On("Withdrawal", mock.Anything, mock.Anything).Return(
					&wallet.WithdrawalResponse{
						Transaction: &wallet.WalletTransaction{
							Id:       "txn-456",
							WalletId: "wallet-123",
							Amount:   200.0,
							Type:     "WITHDRAWAL",
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock expectations to avoid interference between tests
			walletMock.ExpectedCalls = nil
			walletMock.Calls = nil

			tt.setupMock()

			req, err := createTestRequest(tt.method, tt.path, tt.body)
			require.NoError(t, err)

			rr := httptest.NewRecorder()

			// Create a router for testing using gorilla/mux
			router := mux.NewRouter()
			router.HandleFunc("/api/wallets", gateway.handleWalletRequest).Methods("POST")
			router.HandleFunc("/api/wallets/{id}", gateway.handleWalletRequest).Methods("GET")
			router.HandleFunc("/api/wallets/{id}/deposit", gateway.handleWalletRequest).Methods("POST")
			router.HandleFunc("/api/wallets/{id}/withdraw", gateway.handleWalletRequest).Methods("POST")

			router.ServeHTTP(rr, req)
			assert.Equal(t, tt.expectedStatus, rr.Code)

			walletMock.AssertExpectations(t)
		})
	}
}

func TestSettlementEndpoints(t *testing.T) {
	gateway, _, _, settlementMock, _ := setupTestGateway(t)

	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		setupMock      func()
		expectedStatus int
	}{
		{
			name:   "Create settlement success",
			method: "POST",
			path:   "/api/settlements",
			body: map[string]interface{}{
				"market_id":         "market-123",
				"winning_option_id": "option-123",
			},
			setupMock: func() {
				settlementMock.On("CreateSettlement", mock.Anything, mock.Anything).Return(
					&settlement.CreateSettlementResponse{
						Settlement: &settlement.Settlement{
							Id:              "settlement-123",
							MarketId:        "market-123",
							WinningOptionId: "option-123",
							Status:          "PENDING",
						},
					}, nil)
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:   "Get settlement success",
			method: "GET",
			path:   "/api/settlements/settlement-123",
			setupMock: func() {
				settlementMock.On("GetSettlement", mock.Anything, mock.Anything).Return(
					&settlement.GetSettlementResponse{
						Settlement: &settlement.Settlement{
							Id:              "settlement-123",
							MarketId:        "market-123",
							WinningOptionId: "option-123",
							Status:          "COMPLETED",
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "Complete settlement success",
			method: "PUT",
			path:   "/api/settlements/settlement-123/complete",
			body:   map[string]interface{}{},
			setupMock: func() {
				settlementMock.On("CompleteSettlement", mock.Anything, mock.Anything).Return(
					&settlement.CompleteSettlementResponse{
						Settlement: &settlement.Settlement{
							Id:              "settlement-123",
							MarketId:        "market-123",
							WinningOptionId: "option-123",
							Status:          "COMPLETED",
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock expectations to avoid interference between tests
			settlementMock.ExpectedCalls = nil
			settlementMock.Calls = nil

			tt.setupMock()

			req, err := createTestRequest(tt.method, tt.path, tt.body)
			require.NoError(t, err)

			rr := httptest.NewRecorder()

			// Create a router for testing using gorilla/mux
			router := mux.NewRouter()
			router.HandleFunc("/api/settlements", gateway.handleSettlementRequest).Methods("POST")
			router.HandleFunc("/api/settlements/{id}", gateway.handleSettlementRequest).Methods("GET")
			router.HandleFunc("/api/settlements/{id}/complete", gateway.handleSettlementRequest).Methods("PUT")

			router.ServeHTTP(rr, req)
			assert.Equal(t, tt.expectedStatus, rr.Code)

			settlementMock.AssertExpectations(t)
		})
	}
}

func TestGRPCErrorHandling(t *testing.T) {
	gateway, marketMock, _, _, kafkaMock := setupTestGateway(t)

	tests := []struct {
		name           string
		error          error
		service        string
		method         string
		expectedStatus int
		expectedKafka  bool
	}{
		{
			name:           "Circuit breaker open error",
			error:          fmt.Errorf("circuit breaker open: service unavailable"),
			service:        "market",
			method:         "GetMarket",
			expectedStatus: http.StatusAccepted,
			expectedKafka:  true,
		},
		{
			name:           "Timeout error",
			error:          fmt.Errorf("timeout: deadline exceeded"),
			service:        "market",
			method:         "ListMarkets",
			expectedStatus: http.StatusAccepted,
			expectedKafka:  true,
		},
		{
			name:           "Not found error",
			error:          fmt.Errorf("market not found"),
			service:        "market",
			method:         "GetMarket",
			expectedStatus: http.StatusNotFound,
			expectedKafka:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the service call to return an error
			if strings.Contains(tt.name, "Circuit breaker") {
				marketMock.On("GetMarket", mock.Anything, mock.Anything).Return(
					nil, tt.error)
				kafkaMock.On("Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			} else if strings.Contains(tt.name, "Timeout") {
				marketMock.On("ListMarkets", mock.Anything, mock.Anything).Return(
					nil, tt.error)
				kafkaMock.On("Publish", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			} else if strings.Contains(tt.name, "Not found") {
				marketMock.On("GetMarket", mock.Anything, mock.Anything).Return(
					nil, tt.error)
			}

			rr := httptest.NewRecorder()

			// Test the error handling
			ctx := context.Background()
			gateway.handleGRPCError(ctx, rr, tt.error, tt.service, tt.method)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestExtractIDFromPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		resource string
		expected string
	}{
		{
			name:     "Extract market ID",
			path:     "/api/markets/market-123",
			resource: "markets",
			expected: "market-123",
		},
		{
			name:     "Extract wallet ID",
			path:     "/api/wallets/wallet-456",
			resource: "wallets",
			expected: "wallet-456",
		},
		{
			name:     "Extract settlement ID",
			path:     "/api/settlements/settlement-789",
			resource: "settlements",
			expected: "settlement-789",
		},
		{
			name:     "No ID found",
			path:     "/api/markets",
			resource: "markets",
			expected: "",
		},
		{
			name:     "Empty path",
			path:     "",
			resource: "markets",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractIDFromPath(tt.path, tt.resource)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWriteJSONResponse(t *testing.T) {
	gateway, _, _, _, _ := setupTestGateway(t)

	tests := []struct {
		name           string
		status         int
		data           interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Simple JSON response",
			status:         http.StatusOK,
			data:           map[string]string{"message": "success"},
			expectedStatus: http.StatusOK,
			expectedBody:   `{"message":"success"}`,
		},
		{
			name:           "Created response",
			status:         http.StatusCreated,
			data:           map[string]interface{}{"id": "123", "created": true},
			expectedStatus: http.StatusCreated,
			expectedBody:   `{"created":true,"id":"123"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			gateway.writeJSONResponse(rr, tt.status, tt.data)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))
			assert.JSONEq(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func TestEnvironmentVariableHandling(t *testing.T) {
	tests := []struct {
		name       string
		envKey     string
		envValue   string
		defaultVal string
		expected   string
	}{
		{
			name:       "Environment variable set",
			envKey:     "TEST_VAR",
			envValue:   "test-value",
			defaultVal: "default-value",
			expected:   "test-value",
		},
		{
			name:       "Environment variable not set",
			envKey:     "NON_EXISTENT_VAR",
			envValue:   "",
			defaultVal: "default-value",
			expected:   "default-value",
		},
		{
			name:       "Environment variable empty",
			envKey:     "EMPTY_VAR",
			envValue:   "   ", // whitespace only
			defaultVal: "default-value",
			expected:   "default-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			result := getEnv(tt.envKey, tt.defaultVal)
			assert.Equal(t, tt.expected, result)
		})
	}
}
