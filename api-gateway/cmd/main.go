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
    "golang.org/x/sync/singleflight"
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
    "google.golang.org/protobuf/types/known/timestamppb"
    "github.com/redis/go-redis/v9"
)

type KafkaPublisher interface {
	Publish(ctx context.Context, topic string, key string, value interface{}) error
}

type APIGateway struct {
    logger            *zap.Logger
    marketClient      *resgrpc.ResilientClient
    walletClient      *resgrpc.ResilientClient
    settlementClient  *resgrpc.ResilientClient
    transactionClient *resgrpc.ResilientClient
    kafkaProducer     KafkaPublisher
    marketStub        market.MarketServiceClient
    walletStub        wallet.WalletServiceClient
    settlementStub    settlement.SettlementServiceClient
    transactionStub   transaction.TransactionServiceClient
    redis             *redis.Client
    sfGroup           singleflight.Group
}

func NewAPIGateway(logger *zap.Logger) (*APIGateway, error) {
	marketAddr := getEnv("MARKET_SERVICE_GRPC_ADDR", "market-service:50051")
	walletAddr := getEnv("WALLET_SERVICE_GRPC_ADDR", "wallet-service:50052")
	settlementAddr := getEnv("SETTLEMENT_SERVICE_GRPC_ADDR", "settlement-service:50053")
	transactionAddr := getEnv("TRANSACTION_SERVICE_GRPC_ADDR", "transaction-service:50052")

	marketConfig := resgrpc.DefaultClientConfig("market", marketAddr)
	walletConfig := resgrpc.DefaultClientConfig("wallet", walletAddr)
	settlementConfig := resgrpc.DefaultClientConfig("settlement", settlementAddr)
	transactionConfig := resgrpc.DefaultClientConfig("transaction", transactionAddr)

    marketClient, err := resgrpc.NewResilientClient(marketConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create market client: %w", err)
	}

    walletClient, err := resgrpc.NewResilientClient(walletConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet client: %w", err)
	}

    settlementClient, err := resgrpc.NewResilientClient(settlementConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create settlement client: %w", err)
	}

    transactionClient, err := resgrpc.NewResilientClient(transactionConfig, logger)
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

    // Redis client (for caching)
    redisURL := getEnv("REDIS_URL", "redis://redis:6379")
    ropts, rerr := redis.ParseURL(redisURL)
    if rerr != nil {
        return nil, fmt.Errorf("failed to parse redis url: %w", rerr)
    }
    rcli := redis.NewClient(ropts)
    if err := rcli.Ping(context.Background()).Err(); err != nil {
        logger.Warn("Redis ping failed", zap.Error(err))
    }

    return &APIGateway{
        logger:            logger,
        marketClient:      marketClient,
        walletClient:      walletClient,
        settlementClient:  settlementClient,
        transactionClient: transactionClient,
        kafkaProducer:     kafkaProducer,
        marketStub:        market.NewMarketServiceClient(marketClient.GetConnection()),
        walletStub:        wallet.NewWalletServiceClient(walletClient.GetConnection()),
        settlementStub:    settlement.NewSettlementServiceClient(settlementClient.GetConnection()),
        transactionStub:   transaction.NewTransactionServiceClient(transactionClient.GetConnection()),
        redis:             rcli,
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
    // Pagination params
    q := r.URL.Query()
    page := 1
    size := 20
    if v := strings.TrimSpace(q.Get("page")); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { page = n } }
    if v := strings.TrimSpace(q.Get("page_size")); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { size = n } }
    if size > 50 { size = 50 }

    cacheKey := fmt.Sprintf("markets:list:page:%d:%d", page, size)
    // Attempt cache read
    if g.redis != nil {
        var cached struct{ Markets []*market.Market `json:"markets"` }
        if ok := g.getCacheJSON(ctx, cacheKey, &cached); ok {
            g.writeJSONResponse(w, http.StatusOK, cached)
            return
        }
        _ = cached
    }

    // Singleflight to avoid stampede on cache miss
    v, _, _ := g.sfGroup.Do(cacheKey, func() (interface{}, error) {
        resp, err := g.marketStub.ListMarkets(ctx, &market.ListMarketsRequest{Limit: int32(size), Offset: int32((page-1)*size)})
        if err != nil { return nil, err }
        total := int(resp.Total)
        pageSlice := resp.Markets
        // Conditional GET based on newest updated_at in page
        var newest time.Time
        for _, m := range pageSlice {
            if m != nil && m.UpdatedAt != nil {
                t := m.UpdatedAt.AsTime()
                if t.After(newest) { newest = t }
            }
        }
        if !newest.IsZero() {
            lastMod := newest.UTC().Format(http.TimeFormat)
            etag := fmt.Sprintf("markets-%d-%d-%d", page, size, newest.Unix())
            w.Header().Set("Last-Modified", lastMod)
            w.Header().Set("ETag", etag)
            inm := strings.TrimSpace(r.Header.Get("If-None-Match"))
            ims := strings.TrimSpace(r.Header.Get("If-Modified-Since"))
            if inm != "" && inm == etag {
                w.WriteHeader(http.StatusNotModified)
                return nil, nil
            }
            if ims != "" {
                if t, err := time.Parse(http.TimeFormat, ims); err == nil {
                    if !newest.After(t) {
                        w.WriteHeader(http.StatusNotModified)
                        return nil, nil
                    }
                }
            }
        }
        out := map[string]interface{}{
            "markets": pageSlice,
            "page": page,
            "page_size": size,
            "total": total,
            "next_page": func() int { if page*size < total { return page+1 } ; return 0 }(),
        }
        if g.redis != nil && page <= 10 {
            g.setCacheJSON(ctx, cacheKey, out, g.jitterTTL(30*time.Second))
        }
        return out, nil
    })

    if v == nil {
        g.handleGRPCError(ctx, w, fmt.Errorf("list markets failed"), "market", "ListMarkets")
        return
    }
    g.writeJSONResponse(w, http.StatusOK, v)
}

