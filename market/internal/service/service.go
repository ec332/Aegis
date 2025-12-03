package service

import (
    "context"
    "crypto/rand"
    "fmt"
    "strings"
    "time"

    settlement "github.com/aegis/proto/gen/settlement"
    "github.com/ec332/aegis/market/internal/repository"
    "github.com/ec332/aegis/market/pkg/models"
    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

// Service handles business logic for markets
type Service struct {
    repo        *repository.Repository
    redisClient *redis.Client
    logger      *zap.Logger
    settlementClient settlement.SettlementServiceClient
}

// New creates a new service instance
func New(repo *repository.Repository, redisClient *redis.Client, logger *zap.Logger, settlementClient settlement.SettlementServiceClient) *Service {
    return &Service{
        repo:        repo,
        redisClient: redisClient,
        logger:      logger,
        settlementClient: settlementClient,
    }
}

// CreateMarket creates a new market with validation (called by API Gateway)
func (s *Service) CreateMarket(ctx context.Context, req models.CreateMarketRequest) (*models.Market, error) {
	// Validation
	if err := s.validateCreateMarketRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	s.logger.Info("CreateMarket request",
		zap.Any("request", req))

	now := time.Now()
	marketID := uuid.New().String()

    // Create market
    market := &models.Market{
        ID:                 marketID,
        Title:              req.Title,
        Description:        req.Description,
        Status:             models.MarketStatusActive,
        ResolutionDatetime: req.ResolutionDatetime,
        WinningOptionID:    nil,
        CreatedAt:          now,
        UpdatedAt:          now,
    }

	// Create options
	options := make([]models.Option, len(req.Options))
	for i, title := range req.Options {
		options[i] = models.Option{
			ID:        uuid.New().String(),
			MarketID:  marketID,
			Title:     title,
			CreatedAt: now,
		}
	}

	// Create liquidity pools (one per option, initial value 0)
	pools := make([]models.LiquidityPool, len(options))
	for i, option := range options {
		pools[i] = models.LiquidityPool{
			ID:            uuid.New().String(),
			MarketID:      marketID,
			OptionID:      option.ID,
			ShareQuantity: 0,
			UpdatedAt:     now,
		}
	}

	// Save to database
	if err := s.repo.CreateMarket(ctx, market, options, pools); err != nil {
		return nil, fmt.Errorf("create market: %w", err)
	}

	// Add options and pools to response
	market.Options = options
	market.LiquidityPools = pools

    return market, nil
}

// GetMarket retrieves a market by ID
func (s *Service) GetMarket(ctx context.Context, marketID string) (*models.Market, error) {
	market, err := s.repo.GetMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}

	return market, nil
}

// ListMarkets retrieves markets
func (s *Service) ListMarkets(ctx context.Context, status *models.MarketStatus, limit, offset int32) ([]models.Market, int32, error) {
    markets, total, err := s.repo.ListMarkets(ctx, status, limit, offset)
    if err != nil {
        return nil, 0, err
    }
    return markets, total, nil
}

// UpdateMarket updates a market's details
func (s *Service) UpdateMarket(ctx context.Context, marketID string, req models.UpdateMarketRequest) (*models.Market, error) {
	// Validate status transition if status is being updated
	if req.Status != nil {
		market, err := s.repo.GetMarket(ctx, marketID)
		if err != nil {
			return nil, err
		}
		if err := s.validateStatusTransition(market.Status, *req.Status); err != nil {
			return nil, fmt.Errorf("invalid status transition: %w", err)
		}
	}

	// Update the market in the repository
	if err := s.repo.UpdateMarket(ctx, marketID, req); err != nil {
		return nil, fmt.Errorf("update market: %w", err)
	}

	// Fetch the updated market
	market, err := s.repo.GetMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}

    return market, nil
}

