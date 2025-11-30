package main

import (
    "context"
    "encoding/json"
    "bytes"
    "fmt"
    "io"
    "net/http"
    "strings"
    "time"
    "os"

    "github.com/gorilla/mux"
    "go.uber.org/zap"
    "github.com/go-chi/cors"

    resgrpc "github.com/aegis/shared/grpc"
    "github.com/aegis/shared/kafka"
    "github.com/aegis/shared/metrics"
    market "github.com/aegis/proto/gen/market"
    wallet "github.com/aegis/proto/gen/wallet"
    settlement "github.com/aegis/proto/gen/settlement"
    transaction "github.com/aegis/proto/gen/transaction"
    "google.golang.org/protobuf/types/known/timestamppb"
)

type KafkaPublisher interface {
	Publish(ctx context.Context, topic string, key string, value interface{}) error
}

type APIGateway struct {
    logger          *zap.Logger
    metrics         *metrics.Registry
    marketClient    *resgrpc.ResilientClient
    walletClient    *resgrpc.ResilientClient
    settlementClient *resgrpc.ResilientClient
    transactionClient *resgrpc.ResilientClient
    kafkaProducer   KafkaPublisher
    marketStub      market.MarketServiceClient
    walletStub      wallet.WalletServiceClient
    settlementStub  settlement.SettlementServiceClient
    transactionStub transaction.TransactionServiceClient
}

func NewAPIGateway(logger *zap.Logger, metricsRegistry *metrics.Registry) (*APIGateway, error) {
    marketAddr := getEnv("MARKET_SERVICE_GRPC_ADDR", "market-service:50051")
    walletAddr := getEnv("WALLET_SERVICE_GRPC_ADDR", "wallet-service:50052")
    settlementAddr := getEnv("SETTLEMENT_SERVICE_GRPC_ADDR", "settlement-service:50053")
    transactionAddr := getEnv("TRANSACTION_SERVICE_GRPC_ADDR", "transaction-service:50052")

    marketConfig := resgrpc.DefaultClientConfig("market", marketAddr)
    walletConfig := resgrpc.DefaultClientConfig("wallet", walletAddr)
    settlementConfig := resgrpc.DefaultClientConfig("settlement", settlementAddr)
    transactionConfig := resgrpc.DefaultClientConfig("transaction", transactionAddr)

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

    transactionClient, err := resgrpc.NewResilientClient(transactionConfig, logger, metricsRegistry)
    if err != nil {
        return nil, fmt.Errorf("failed to create transaction client: %w", err)
    }

    brokersEnv := getEnv("KAFKA_BROKERS", "kafka:29092")
    var brokers []string
    for _, b := range strings.Split(brokersEnv, ",") {
        b = strings.TrimSpace(b)
        if b != "" {
            brokers = append(brokers, b)
        }
    }
    kafkaProducer := kafka.NewProducer(kafka.Config{Brokers: brokers}, logger)

    return &APIGateway{
        logger:           logger,
        metrics:          metricsRegistry,
        marketClient:     marketClient,
        walletClient:     walletClient,
        settlementClient: settlementClient,
        transactionClient: transactionClient,
        kafkaProducer:    kafkaProducer,
        marketStub:       market.NewMarketServiceClient(marketClient.GetConnection()),
        walletStub:       wallet.NewWalletServiceClient(walletClient.GetConnection()),
        settlementStub:   settlement.NewSettlementServiceClient(settlementClient.GetConnection()),
        transactionStub:  transaction.NewTransactionServiceClient(transactionClient.GetConnection()),
    }, nil
}