func (g *APIGateway) getMarket(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    marketID := extractIDFromPath(r.URL.Path, "markets")
    if marketID == "" {
        http.Error(w, "Market ID required", http.StatusBadRequest)
        return
    }
    cacheKey := fmt.Sprintf("market:%s:summary", marketID)
    if g.redis != nil {
        var cached map[string]interface{}
        if ok := g.getCacheJSON(ctx, cacheKey, &cached); ok {
            g.writeJSONResponse(w, http.StatusOK, cached)
            return
        }
        _ = cached
    }
    v, _, _ := g.sfGroup.Do(cacheKey, func() (interface{}, error) {
        resp, err := g.marketStub.GetMarket(ctx, &market.GetMarketRequest{Id: marketID})
        if err != nil { return nil, err }
        if resp != nil && resp.Market != nil && resp.Market.UpdatedAt != nil {
            lastMod := resp.Market.UpdatedAt.AsTime().UTC().Format(http.TimeFormat)
            etag := fmt.Sprintf("%s-%d", resp.Market.Id, resp.Market.UpdatedAt.Seconds)
            w.Header().Set("Last-Modified", lastMod)
            w.Header().Set("ETag", etag)
            if inm := strings.TrimSpace(r.Header.Get("If-None-Match")); inm != "" && inm == etag {
                w.WriteHeader(http.StatusNotModified)
                return nil, nil
            }
            if ims := strings.TrimSpace(r.Header.Get("If-Modified-Since")); ims != "" {
                if t, err := time.Parse(http.TimeFormat, ims); err == nil {
                    if !resp.Market.UpdatedAt.AsTime().After(t) {
                        w.WriteHeader(http.StatusNotModified)
                        return nil, nil
                    }
                }
            }
        }
        out := map[string]interface{}{"market": resp.Market}
        if g.redis != nil {
            ttl := 60 * time.Second
            if resp.Market != nil && strings.EqualFold(resp.Market.Status, "resolved") {
                ttl = 5 * time.Second
            }
            g.setCacheJSON(ctx, cacheKey, out, g.jitterTTL(ttl))
        }
        return out, nil
    })
    if v == nil {
        if w.Header().Get("ETag") != "" || w.Header().Get("Last-Modified") != "" {
            return
        }
        g.handleGRPCError(ctx, w, fmt.Errorf("get market failed"), "market", "GetMarket")
        return
    }
    g.writeJSONResponse(w, http.StatusOK, v)
}

