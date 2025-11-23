package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"

	"aegis/shared/grpc"
	"aegis/shared/kafka"
	"aegis/shared/metrics"
	"aegis/proto/market"
	"aegis/proto/wallet"
	"aegis/proto/settlement"
)

type APIGateway struct {
	logger          *zap.Logger
	metrics         *metrics.Collector
	marketClient    *grpc.ResilientClient
	walletClient    *grpc.ResilientClient
	settlementClient *grpc.ResilientClient
	kafkaProducer   *kafka.Producer
}

func NewAPIGateway(logger *zap.Logger, metricsCollector *metrics.Collector) (*APIGateway, error) {
	// Create resilient gRPC clients with circuit breaker and retry
	marketClient, err := grpc.NewResilientClient("market-service:50051", logger, metricsCollector)
	if err != nil {
		return nil, fmt.Errorf("failed to create market client: %w", err)
	}

	walletClient, err := grpc.NewResilientClient("wallet-service:50052", logger, metricsCollector)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet client: %w", err)
	}

	settlementClient, err := grpc.NewResilientClient("settlement-service:50053", logger, metricsCollector)
	if err != nil {
		return nil, fmt.Errorf("failed to create settlement client: %w", err)
	}

	kafkaProducer, err := kafka.NewProducer([]string{"kafka:9092"}, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka producer: %w", err)
	}

	return &APIGateway{
		logger:           logger,
		metrics:          metricsCollector,
		marketClient:     marketClient,
		walletClient:     walletClient,
		settlementClient: settlementClient,
		kafkaProducer:    kafkaProducer,
	}, nil
}

func (g *APIGateway) handleMarketRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	path := r.URL.Path
	method := r.Method

	switch {
	case method == "GET" && strings.HasSuffix(path, "/markets"):
		g.listMarkets(ctx, w, r)
	case method == "GET" && strings.Contains(path, "/markets/"):
		g.getMarket(ctx, w, r)
	case method == "POST" && strings.HasSuffix(path, "/markets"):
		g.createMarket(ctx, w, r)
	case method == "PUT" && strings.Contains(path, "/markets/"):
		g.updateMarket(ctx, w, r)
	case method == "GET" && strings.Contains(path, "/markets/") && strings.HasSuffix(path, "/options"):
		g.getMarketOptions(ctx, w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (g *APIGateway) listMarkets(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	req := &market.ListMarketsRequest{}
	
	var resp *market.ListMarketsResponse
	err := g.marketClient.Invoke(ctx, "/market.MarketService/ListMarkets", req, &resp)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "market", "ListMarkets")
		return
	}

	g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) getMarket(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	marketID := extractIDFromPath(r.URL.Path, "markets")
	if marketID == "" {
		http.Error(w, "Market ID required", http.StatusBadRequest)
		return
	}

	req := &market.GetMarketRequest{MarketId: marketID}
	
	var resp *market.GetMarketResponse
	err := g.marketClient.Invoke(ctx, "/market.MarketService/GetMarket", req, &resp)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "market", "GetMarket")
		return
	}

	g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) createMarket(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req market.CreateMarketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var resp *market.CreateMarketResponse
	err := g.marketClient.Invoke(ctx, "/market.MarketService/CreateMarket", &req, &resp)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "market", "CreateMarket")
		return
	}

	g.writeJSONResponse(w, http.StatusCreated, resp)
}