// UpdateLiquidityPool updates a liquidity pool and publishes to Redis
func (s *Service) UpdateLiquidityPool(ctx context.Context, marketID, poolID string, poolValue float64) error {
    if err := s.repo.UpdateLiquidityPool(ctx, poolID, poolValue); err != nil {
        return err
    }
    return nil
}

// SubscribeToLiquidityUpdates subscribes to liquidity pool updates for a market from Redis
// removed SubscribeToLiquidityUpdates

// removed publishLiquidityUpdate

// GetMarketPrices calculates the current prices for all options in a market
func (s *Service) GetMarketPrices(ctx context.Context, marketID string) (map[string]float64, error) {
	market, err := s.repo.GetMarket(ctx, marketID)
	if err != nil {
		return nil, err
	}

	// Ensure pools are in same order as options if needed, or just map by ID
	// The repo returns pools ordered by updated_at, which might not match options order.
	// We need to map option IDs to indices or just iterate.
	// Let's map option ID to quantity.
	quantityMap := make(map[string]float64)
	for _, pool := range market.LiquidityPools {
		quantityMap[pool.OptionID] = pool.ShareQuantity
	}

	// Create a slice of quantities corresponding to the market options order
	// Assuming market.Options is populated and ordered
	if len(market.Options) == 0 {
		return nil, fmt.Errorf("market has no options")
	}

	orderedQuantities := make([]float64, len(market.Options))
	for i, option := range market.Options {
		orderedQuantities[i] = quantityMap[option.ID]
	}

	b := market.LiquidityParameter
	if b <= 0 {
		b = 100.0 // fallback
	}
	prices := make(map[string]float64)
	for i, option := range market.Options {
		prices[option.ID] = calculatePrice(orderedQuantities, b, i)
	}

	return prices, nil
}

// CalculateBuyCost calculates the cost to buy a specific amount of shares for an option
func (s *Service) CalculateBuyCost(ctx context.Context, marketID, optionID string, amount float64) (float64, error) {
	market, err := s.repo.GetMarket(ctx, marketID)
	if err != nil {
		return 0, err
	}

	quantityMap := make(map[string]float64)
	for _, pool := range market.LiquidityPools {
		quantityMap[pool.OptionID] = pool.ShareQuantity
	}

	orderedQuantities := make([]float64, len(market.Options))
	optionIndex := -1
	for i, option := range market.Options {
		orderedQuantities[i] = quantityMap[option.ID]
		if option.ID == optionID {
			optionIndex = i
		}
	}

	if optionIndex == -1 {
		return 0, fmt.Errorf("option not found in market")
	}

	b := market.LiquidityParameter
	if b <= 0 {
		b = 100.0 // fallback
	}
	return calculateCostToBuy(orderedQuantities, b, optionIndex, amount), nil
}

// CalculateSellCost calculates the return from selling a specific amount of shares for an option
func (s *Service) CalculateSellCost(ctx context.Context, marketID, optionID string, amount float64) (float64, error) {
	market, err := s.repo.GetMarket(ctx, marketID)
	if err != nil {
		return 0, err
	}

	quantityMap := make(map[string]float64)
	for _, pool := range market.LiquidityPools {
		quantityMap[pool.OptionID] = pool.ShareQuantity
	}

	orderedQuantities := make([]float64, len(market.Options))
	optionIndex := -1
	for i, option := range market.Options {
		orderedQuantities[i] = quantityMap[option.ID]
		if option.ID == optionID {
			optionIndex = i
		}
	}

	if optionIndex == -1 {
		return 0, fmt.Errorf("option not found in market")
	}

	b := market.LiquidityParameter
	if b <= 0 {
		b = 100.0 // fallback
	}
	return calculateCostToSell(orderedQuantities, b, optionIndex, amount), nil
}

// Helper functions

func (s *Service) validateCreateMarketRequest(req models.CreateMarketRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if req.Description == "" {
		return fmt.Errorf("description is required")
	}
	return nil
}

