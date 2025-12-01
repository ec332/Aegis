package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
    "strconv"

	"github.com/go-chi/cors"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

	market "github.com/aegis/proto/gen/market"
	settlement "github.com/aegis/proto/gen/settlement"
	transaction "github.com/aegis/proto/gen/transaction"
	wallet "github.com/aegis/proto/gen/wallet"
	resgrpc "github.com/aegis/shared/grpc"
	"github.com/aegis/shared/kafka"
	"github.com/aegis/shared/metrics"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type KafkaPublisher interface {
	Publish(ctx context.Context, topic string, key string, value interface{}) error
}

type APIGateway struct {
	logger            *zap.Logger
	metrics           *metrics.Registry
	marketClient      *resgrpc.ResilientClient
	walletClient      *resgrpc.ResilientClient
	settlementClient  *resgrpc.ResilientClient
	transactionClient *resgrpc.ResilientClient
	kafkaProducer     KafkaPublisher
	marketStub        market.MarketServiceClient
	walletStub        wallet.WalletServiceClient
	settlementStub    settlement.SettlementServiceClient
	transactionStub   transaction.TransactionServiceClient
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
		logger:            logger,
		metrics:           metricsRegistry,
		marketClient:      marketClient,
		walletClient:      walletClient,
		settlementClient:  settlementClient,
		transactionClient: transactionClient,
		kafkaProducer:     kafkaProducer,
		marketStub:        market.NewMarketServiceClient(marketClient.GetConnection()),
		walletStub:        wallet.NewWalletServiceClient(walletClient.GetConnection()),
		settlementStub:    settlement.NewSettlementServiceClient(settlementClient.GetConnection()),
		transactionStub:   transaction.NewTransactionServiceClient(transactionClient.GetConnection()),
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

    var listResp *market.ListMarketsResponse
    lr, lerr := g.marketStub.ListMarkets(ctx, &market.ListMarketsRequest{})
    if lerr == nil {
        listResp = lr
    } else {
        g.logger.Warn("ListMarkets after create failed", zap.Error(lerr))
    }

    var optsResp *market.GetMarketOptionsResponse
    if resp != nil && resp.Market != nil && strings.TrimSpace(resp.Market.Id) != "" {
        or, oerr := g.marketStub.GetMarketOptions(ctx, &market.GetMarketOptionsRequest{MarketId: resp.Market.Id})
        if oerr == nil {
            optsResp = or
        } else {
            g.logger.Warn("GetMarketOptions after create failed", zap.Error(oerr))
        }
    }

    out := map[string]interface{}{
        "market":  resp.Market,
        "markets": func() interface{} { if listResp != nil { return listResp.Markets } ; return []interface{}{} }(),
        "options": func() interface{} { if optsResp != nil { return optsResp.Options } ; return []interface{}{} }(),
    }
    g.writeJSONResponse(w, http.StatusCreated, out)
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
        if strings.HasSuffix(path, "/transactions") {
            g.getWalletTransactions(ctx, w, r)
        } else if strings.Contains(path, "/wallets/user/") {
            g.getWalletByUserID(ctx, w, r)
        } else {
            g.getWallet(ctx, w, r)
        }
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
	case method == "POST" && strings.HasSuffix(path, "/auth/nonce"):
		g.requestNonce(ctx, w, r)
	case method == "POST" && strings.HasSuffix(path, "/auth/verify"):
		g.verifySignature(ctx, w, r)
	case method == "GET" && strings.HasSuffix(path, "/auth/me"):
		g.me(ctx, w, r)
	case method == "POST" && strings.HasSuffix(path, "/user/nonce"):
		g.requestNonce(ctx, w, r)
	case method == "POST" && strings.HasSuffix(path, "/user/verify"):
		g.verifySignature(ctx, w, r)
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
    authz := strings.TrimSpace(r.Header.Get("Authorization"))
    if authz == "" {
        http.Error(w, "authorization required", http.StatusUnauthorized)
        return
    }
    var req wallet.CreateWalletAccountRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    if strings.TrimSpace(req.Currency) == "" {
        req.Currency = strings.TrimSpace(getEnv("WALLET_DEFAULT_CURRENCY", "USD"))
    }
    
    ctx = g.withAuth(ctx, r)
    resp, err := g.walletStub.CreateWalletAccount(ctx, &req)

	if err != nil {
		g.handleGRPCError(ctx, w, err, "wallet", "CreateWalletAccount")
		return
	}

	g.writeJSONResponse(w, http.StatusCreated, resp)
}

