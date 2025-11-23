package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/gorilla/mux"
    "go.uber.org/zap"

    resgrpc "github.com/aegis/shared/grpc"
    "github.com/aegis/shared/kafka"
    "github.com/aegis/shared/metrics"
    market "github.com/aegis/proto/gen/market"
    wallet "github.com/aegis/proto/gen/wallet"
    settlement "github.com/aegis/proto/gen/settlement"
)

type APIGateway struct {
    logger          *zap.Logger
    metrics         *metrics.Registry
    marketClient    *resgrpc.ResilientClient
    walletClient    *resgrpc.ResilientClient
    settlementClient *resgrpc.ResilientClient
    kafkaProducer   *kafka.Producer
    marketStub      market.MarketServiceClient
    walletStub      wallet.WalletServiceClient
    settlementStub  settlement.SettlementServiceClient
}

func NewAPIGateway(logger *zap.Logger, metricsRegistry *metrics.Registry) (*APIGateway, error) {
    marketConfig := resgrpc.DefaultClientConfig("market", "market-service:50051")
    walletConfig := resgrpc.DefaultClientConfig("wallet", "wallet-service:50052")
    settlementConfig := resgrpc.DefaultClientConfig("settlement", "settlement-service:50053")

    marketClient, err := resgrpc.NewResilientClient(marketConfig, logger, metricsRegistry)
    if err != nil {
        return nil, fmt.Errorf("failed to create market client: %w", err)
    }

    walletClient, err := resgrpc.NewResilientClient(walletConfig, logger, metricsRegistry)
    if err != nil {
        return nil, fmt.Errorf("failed to create wallet client: %w", err)
    }

    settlementClient, err := resgrpc.NewResilientClient(settlementConfig, logger, metricsRegistry)
    if err != nil {
        return nil, fmt.Errorf("failed to create settlement client: %w", err)
    }

    kafkaProducer := kafka.NewProducer(kafka.Config{Brokers: []string{"kafka:9092"}}, logger)

    return &APIGateway{
        logger:           logger,
        metrics:          metricsRegistry,
        marketClient:     marketClient,
        walletClient:     walletClient,
        settlementClient: settlementClient,
        kafkaProducer:    kafkaProducer,
        marketStub:       market.NewMarketServiceClient(marketClient.GetConnection()),
        walletStub:       wallet.NewWalletServiceClient(walletClient.GetConnection()),
        settlementStub:   settlement.NewSettlementServiceClient(settlementClient.GetConnection()),
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
	
    resp, err := g.marketStub.ListMarkets(ctx, req)
	
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

    req := &market.GetMarketRequest{Id: marketID}
	
    resp, err := g.marketStub.GetMarket(ctx, req)
	
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

    resp, err := g.marketStub.CreateMarket(ctx, &req)
	
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
    req.Id = marketID

    resp, err := g.marketStub.UpdateMarket(ctx, &req)
	
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
	
    resp, err := g.marketStub.GetMarketOptions(ctx, req)
	
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

    resp, err := g.walletStub.CreateWalletAccount(ctx, &req)
	
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

    req := &wallet.GetWalletAccountRequest{Id: walletID}
	
    resp, err := g.walletStub.GetWalletAccount(ctx, req)
	
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
    req.AccountId = walletID

    resp, err := g.walletStub.Deposit(ctx, &req)
	
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
    req.AccountId = walletID

    resp, err := g.walletStub.Withdrawal(ctx, &req)
	
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

    resp, err := g.settlementStub.CreateSettlement(ctx, &req)
	
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

    req := &settlement.GetSettlementRequest{Id: settlementID}
	
    resp, err := g.settlementStub.GetSettlement(ctx, req)
	
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
    req.Id = settlementID

    resp, err := g.settlementStub.CompleteSettlement(ctx, &req)
	
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
		
        if kafkaErr := g.kafkaProducer.Publish(ctx, topic, fmt.Sprintf("%s_%s", service, method), message); kafkaErr != nil {
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

    metricsRegistry := metrics.NewRegistry(logger)

    gateway, err := NewAPIGateway(logger, metricsRegistry)
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