func (g *APIGateway) updateMarket(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	marketID := extractIDFromPath(r.URL.Path, "markets")
	if marketID == "" {
		http.Error(w, "Market ID required", http.StatusBadRequest)
		return
	}

	var req market.UpdateMarketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.MarketId = marketID

	var resp *market.UpdateMarketResponse
	err := g.marketClient.Invoke(ctx, "/market.MarketService/UpdateMarket", &req, &resp)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "market", "UpdateMarket")
		return
	}

	g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) getMarketOptions(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	marketID := extractIDFromPath(r.URL.Path, "markets")
	if marketID == "" {
		http.Error(w, "Market ID required", http.StatusBadRequest)
		return
	}

	req := &market.GetMarketOptionsRequest{MarketId: marketID}
	
	var resp *market.GetMarketOptionsResponse
	err := g.marketClient.Invoke(ctx, "/market.MarketService/GetMarketOptions", req, &resp)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "market", "GetMarketOptions")
		return
	}

	g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) handleWalletRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	path := r.URL.Path
	method := r.Method

	switch {
	case method == "POST" && strings.HasSuffix(path, "/wallets"):
		g.createWallet(ctx, w, r)
	case method == "GET" && strings.Contains(path, "/wallets/"):
		g.getWallet(ctx, w, r)
	case method == "POST" && strings.Contains(path, "/wallets/") && strings.HasSuffix(path, "/deposit"):
		g.deposit(ctx, w, r)
	case method == "POST" && strings.Contains(path, "/wallets/") && strings.HasSuffix(path, "/withdraw"):
		g.withdraw(ctx, w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (g *APIGateway) createWallet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req wallet.CreateWalletAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var resp *wallet.CreateWalletAccountResponse
	err := g.walletClient.Invoke(ctx, "/wallet.WalletService/CreateWalletAccount", &req, &resp)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "wallet", "CreateWalletAccount")
		return
	}

	g.writeJSONResponse(w, http.StatusCreated, resp)
}

func (g *APIGateway) getWallet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	walletID := extractIDFromPath(r.URL.Path, "wallets")
	if walletID == "" {
		http.Error(w, "Wallet ID required", http.StatusBadRequest)
		return
	}

	req := &wallet.GetWalletAccountRequest{WalletId: walletID}
	
	var resp *wallet.GetWalletAccountResponse
	err := g.walletClient.Invoke(ctx, "/wallet.WalletService/GetWalletAccount", req, &resp)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "wallet", "GetWalletAccount")
		return
	}

	g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) deposit(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	walletID := extractIDFromPath(r.URL.Path, "wallets")
	if walletID == "" {
		http.Error(w, "Wallet ID required", http.StatusBadRequest)
		return
	}

	var req wallet.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.WalletId = walletID

	var resp *wallet.DepositResponse
	err := g.walletClient.Invoke(ctx, "/wallet.WalletService/Deposit", &req, &resp)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "wallet", "Deposit")
		return
	}

	g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) withdraw(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	walletID := extractIDFromPath(r.URL.Path, "wallets")
	if walletID == "" {
		http.Error(w, "Wallet ID required", http.StatusBadRequest)
		return
	}

	var req wallet.WithdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.WalletId = walletID

	var resp *wallet.WithdrawalResponse
	err := g.walletClient.Invoke(ctx, "/wallet.WalletService/Withdrawal", &req, &resp)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "wallet", "Withdrawal")
		return
	}

	g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) handleSettlementRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	path := r.URL.Path
	method := r.Method

	switch {
	case method == "POST" && strings.HasSuffix(path, "/settlements"):
		g.createSettlement(ctx, w, r)
	case method == "GET" && strings.Contains(path, "/settlements/"):
		g.getSettlement(ctx, w, r)
	case method == "PUT" && strings.Contains(path, "/settlements/") && strings.HasSuffix(path, "/complete"):
		g.completeSettlement(ctx, w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (g *APIGateway) createSettlement(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req settlement.CreateSettlementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var resp *settlement.CreateSettlementResponse
	err := g.settlementClient.Invoke(ctx, "/settlement.SettlementService/CreateSettlement", &req, &resp)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "settlement", "CreateSettlement")
		return
	}

	g.writeJSONResponse(w, http.StatusCreated, resp)
}