func (s *Service) validateStatusTransition(from, to models.MarketStatus) error {
    // Define valid transitions
    validTransitions := map[models.MarketStatus][]models.MarketStatus{
        models.MarketStatusDraft:     {models.MarketStatusActive, models.MarketStatusHidden},
        models.MarketStatusActive:    {models.MarketStatusHidden, models.MarketStatusResolving, models.MarketStatusResolved},
        models.MarketStatusHidden:    {models.MarketStatusActive, models.MarketStatusDraft},
        models.MarketStatusResolving: {models.MarketStatusResolved},
        models.MarketStatusResolved:  {},
    }

	allowed, exists := validTransitions[from]
	if !exists {
		return fmt.Errorf("unknown status: %s", from)
	}

	for _, allowedStatus := range allowed {
		if allowedStatus == to {
			return nil
		}
	}

	return fmt.Errorf("cannot transition from %s to %s", from, to)
}

func (s *Service) TriggerSettlementsForExpiredMarkets(ctx context.Context, since, until time.Time) error {
    s.logger.Info("cron settlement sweep", zap.Time("since", since), zap.Time("until", until))
    markets, err := s.repo.ListMarketsNeedingResolution(ctx, until, 100)
    if err != nil {
        return err
    }
    s.logger.Info("cron settlement candidates", zap.Int("count", len(markets)))
    for _, m := range markets {
        s.logger.Info("cron resolving market", zap.String("market_id", m.ID))
        var sid string
        var win string
        req := &settlement.CreateSettlementRequest{MarketId: m.ID}
        resp, err := s.settlementClient.CreateSettlement(ctx, req)
        if err == nil && resp != nil && resp.Settlement != nil {
            sid = resp.Settlement.GetId()
            comp, cerr := s.settlementClient.CompleteSettlement(ctx, &settlement.CompleteSettlementRequest{Id: sid})
            if cerr == nil && comp != nil && comp.Settlement != nil {
                win = strings.TrimSpace(comp.Settlement.GetWinningOptionId())
            } else {
                s.logger.Error("complete settlement failed", zap.String("settlement_id", sid), zap.Error(cerr))
            }
        } else {
            s.logger.Error("create settlement failed", zap.String("market_id", m.ID), zap.Error(err))
        }
        if strings.TrimSpace(win) == "" {
            opts, oerr := s.repo.GetOptionsByMarketID(ctx, m.ID)
            if oerr == nil && len(opts) > 0 {
                b := make([]byte, 1)
                if _, rerr := rand.Read(b); rerr == nil {
                    idx := int(b[0]) % len(opts)
                    win = opts[idx].ID
                } else {
                    win = opts[0].ID
                }
                s.logger.Info("assigned fallback winner", zap.String("market_id", m.ID), zap.String("option_id", win))
            } else {
                s.logger.Warn("no options available to assign winner", zap.String("market_id", m.ID), zap.Error(oerr))
            }
        }
        statusResolved := models.MarketStatusResolved
        if strings.TrimSpace(win) != "" {
            if err := s.repo.UpdateMarket(ctx, m.ID, models.UpdateMarketRequest{Status: &statusResolved, WinningOptionID: &win}); err != nil {
                s.logger.Error("failed to persist market winner", zap.String("market_id", m.ID), zap.Error(err))
            }
        } else {
            if err := s.repo.UpdateMarket(ctx, m.ID, models.UpdateMarketRequest{Status: &statusResolved}); err != nil {
                s.logger.Error("failed to mark market resolved", zap.String("market_id", m.ID), zap.Error(err))
            }
        }
        if strings.TrimSpace(sid) != "" {
            _, perr := s.settlementClient.ProcessPayout(ctx, &settlement.ProcessPayoutRequest{SettlementId: sid})
            if perr != nil {
                s.logger.Warn("process payout returned error", zap.String("settlement_id", sid), zap.Error(perr))
            }
        }
        s.logger.Info("market resolved", zap.String("market_id", m.ID), zap.String("settlement_id", sid), zap.String("winning_option_id", win))
    }
    return nil
}