func (g *APIGateway) createMarket(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    g.logger.Info("HTTP request",
        zap.String("method", r.Method),
        zap.String("path", r.URL.Path),
        zap.String("remote", r.RemoteAddr))

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

    g.logger.Info("CreateMarket request",
        zap.String("question", strings.TrimSpace(body.Question)),
        zap.Int("options_count", len(body.Options)))

    resp, err := g.marketStub.CreateMarket(ctx, &req)

    if err != nil {
        g.handleGRPCError(ctx, w, err, "market", "CreateMarket")
        return
    }

    // Removed synchronous fanout to listings/options to reduce load

    out := map[string]interface{}{
        "market":  resp.Market,
    }
    if g.redis != nil {
        // Invalidate listing pages via SCAN
        cursor := uint64(0)
        for {
            keys, cur, _ := g.redis.Scan(ctx, cursor, "markets:list:page:*", 100).Result()
            if len(keys) > 0 { _, _ = g.redis.Del(ctx, keys...).Result() }
            cursor = cur
            if cursor == 0 { break }
        }
    }
    g.logger.Info("CreateMarket success",
        zap.String("market_id", resp.Market.GetId()))
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

    // Invalidate cache entries
    if g.redis != nil {
        _ = g.redis.Del(ctx, fmt.Sprintf("market:%s:summary", marketID)).Err()
        cursor := uint64(0)
        for {
            keys, cur, _ := g.redis.Scan(ctx, cursor, "markets:list:page:*", 100).Result()
            if len(keys) > 0 { _, _ = g.redis.Del(ctx, keys...).Result() }
            cursor = cur
            if cursor == 0 { break }
        }
        _ = g.redis.Del(ctx, fmt.Sprintf("market:%s:options", marketID)).Err()
    }

    g.writeJSONResponse(w, http.StatusOK, resp)
}