func (g *APIGateway) handleMarketRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	path := r.URL.Path
	method := r.Method

	switch {
	case method == "GET" && strings.HasSuffix(path, "/markets"):
		g.listMarkets(ctx, w, r)
	case method == "GET" && strings.Contains(path, "/markets/") && strings.HasSuffix(path, "/options"):
		g.getMarketOptions(ctx, w, r)
	case method == "GET" && strings.Contains(path, "/markets/"):
		g.getMarket(ctx, w, r)
	case method == "POST" && strings.HasSuffix(path, "/markets"):
		g.createMarket(ctx, w, r)
	case method == "PUT" && strings.Contains(path, "/markets/"):
		g.updateMarket(ctx, w, r)
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
	g.logger.Info("HTTP request",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.String("remote", r.RemoteAddr))

	bodyBytes, _ := io.ReadAll(r.Body)
	g.logger.Info("Request body", zap.String("body", string(bodyBytes)))
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var body struct {
		Question    string   `json:"question"`
		Description string   `json:"description"`
		Options     []string `json:"options"`
		EndTime     string   `json:"end_time"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req := market.CreateMarketRequest{
		Question:    body.Question,
		Description: body.Description,
		Options:     body.Options,
	}
	if body.EndTime != "" {
		t, err := time.Parse(time.RFC3339, body.EndTime)
		if err != nil {
			http.Error(w, "Invalid end_time format", http.StatusBadRequest)
			return
		}
		req.EndTime = timestamppb.New(t)
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

func (g *APIGateway) handleUserRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	path := r.URL.Path
	method := r.Method

	switch {
	case method == "GET" && strings.Contains(path, "/users/wallet/"):
		// GET /api/users/wallet/{wallet_address}
		g.getUserByWallet(ctx, w, r)
	case method == "GET" && strings.Contains(path, "/users/") && !strings.Contains(path, "/wallet/"):
		// GET /api/users/{id}
		g.getUser(ctx, w, r)
	case method == "POST" && strings.HasSuffix(path, "/users"):
		// POST /api/users
		g.createUser(ctx, w, r)
	case method == "PUT" && strings.Contains(path, "/users/") && !strings.Contains(path, "/wallet/"):
		// PUT /api/users/{id}
		g.updateUser(ctx, w, r)
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

// User management handlers

func (g *APIGateway) getUser(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := extractIDFromPath(r.URL.Path, "users")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	req := &wallet.GetUserRequest{Id: userID}
	
	resp, err := g.walletStub.GetUser(ctx, req)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "wallet", "GetUser")
		return
	}

	g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) getUserByWallet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Extract wallet address from path like /api/users/wallet/{wallet_address}
	parts := strings.Split(r.URL.Path, "/")
	var walletAddress string
	for i, part := range parts {
		if part == "wallet" && i+1 < len(parts) {
			walletAddress = parts[i+1]
			break
		}
	}
	
	if walletAddress == "" {
		http.Error(w, "Wallet address required", http.StatusBadRequest)
		return
	}

	req := &wallet.GetUserByWalletRequest{WalletAddress: walletAddress}
	
	resp, err := g.walletStub.GetUserByWallet(ctx, req)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "wallet", "GetUserByWallet")
		return
	}

	g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) createUser(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var body struct {
		WalletAddress string  `json:"wallet_address"`
		Balance       float64 `json:"balance"`
		Role          string  `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req := wallet.CreateUserRequest{
		WalletAddress: body.WalletAddress,
		Balance:       body.Balance,
		Role:          body.Role,
	}

	resp, err := g.walletStub.CreateUser(ctx, &req)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "wallet", "CreateUser")
		return
	}

	g.writeJSONResponse(w, http.StatusCreated, resp)
}

