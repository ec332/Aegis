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
)

// Service handles business logic for markets
type Service struct {
	repo        *repository.Repository
	redisClient *redis.Client
}

// New creates a new service instance
func New(repo *repository.Repository, redisClient *redis.Client) *Service {
	return &Service{
		repo:        repo,
		redisClient: redisClient,
	}
}

// CreateMarket creates a new market with validation (called by API Gateway)
func (s *Service) CreateMarket(ctx context.Context, req models.CreateMarketRequest) (*models.Market, error) {
	// Validation
	if err := s.validateCreateMarketRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

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
			ID:        uuid.New().String(),
			MarketID:  marketID,
			OptionID:  option.ID,
			PoolValue: 0,
			UpdatedAt: now,
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
		fmt.Printf("Warning: failed to publish market creation: %v\n", err)
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
		fmt.Printf("Warning: failed to publish market update: %v\n", err)
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
		fmt.Printf("Warning: failed to publish liquidity update: %v\n", err)
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
					fmt.Printf("Error unmarshaling liquidity update: %v\n", err)
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

// Helper functions

func (s *Service) validateCreateMarketRequest(req models.CreateMarketRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if req.Description == "" {
		return fmt.Errorf("description is required")
	}
	if len(req.Options) < 2 {
		return fmt.Errorf("at least 2 options are required")
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