func (g *APIGateway) getMarketOptions(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    marketID := extractIDFromPath(r.URL.Path, "markets")
    if marketID == "" {
        http.Error(w, "Market ID required", http.StatusBadRequest)
        return
    }
    cacheKey := fmt.Sprintf("market:%s:options", marketID)
    if g.redis != nil {
        var cached map[string]interface{}
        if ok := g.getCacheJSON(ctx, cacheKey, &cached); ok {
            g.writeJSONResponse(w, http.StatusOK, cached)
            return
        }
        _ = cached
    }
    v, _, _ := g.sfGroup.Do(cacheKey, func() (interface{}, error) {
        resp, err := g.marketStub.GetMarketOptions(ctx, &market.GetMarketOptionsRequest{MarketId: marketID})
        if err != nil { return nil, err }
        out := map[string]interface{}{"options": resp.Options}
        if g.redis != nil { g.setCacheJSON(ctx, cacheKey, out, g.jitterTTL(8*time.Second)) }
        return out, nil
    })
    if v == nil {
        g.handleGRPCError(ctx, w, fmt.Errorf("get market options failed"), "market", "GetMarketOptions")
        return
    }
    g.writeJSONResponse(w, http.StatusOK, v)
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
        if st, ok := status.FromError(err); ok {
            switch st.Code() {
            case codes.InvalidArgument:
                http.Error(w, "Wallet address required", http.StatusBadRequest)
                return
            case codes.NotFound:
                g.writeJSONResponse(w, http.StatusNotFound, map[string]interface{}{"user": nil})
                return
            default:
                g.writeJSONResponse(w, http.StatusNotFound, map[string]interface{}{"user": nil})
                return
            }
        }
        g.writeJSONResponse(w, http.StatusNotFound, map[string]interface{}{"user": nil})
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

    // Invalidate related caches to reflect winner and status changes promptly
    if g.redis != nil && resp != nil && resp.Settlement != nil {
        mid := strings.TrimSpace(resp.Settlement.MarketId)
        if mid != "" {
            _ = g.redis.Del(ctx, fmt.Sprintf("market:%s:summary", mid)).Err()
            _ = g.redis.Del(ctx, fmt.Sprintf("market:%s:options", mid)).Err()
            // Invalidate listing pages via SCAN
            cursor := uint64(0)
            for {
                keys, cur, _ := g.redis.Scan(ctx, cursor, "markets:list:page:*", 100).Result()
                if len(keys) > 0 { _, _ = g.redis.Del(ctx, keys...).Result() }
                cursor = cur
                if cursor == 0 { break }
            }
        }
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
    authz := strings.TrimSpace(r.Header.Get("Authorization"))
    if authz == "" {
        http.Error(w, "authorization required", http.StatusUnauthorized)
        return
    }
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

    // Basic validation
    if body.NumberOfShares <= 0 || body.PricePerShare < 0 {
        http.Error(w, "invalid shares or price_per_share", http.StatusBadRequest)
        return
    }
    // Pre-check funds for BUY transactions
    amount := body.PricePerShare * float64(body.NumberOfShares)
    if txType == transaction.TransactionType_BUY {
        defCur := strings.TrimSpace(getEnv("WALLET_DEFAULT_CURRENCY", "USD"))
        ctx = g.withAuth(ctx, r)
        accResp, err := g.walletStub.GetWalletAccountByUserID(ctx, &wallet.GetWalletAccountByUserIDRequest{UserId: body.UserID, Currency: defCur})
        if err != nil {
            st, ok := status.FromError(err)
            if ok && st.Code() == codes.NotFound {
                http.Error(w, "wallet account not found", http.StatusBadRequest)
                return
            }
            g.handleGRPCError(ctx, w, err, "wallet", "GetWalletAccountByUserID")
            return
        }
        bal := accResp.Account.AvailableBalance
        if amount > bal {
            http.Error(w, "insufficient balance", http.StatusBadRequest)
            return
        }
    }

    req := &transaction.TransactionRequest{
        UserId:          body.UserID,
        MarketId:        body.MarketID,
        OptionId:        body.OptionID,
        TransactionType: txType,
        NumberOfShares:  body.NumberOfShares,
        PricePerShare:   body.PricePerShare,
    }
    ctx = g.withAuth(ctx, r)
    resp, err := g.transactionStub.CreateTransaction(ctx, req)
    if err != nil {
        g.handleGRPCError(ctx, w, err, "transaction", "CreateTransaction")
        return
    }
    // Invalidate market options cache and fetch updated options for immediate UI refresh
    if g.redis != nil {
        _ = g.redis.Del(ctx, fmt.Sprintf("market:%s:options", body.MarketID)).Err()
    }
    opts, oerr := g.marketStub.GetMarketOptions(ctx, &market.GetMarketOptionsRequest{MarketId: body.MarketID})
    if oerr != nil {
        // If options fetch fails, still return transaction response
        g.writeJSONResponse(w, http.StatusCreated, resp)
        return
    }
    // Fetch updated wallet account for immediate balance refresh
    defCur := strings.TrimSpace(getEnv("WALLET_DEFAULT_CURRENCY", "USD"))
    var acctResp *wallet.GetWalletAccountByUserIDResponse
    if strings.TrimSpace(body.UserID) != "" {
        ar, aerr := g.walletStub.GetWalletAccountByUserID(ctx, &wallet.GetWalletAccountByUserIDRequest{UserId: body.UserID, Currency: defCur})
        if aerr == nil { acctResp = ar }
    }
    out := map[string]interface{}{
        "transaction": resp,
        "options":     opts.Options,
    }
    if acctResp != nil {
        out["wallet"] = acctResp.Account
    }
    g.writeJSONResponse(w, http.StatusCreated, out)
}

func (g *APIGateway) getTransactions(ctx context.Context, w http.ResponseWriter, r *http.Request) {
    userID := r.URL.Query().Get("user_id")
    marketID := r.URL.Query().Get("market_id")
    // Pagination
    q := r.URL.Query()
    page := 1
    size := 20
    if v := strings.TrimSpace(q.Get("page")); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { page = n } }
    if v := strings.TrimSpace(q.Get("page_size")); v != "" { if n, err := strconv.Atoi(v); err == nil && n > 0 { size = n } }
    if size > 50 { size = 50 }

    req := &transaction.GetTransactionsRequest{Limit: int32(size), Offset: int32((page-1)*size)}
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
    total := int(resp.GetTotal())
    out := map[string]interface{}{
        "transactions": resp.Transactions,
        "page": page,
        "page_size": size,
        "total": total,
        "next_page": func() int { if page*size < total { return page+1 } ; return 0 }(),
    }
    g.writeJSONResponse(w, http.StatusOK, out)
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

// Cache helpers
func (g *APIGateway) getCacheJSON(ctx context.Context, key string, dest interface{}) bool {
    if g.redis == nil { return false }
    raw, err := g.redis.Get(ctx, key).Bytes()
    if err != nil { return false }
    if err := json.Unmarshal(raw, &dest); err != nil { return false }
    return true
}

func (g *APIGateway) setCacheJSON(ctx context.Context, key string, data interface{}, ttl time.Duration) {
    if g.redis == nil { return }
    b, err := json.Marshal(data)
    if err != nil { return }
    _ = g.redis.Set(ctx, key, b, ttl).Err()
}

func (g *APIGateway) jitterTTL(ttl time.Duration) time.Duration {
    n := ttl
    jitter := time.Duration(int64(n) / 10) // 10%
    if time.Now().UnixNano()%2 == 0 { return ttl + jitter }
    return ttl - jitter
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

    gateway, err := NewAPIGateway(logger)
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
