package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
}

// New creates a new service instance
func New(repo *repository.Repository, redisClient *redis.Client, logger *zap.Logger) *Service {
	return &Service{
		repo:        repo,
		redisClient: redisClient,
		logger:      logger,
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
        Status:             models.MarketStatusDraft,
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

	// Publish market creation event to Redis
	if err := s.publishLiquidityUpdate(ctx, marketID, pools); err != nil {
		s.logger.Warn("failed to publish market creation",
			zap.Error(err))
	}

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
func (s *Service) ListMarkets(ctx context.Context, status *models.MarketStatus) ([]models.Market, error) {
	markets, err := s.repo.ListMarkets(ctx, status)
	if err != nil {
		return nil, err
	}

	return markets, nil
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

	// Publish update to Redis
	if err := s.publishLiquidityUpdate(ctx, marketID, market.LiquidityPools); err != nil {
		s.logger.Warn("failed to publish market update",
			zap.Error(err))
	}

	return market, nil
}

// UpdateLiquidityPool updates a liquidity pool and publishes to Redis
func (s *Service) UpdateLiquidityPool(ctx context.Context, marketID, poolID string, poolValue float64) error {
	if err := s.repo.UpdateLiquidityPool(ctx, poolID, poolValue); err != nil {
		return err
	}

	// Fetch updated pools
	pools, err := s.repo.GetLiquidityPoolsByMarketID(ctx, marketID)
	if err != nil {
		return err
	}

	// Publish to Redis
	if err := s.publishLiquidityUpdate(ctx, marketID, pools); err != nil {
		s.logger.Warn("failed to publish liquidity update",
			zap.Error(err))
	}

	return nil
}

// SubscribeToLiquidityUpdates subscribes to liquidity pool updates for a market from Redis
func (s *Service) SubscribeToLiquidityUpdates(ctx context.Context, marketID string) (<-chan models.LiquidityUpdate, error) {
	pubsub := s.redisClient.Subscribe(ctx, fmt.Sprintf("market:%s:liquidity", marketID))

	ch := make(chan models.LiquidityUpdate)

	go func() {
		defer close(ch)
		defer pubsub.Close()

		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-pubsub.Channel():
				var update models.LiquidityUpdate
				if err := json.Unmarshal([]byte(msg.Payload), &update); err != nil {
					s.logger.Error("failed to unmarshal liquidity update",
						zap.Error(err))
					continue
				}
				ch <- update
			}
		}
	}()

	return ch, nil
}

func (s *Service) publishLiquidityUpdate(ctx context.Context, marketID string, pools []models.LiquidityPool) error {
	update := models.LiquidityUpdate{
		MarketID:       marketID,
		LiquidityPools: pools,
		Timestamp:      time.Now(),
	}

	data, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("marshal liquidity update: %w", err)
	}

	channel := fmt.Sprintf("market:%s:liquidity", marketID)
	if err := s.redisClient.Publish(ctx, channel, data).Err(); err != nil {
		return fmt.Errorf("publish to redis: %w", err)
	}

	return nil
}

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
		models.MarketStatusActive:    {models.MarketStatusHidden, models.MarketStatusResolving},
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

// CreateUser creates a new user
func (s *Service) CreateUser(ctx context.Context, req models.CreateUserRequest) (*models.User, error) {
	if req.WalletAddress == "" {
		return nil, fmt.Errorf("wallet_address is required")
	}
	if req.Balance < 0 {
		return nil, fmt.Errorf("balance cannot be negative")
	}

	now := time.Now()
	userID := uuid.New().String()
	nonce := uuid.New().String() // Generate a random nonce

	user := &models.User{
		ID:            userID,
		WalletAddress: req.WalletAddress,
		Balance:       req.Balance,
		Nonce:         nonce,
		Role:          models.UserRoleUser, // Default to user role
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

// GetUser retrieves a user by ID
func (s *Service) GetUser(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

// GetUserByWalletAddress retrieves a user by wallet address
func (s *Service) GetUserByWalletAddress(ctx context.Context, walletAddress string) (*models.User, error) {
	user, err := s.repo.GetUserByWalletAddress(ctx, walletAddress)
	if err != nil {
		return nil, fmt.Errorf("get user by wallet: %w", err)
	}
	return user, nil
}

// UpdateUser updates a user and returns the updated user
func (s *Service) UpdateUser(ctx context.Context, userID string, req models.UpdateUserRequest) (*models.User, error) {
	if err := s.repo.UpdateUser(ctx, userID, req); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	// Return the updated user
	return s.GetUser(ctx, userID)
}