func (g *APIGateway) updateUser(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := extractIDFromPath(r.URL.Path, "users")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	var body struct {
		WalletAddress string  `json:"wallet_address,omitempty"`
		Balance       float64 `json:"balance,omitempty"`
		Role          string  `json:"role,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	req := wallet.UpdateUserRequest{
		Id:            userID,
		WalletAddress: body.WalletAddress,
		Balance:       body.Balance,
		Role:          body.Role,
	}

	resp, err := g.walletStub.UpdateUser(ctx, &req)
	
	if err != nil {
		g.handleGRPCError(ctx, w, err, "wallet", "UpdateUser")
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

		if g.kafkaProducer != nil {
			if kafkaErr := g.kafkaProducer.Publish(ctx, topic, fmt.Sprintf("%s_%s", service, method), message); kafkaErr != nil {
				g.logger.Error("Failed to send fallback message to Kafka",
					zap.String("topic", topic),
					zap.Error(kafkaErr),
				)
			}
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

func (g *APIGateway) handleTransactionRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	path := r.URL.Path
	method := r.Method

	switch {
	case method == "POST" && strings.HasSuffix(path, "/transactions"):
		g.createTransaction(ctx, w, r)
	case method == "GET" && strings.Contains(path, "/transactions"):
		g.getTransactions(ctx, w, r)
	default:
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (g *APIGateway) createTransaction(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var body struct {
		UserID          string  `json:"user_id"`
		MarketID        string  `json:"market_id"`
		OptionID        string  `json:"option_id"`
		TransactionType string  `json:"transaction_type"`
		NumberOfShares  int32   `json:"number_of_shares"`
		PricePerShare   float64 `json:"price_per_share"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert transaction type string to enum
	var txType transaction.TransactionType
	switch body.TransactionType {
	case "BUY":
		txType = transaction.TransactionType_BUY
	case "SELL":
		txType = transaction.TransactionType_SELL
	default:
		http.Error(w, "Invalid transaction_type, must be BUY or SELL", http.StatusBadRequest)
		return
	}

	req := &transaction.TransactionRequest{
		UserId:          body.UserID,
		MarketId:        body.MarketID,
		OptionId:        body.OptionID,
		TransactionType: txType,
		NumberOfShares:  body.NumberOfShares,
		PricePerShare:   body.PricePerShare,
	}

	resp, err := g.transactionStub.CreateTransaction(ctx, req)
	if err != nil {
		g.handleGRPCError(ctx, w, err, "transaction", "CreateTransaction")
		return
	}

	g.writeJSONResponse(w, http.StatusCreated, resp)
}

func (g *APIGateway) getTransactions(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	marketID := r.URL.Query().Get("market_id")

	req := &transaction.GetTransactionsRequest{}
	if userID != "" {
		req.UserId = &userID
	}
	if marketID != "" {
		req.MarketId = &marketID
	}

	resp, err := g.transactionStub.GetTransactions(ctx, req)
	if err != nil {
		g.handleGRPCError(ctx, w, err, "transaction", "GetTransactions")
		return
	}

	g.writeJSONResponse(w, http.StatusOK, resp)
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

func (g *APIGateway) writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
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

	
	// User routes (now handled by Wallet Service)
	router.HandleFunc("/api/users", gateway.handleUserRequest).Methods("POST")
	router.HandleFunc("/api/users/{id}", gateway.handleUserRequest).Methods("GET", "PUT")
	router.HandleFunc("/api/users/wallet/{wallet_address}", gateway.handleUserRequest).Methods("GET")
	
	// Settlement routes
	router.HandleFunc("/api/settlements", gateway.handleSettlementRequest).Methods("POST")
	router.HandleFunc("/api/settlements/{id}", gateway.handleSettlementRequest).Methods("GET")
	router.HandleFunc("/api/settlements/{id}/complete", gateway.handleSettlementRequest).Methods("PUT")
	
	// Transaction routes
	router.HandleFunc("/api/transactions", gateway.handleTransactionRequest).Methods("GET", "POST")
	
	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
	}).Methods("GET")

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

	corsMiddleware := cors.Handler(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   methods,
		AllowedHeaders:   headers,
		ExposedHeaders:   []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	handler := corsMiddleware(router)

	logger.Info("Starting API Gateway on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

func getEnv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v != "" {
		return v
	}
	return def
}