func (g *APIGateway) getWallet(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    authz := strings.TrimSpace(r.Header.Get("Authorization"))
    if authz == "" {
        http.Error(w, "authorization required", http.StatusUnauthorized)
        return
    }
    walletID := extractIDFromPath(r.URL.Path, "wallets")
    if walletID == "" {
        http.Error(w, "Wallet ID required", http.StatusBadRequest)
        return
    }

	req := &wallet.GetWalletAccountRequest{Id: walletID}

	ctx = g.withAuth(ctx, r)
	resp, err := g.walletStub.GetWalletAccount(ctx, req)

	if err != nil {
		g.handleGRPCError(ctx, w, err, "wallet", "GetWalletAccount")
		return
	}

	g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) getWalletByUserID(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    authz := strings.TrimSpace(r.Header.Get("Authorization"))
    if authz == "" {
        http.Error(w, "authorization required", http.StatusUnauthorized)
        return
    }
    parts := strings.Split(r.URL.Path, "/")
    var userID string
    for i, p := range parts {
        if p == "user" && i+1 < len(parts) {
            userID = parts[i+1]
            break
        }
    }
    if strings.TrimSpace(userID) == "" {
        http.Error(w, "User ID required", http.StatusBadRequest)
        return
    }
    defCur := strings.TrimSpace(getEnv("WALLET_DEFAULT_CURRENCY", "USD"))
    req := &wallet.GetWalletAccountByUserIDRequest{UserId: userID, Currency: defCur}
    ctx = g.withAuth(ctx, r)
    resp, err := g.walletStub.GetWalletAccountByUserID(ctx, req)
    if err != nil {
        msg := strings.ToLower(err.Error())
        if strings.Contains(msg, "not found") {
            g.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"account": nil})
            return
        }
        g.handleGRPCError(ctx, w, err, "wallet", "GetWalletAccountByUserID")
        return
    }
    g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) deposit(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    authz := strings.TrimSpace(r.Header.Get("Authorization"))
    if authz == "" {
        http.Error(w, "authorization required", http.StatusUnauthorized)
        return
    }
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

	ctx = g.withAuth(ctx, r)
	resp, err := g.walletStub.Deposit(ctx, &req)

	if err != nil {
		g.handleGRPCError(ctx, w, err, "wallet", "Deposit")
		return
	}

	g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) withdraw(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    authz := strings.TrimSpace(r.Header.Get("Authorization"))
    if authz == "" {
        http.Error(w, "authorization required", http.StatusUnauthorized)
        return
    }
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

	ctx = g.withAuth(ctx, r)
	resp, err := g.walletStub.Withdrawal(ctx, &req)

	if err != nil {
		g.handleGRPCError(ctx, w, err, "wallet", "Withdrawal")
		return
	}

	g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) getWalletTransactions(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    authz := strings.TrimSpace(r.Header.Get("Authorization"))
    if authz == "" {
        http.Error(w, "authorization required", http.StatusUnauthorized)
        return
    }
    walletID := extractIDFromPath(r.URL.Path, "wallets")
    if walletID == "" {
        http.Error(w, "Wallet ID required", http.StatusBadRequest)
        return
    }
    q := r.URL.Query()
    var limit int32 = 50
    var offset int32 = 0
    if v := strings.TrimSpace(q.Get("limit")); v != "" {
        if n, err := strconv.Atoi(v); err == nil {
            limit = int32(n)
        }
    }
    if v := strings.TrimSpace(q.Get("offset")); v != "" {
        if n, err := strconv.Atoi(v); err == nil {
            offset = int32(n)
        }
    }
    req := &wallet.GetWalletTransactionsRequest{AccountId: walletID, Limit: limit, Offset: offset}
    ctx = g.withAuth(ctx, r)
    resp, err := g.walletStub.GetWalletTransactions(ctx, req)
    if err != nil {
        g.handleGRPCError(ctx, w, err, "wallet", "GetWalletTransactions")
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
    st, ok := status.FromError(err)
    code := codes.Unknown
    if ok {
        code = st.Code()
    }

    if service == "wallet" && method == "GetWalletAccountByUserID" && code == codes.NotFound {
        g.writeJSONResponse(w, http.StatusOK, map[string]interface{}{"account": nil})
        return
    }

    g.logger.Error("gRPC call failed",
        zap.String("service", service),
        zap.String("method", method),
        zap.String("code", code.String()),
        zap.Error(err),
    )

    if code == codes.Unavailable || code == codes.DeadlineExceeded ||
        strings.Contains(strings.ToLower(err.Error()), "circuit breaker open") || strings.Contains(strings.ToLower(err.Error()), "timeout") {
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

    httpStatus := http.StatusInternalServerError
    switch code {
    case codes.Unauthenticated:
        httpStatus = http.StatusUnauthorized
    case codes.PermissionDenied:
        httpStatus = http.StatusForbidden
    case codes.NotFound:
        httpStatus = http.StatusNotFound
    case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
        httpStatus = http.StatusBadRequest
    case codes.AlreadyExists, codes.Aborted:
        httpStatus = http.StatusConflict
    case codes.ResourceExhausted:
        httpStatus = http.StatusTooManyRequests
    case codes.Unavailable:
        httpStatus = http.StatusServiceUnavailable
    case codes.DeadlineExceeded:
        httpStatus = http.StatusGatewayTimeout
    case codes.Canceled:
        httpStatus = 499
    default:
        httpStatus = http.StatusInternalServerError
    }

    http.Error(w, err.Error(), httpStatus)
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

func (g *APIGateway) withAuth(ctx context.Context, r *http.Request) context.Context {
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	if authz == "" {
		return ctx
	}
	md := metadata.Pairs("authorization", authz)
	return metadata.NewOutgoingContext(ctx, md)
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
    router.HandleFunc("/api/wallets/{id}/transactions", gateway.handleWalletRequest).Methods("GET")
    router.HandleFunc("/api/wallets/{id}/deposit", gateway.handleWalletRequest).Methods("POST")
    router.HandleFunc("/api/wallets/{id}/withdraw", gateway.handleWalletRequest).Methods("POST")
    router.HandleFunc("/api/wallets/user/{user_id}", gateway.handleWalletRequest).Methods("GET")

	// User routes and auth
	router.HandleFunc("/api/users", gateway.handleUserRequest).Methods("POST")
	router.HandleFunc("/api/users/{id}", gateway.handleUserRequest).Methods("GET", "PUT")
	router.HandleFunc("/api/users/wallet/{wallet_address}", gateway.handleUserRequest).Methods("GET")
	router.HandleFunc("/user/nonce", gateway.handleUserRequest).Methods("POST")
	router.HandleFunc("/user/verify", gateway.handleUserRequest).Methods("POST")
	router.HandleFunc("/auth/nonce", gateway.handleUserRequest).Methods("POST")
	router.HandleFunc("/auth/verify", gateway.handleUserRequest).Methods("POST")
	router.HandleFunc("/auth/dev-login", gateway.devLogin).Methods("POST")
	router.HandleFunc("/api/auth/dev-login", gateway.devLogin).Methods("POST")
	router.HandleFunc("/auth/me", gateway.handleUserRequest).Methods("GET")

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
        AllowedOriginValidator: func(r *http.Request, origin string) bool {
            o := strings.ToLower(strings.TrimSpace(origin))
            if o == "" {
                return false
            }
            if strings.HasPrefix(o, "http://localhost") || strings.HasPrefix(o, "https://localhost") {
                return true
            }
            if strings.HasPrefix(o, "http://127.0.0.1") || strings.HasPrefix(o, "https://127.0.0.1") {
                return true
            }
            if strings.HasPrefix(o, "http://0.0.0.0") || strings.HasPrefix(o, "https://0.0.0.0") {
                return true
            }
            return false
        },
	})

	handler := corsMiddleware(router)

	logger.Info("Starting API Gateway on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

func (g *APIGateway) requestNonce(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var body struct {
		Wallet string `json:"wallet"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Wallet) == "" {
		http.Error(w, "wallet required", http.StatusBadRequest)
		return
	}
	req := &wallet.RequestNonceRequest{Wallet: body.Wallet}
	resp, err := g.walletStub.RequestNonce(ctx, req)
	if err != nil {
		g.handleGRPCError(ctx, w, err, "wallet", "RequestNonce")
		return
	}
	g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) verifySignature(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var body struct {
		Wallet    string `json:"wallet"`
		Signature string `json:"signature"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Wallet) == "" || strings.TrimSpace(body.Signature) == "" {
		http.Error(w, "wallet and signature required", http.StatusBadRequest)
		return
	}
	req := &wallet.VerifySignatureRequest{Wallet: body.Wallet, Signature: body.Signature}
	resp, err := g.walletStub.VerifySignature(ctx, req)
	if err != nil {
		g.handleGRPCError(ctx, w, err, "wallet", "VerifySignature")
		return
	}
	g.writeJSONResponse(w, http.StatusOK, resp)
}

// GET /auth/me - returns current user profile using JWT
func (g *APIGateway) me(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	if authz == "" || !strings.HasPrefix(authz, "Bearer ") {
		http.Error(w, "authorization required", http.StatusUnauthorized)
		return
	}
	tokenString := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
	secret := strings.TrimSpace(getEnv("AUTH_JWT_SECRET", "dev-secret"))

	// Validate token and extract wallet claim
	type validator interface{}
	_ = validator(nil)
	claimsWallet, err := parseWalletFromJWT(tokenString, secret)
	if err != nil || claimsWallet == "" {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}
	resp, err := g.walletStub.GetUserByWallet(ctx, &wallet.GetUserByWalletRequest{WalletAddress: claimsWallet})
	if err != nil {
		g.handleGRPCError(ctx, w, err, "wallet", "GetUserByWallet")
		return
	}
	g.writeJSONResponse(w, http.StatusOK, resp)
}

// POST /auth/dev-login { wallet?: string }
// Issues a JWT for development without signature verification
func (g *APIGateway) devLogin(w http.ResponseWriter, r *http.Request) {
    enabled := strings.EqualFold(strings.TrimSpace(getEnv("AUTH_DEV_LOGIN_ENABLED", "true")), "true")
    if !enabled {
        http.Error(w, "dev login disabled", http.StatusForbidden)
        return
    }
    var body struct{ Wallet string `json:"wallet"` }
    _ = json.NewDecoder(r.Body).Decode(&body)
    walletAddr := strings.TrimSpace(body.Wallet)
    if walletAddr == "" {
        walletAddr = strings.TrimSpace(getEnv("AUTH_DEV_LOGIN_WALLET", "0xTESTUSER"))
    }
    if walletAddr == "" {
        http.Error(w, "wallet required", http.StatusBadRequest)
        return
    }

    // Ensure user exists in wallet service by requesting a nonce (creates user if missing)
    ctx := r.Context()
    _, _ = g.walletStub.RequestNonce(ctx, &wallet.RequestNonceRequest{Wallet: walletAddr})

    // Fetch user to get ID
    userResp, err := g.walletStub.GetUserByWallet(ctx, &wallet.GetUserByWalletRequest{WalletAddress: walletAddr})
    if err != nil || userResp == nil || userResp.User == nil {
        // proceed anyway; token issuance below, but account creation may fail
        userResp = &wallet.GetUserResponse{User: &wallet.User{Id: "", WalletAddress: walletAddr}}
    }

    secret := strings.TrimSpace(getEnv("AUTH_JWT_SECRET", "dev-secret"))
    claims := jwt.MapClaims{
        "wallet": walletAddr,
        "exp":    time.Now().Add(24 * time.Hour).Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, err := token.SignedString([]byte(secret))
    if err != nil {
        http.Error(w, "failed to issue token", http.StatusInternalServerError)
        return
    }

    // No wallet account is pre-created here. Frontend will create on-demand.

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}

// Minimal JWT parsing for HS256 to get wallet claim
func parseWalletFromJWT(tokenString string, secret string) (string, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if w, ok := claims["wallet"].(string); ok {
			return w, nil
		}
		return "", fmt.Errorf("wallet claim missing")
	}
	return "", fmt.Errorf("invalid token")
}

func getEnv(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v != "" {
		return v
	}
	return def
}