func (g *APIGateway) getSettlement(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	settlementID := extractIDFromPath(r.URL.Path, "settlements")
	if settlementID == "" {
		http.Error(w, "Settlement ID required", http.StatusBadRequest)
		return
	}

	req := &settlement.GetSettlementRequest{SettlementId: settlementID}
	
	var resp *settlement.GetSettlementResponse
	err := g.settlementClient.Invoke(ctx, "/settlement.SettlementService/GetSettlement", req, &resp)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "settlement", "GetSettlement")
		return
	}

	g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) completeSettlement(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	settlementID := extractIDFromPath(r.URL.Path, "settlements")
	if settlementID == "" {
		http.Error(w, "Settlement ID required", http.StatusBadRequest)
		return
	}

	var req settlement.CompleteSettlementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	req.SettlementId = settlementID

	var resp *settlement.CompleteSettlementResponse
	err := g.settlementClient.Invoke(ctx, "/settlement.SettlementService/CompleteSettlement", &req, &resp)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "settlement", "CompleteSettlement")
		return
	}

	g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) handleGRPCError(ctx context.Context, w http.ResponseWriter, err error, service, method string) {
	g.logger.Error("gRPC call failed",
		zap.String("service", service),
		zap.String("method", method),
		zap.Error(err),
	)

	// Check if this was a circuit breaker or timeout error that triggered Kafka fallback
	if strings.Contains(err.Error(), "circuit breaker open") || strings.Contains(err.Error(), "timeout") {
		// Send to Kafka for async processing
		topic := fmt.Sprintf("%s.%s.fallback", service, method)
		message := map[string]interface{}{
			"service":   service,
			"method":    method,
			"timestamp": time.Now().Unix(),
			"error":     err.Error(),
		}
		
		if kafkaErr := g.kafkaProducer.SendMessage(ctx, topic, message); kafkaErr != nil {
			g.logger.Error("Failed to send fallback message to Kafka",
				zap.String("topic", topic),
				zap.Error(kafkaErr),
			)
		}
		
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "accepted",
			"message": "Request queued for async processing",
		})
		return
	}

	// Convert gRPC error to HTTP status
	status := http.StatusInternalServerError
	if strings.Contains(err.Error(), "not found") {
		status = http.StatusNotFound
	} else if strings.Contains(err.Error(), "invalid") {
		status = http.StatusBadRequest
	}

	http.Error(w, err.Error(), status)
}

func (g *APIGateway) writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func extractIDFromPath(path, resource string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == resource && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	metricsCollector := metrics.NewCollector()

	gateway, err := NewAPIGateway(logger, metricsCollector)
	if err != nil {
		logger.Fatal("Failed to create API Gateway", zap.Error(err))
	}

	router := mux.NewRouter()
	
	// Market routes
	router.HandleFunc("/api/markets", gateway.handleMarketRequest).Methods("GET", "POST")
	router.HandleFunc("/api/markets/{id}", gateway.handleMarketRequest).Methods("GET", "PUT")
	router.HandleFunc("/api/markets/{id}/options", gateway.handleMarketRequest).Methods("GET")
	
	// Wallet routes
	router.HandleFunc("/api/wallets", gateway.handleWalletRequest).Methods("POST")
	router.HandleFunc("/api/wallets/{id}", gateway.handleWalletRequest).Methods("GET")
	router.HandleFunc("/api/wallets/{id}/deposit", gateway.handleWalletRequest).Methods("POST")
	router.HandleFunc("/api/wallets/{id}/withdraw", gateway.handleWalletRequest).Methods("POST")
	
	// Settlement routes
	router.HandleFunc("/api/settlements", gateway.handleSettlementRequest).Methods("POST")
	router.HandleFunc("/api/settlements/{id}", gateway.handleSettlementRequest).Methods("GET")
	router.HandleFunc("/api/settlements/{id}/complete", gateway.handleSettlementRequest).Methods("PUT")
	
	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}).Methods("GET")

	logger.Info("Starting API Gateway on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}