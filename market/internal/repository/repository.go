package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"github.com/ec332/aegis/market/pkg/models"
	_ "github.com/lib/pq"
)

// Repository handles database operations
type Repository struct {
	db *sql.DB
}

// New creates a new repository instance
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateMarket creates a new market with options and liquidity pools in a transaction
func (r *Repository) CreateMarket(ctx context.Context, market *models.Market, options []models.Option, pools []models.LiquidityPool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert market
	query := `
		INSERT INTO markets (id, title, description, status, resolution_datetime, winning_option_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = tx.ExecContext(ctx, query,
		market.ID, market.Title, market.Description, market.Status,
		market.ResolutionDatetime, market.WinningOptionID,
		market.CreatedAt, market.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert market: %w", err)
	}

	// Insert options
	optionQuery := `
		INSERT INTO options (id, market_id, title, created_at)
		VALUES ($1, $2, $3, $4)
	`
	for _, option := range options {
		_, err = tx.ExecContext(ctx, optionQuery,
			option.ID, option.MarketID, option.Title, option.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert option: %w", err)
		}
	}

	// Insert liquidity pools (one per option)
	poolQuery := `
		INSERT INTO liquidity_pool (id, market_id, option_id, pool_value, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	for _, pool := range pools {
		_, err = tx.ExecContext(ctx, poolQuery,
			pool.ID, pool.MarketID, pool.OptionID, pool.PoolValue, pool.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert liquidity pool: %w", err)
		}
	}

	return tx.Commit()
}


// GetMarket retrieves a market by ID with its options and liquidity pools
func (r *Repository) GetMarket(ctx context.Context, marketID string) (*models.Market, error) {
	market := &models.Market{}
	query := `
		SELECT id, title, description, status, resolution_datetime, 
		       winning_option_id, created_at, updated_at
		FROM markets
		WHERE id = $1
	`
	err := r.db.QueryRowContext(ctx, query, marketID).Scan(
		&market.ID, &market.Title, &market.Description, &market.Status,
		&market.ResolutionDatetime, &market.WinningOptionID,
		&market.CreatedAt, &market.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("market not found")
		}
		return nil, fmt.Errorf("query market: %w", err)
	}

	// Get options
	options, err := r.GetOptionsByMarketID(ctx, marketID)
	if err != nil {
		return nil, fmt.Errorf("get options: %w", err)
	}
	market.Options = options

	// Get liquidity pools
	pools, err := r.GetLiquidityPoolsByMarketID(ctx, marketID)
	if err != nil {
		return nil, fmt.Errorf("get liquidity pools: %w", err)
	}
	market.LiquidityPools = pools

	return market, nil
}

// ListMarkets retrieves markets based on filter criteria
func (r *Repository) ListMarkets(ctx context.Context, status *models.MarketStatus) ([]models.Market, error) {
	query := `
		SELECT id, title, description, status, resolution_datetime,
		       winning_option_id, created_at, updated_at
		FROM markets
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 1

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *status)
		argCount++
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query markets: %w", err)
	}
	defer rows.Close()

	markets := []models.Market{}
	for rows.Next() {
		market := models.Market{}
		err := rows.Scan(
			&market.ID, &market.Title, &market.Description, &market.Status,
			&market.ResolutionDatetime, &market.WinningOptionID,
			&market.CreatedAt, &market.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan market: %w", err)
		}
		markets = append(markets, market)
	}

	// Fetch options and liquidity pools for each market
	for i := range markets {
		options, err := r.GetOptionsByMarketID(ctx, markets[i].ID)
		if err != nil {
			return nil, fmt.Errorf("get options for market %s: %w", markets[i].ID, err)
		}
		markets[i].Options = options

		pools, err := r.GetLiquidityPoolsByMarketID(ctx, markets[i].ID)
		if err != nil {
			return nil, fmt.Errorf("get liquidity pools for market %s: %w", markets[i].ID, err)
		}
		markets[i].LiquidityPools = pools
	}

	return markets, nil
}

// UpdateMarket updates market fields
func (r *Repository) UpdateMarket(ctx context.Context, marketID string, updates models.UpdateMarketRequest) error {
	query := "UPDATE markets SET updated_at = $1"
	args := []interface{}{time.Now()}
	argCount := 2

	if updates.Status != nil {
		query += fmt.Sprintf(", status = $%d", argCount)
		args = append(args, *updates.Status)
		argCount++
	}
	if updates.WinningOptionID != nil {
		query += fmt.Sprintf(", winning_option_id = $%d", argCount)
		args = append(args, *updates.WinningOptionID)
		argCount++
	}
	if updates.ResolutionDatetime != nil {
		query += fmt.Sprintf(", resolution_datetime = $%d", argCount)
		args = append(args, updates.ResolutionDatetime)
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, marketID)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update market: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("market not found")
	}

	return nil
}

// GetOptionsByMarketID retrieves all options for a market
func (r *Repository) GetOptionsByMarketID(ctx context.Context, marketID string) ([]models.Option, error) {
	query := `
		SELECT id, market_id, title, created_at
		FROM options
		WHERE market_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.db.QueryContext(ctx, query, marketID)
	if err != nil {
		return nil, fmt.Errorf("query options: %w", err)
	}
	defer rows.Close()

	options := []models.Option{}
	for rows.Next() {
		option := models.Option{}
		err := rows.Scan(
			&option.ID, &option.MarketID, &option.Title, &option.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan option: %w", err)
		}
		options = append(options, option)
	}

	return options, nil
}

// GetLiquidityPoolsByMarketID retrieves all liquidity pools for a market
func (r *Repository) GetLiquidityPoolsByMarketID(ctx context.Context, marketID string) ([]models.LiquidityPool, error) {
	query := `
		SELECT id, market_id, option_id, pool_value, updated_at
		FROM liquidity_pool
		WHERE market_id = $1
		ORDER BY updated_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, marketID)
	if err != nil {
		return nil, fmt.Errorf("query liquidity pools: %w", err)
	}
	defer rows.Close()

	pools := []models.LiquidityPool{}
	for rows.Next() {
		pool := models.LiquidityPool{}
		err := rows.Scan(
			&pool.ID, &pool.MarketID, &pool.OptionID, &pool.PoolValue, &pool.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan liquidity pool: %w", err)
		}
		pools = append(pools, pool)
	}

	return pools, nil
}

// UpdateLiquidityPool updates a liquidity pool value
func (r *Repository) UpdateLiquidityPool(ctx context.Context, poolID string, poolValue float64) error {
	query := `
		UPDATE liquidity_pool
		SET pool_value = $1, updated_at = $2
		WHERE id = $3
	`
	result, err := r.db.ExecContext(ctx, query, poolValue, time.Now(), poolID)
	if err != nil {
		return fmt.Errorf("update liquidity pool: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("liquidity pool not found")
	}

	return nil
}

// InitSchema initializes the database schema
func (r *Repository) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS markets (
		id UUID PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		description TEXT NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'draft',
		resolution_datetime TIMESTAMP,
		winning_option_id UUID,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS options (
		id UUID PRIMARY KEY,
		market_id UUID NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
		title VARCHAR(255) NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_options_market_id ON options(market_id);

	CREATE TABLE IF NOT EXISTS liquidity_pool (
		id UUID PRIMARY KEY,
		market_id UUID NOT NULL REFERENCES markets(id) ON DELETE CASCADE,
		option_id UUID NOT NULL REFERENCES options(id) ON DELETE CASCADE,
		pool_value DECIMAL(20, 8) NOT NULL DEFAULT 0,
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_liquidity_pool_market_id ON liquidity_pool(market_id);
	CREATE INDEX IF NOT EXISTS idx_liquidity_pool_option_id ON liquidity_pool(option_id);
	CREATE INDEX IF NOT EXISTS idx_markets_status ON markets(status);
	CREATE INDEX IF NOT EXISTS idx_markets_created_at ON markets(created_at);
	`

	_, err := r.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("init schema: %w", err)
	}

	return nil
}
