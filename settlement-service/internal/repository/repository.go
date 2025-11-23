package repository

import (
	"database/sql"
	"fmt"
	"time"

	"settlement-service/pkg/models"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateSettlement(settlement *models.Settlement) error {
	query := `
		INSERT INTO settlements (id, market_id, winning_option_id, total_pool, winning_pool, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(query, settlement.ID, settlement.MarketID, settlement.WinningOptionID, 
		settlement.TotalPool, settlement.WinningPool, settlement.Status, settlement.CreatedAt, settlement.UpdatedAt)
	return err
}

func (r *Repository) GetSettlement(id string) (*models.Settlement, error) {
	query := `
		SELECT id, market_id, winning_option_id, total_pool, winning_pool, status, settled_at, created_at, updated_at
		FROM settlements WHERE id = $1
	`
	
	settlement := &models.Settlement{}
	err := r.db.QueryRow(query, id).Scan(
		&settlement.ID, &settlement.MarketID, &settlement.WinningOptionID,
		&settlement.TotalPool, &settlement.WinningPool, &settlement.Status,
		&settlement.SettledAt, &settlement.CreatedAt, &settlement.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("settlement not found")
	}
	return settlement, err
}

func (r *Repository) GetSettlementByMarketID(marketID string) (*models.Settlement, error) {
	query := `
		SELECT id, market_id, winning_option_id, total_pool, winning_pool, status, settled_at, created_at, updated_at
		FROM settlements WHERE market_id = $1
	`
	
	settlement := &models.Settlement{}
	err := r.db.QueryRow(query, marketID).Scan(
		&settlement.ID, &settlement.MarketID, &settlement.WinningOptionID,
		&settlement.TotalPool, &settlement.WinningPool, &settlement.Status,
		&settlement.SettledAt, &settlement.CreatedAt, &settlement.UpdatedAt,
	)
	
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("settlement not found")
	}
	return settlement, err
}

func (r *Repository) UpdateSettlementStatus(id string, status models.SettlementStatus) error {
	query := `
		UPDATE settlements 
		SET status = $1, settled_at = $2, updated_at = $3
		WHERE id = $4
	`
	now := time.Now()
	settledAt := &now
	if status != models.SettlementStatusCompleted {
		settledAt = nil
	}
	
	_, err := r.db.Exec(query, status, settledAt, now, id)
	return err
}

func (r *Repository) CreateSettlementDistribution(distribution *models.SettlementDistribution) error {
	query := `
		INSERT INTO settlement_distributions (id, settlement_id, user_id, amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(query, distribution.ID, distribution.SettlementID, distribution.UserID,
		distribution.Amount, distribution.Status, distribution.CreatedAt, distribution.UpdatedAt)
	return err
}

func (r *Repository) GetSettlementDistributions(settlementID string) ([]*models.SettlementDistribution, error) {
	query := `
		SELECT id, settlement_id, user_id, amount, status, created_at, updated_at
		FROM settlement_distributions WHERE settlement_id = $1
	`
	
	rows, err := r.db.Query(query, settlementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var distributions []*models.SettlementDistribution
	for rows.Next() {
		distribution := &models.SettlementDistribution{}
		err := rows.Scan(&distribution.ID, &distribution.SettlementID, &distribution.UserID,
			&distribution.Amount, &distribution.Status, &distribution.CreatedAt, &distribution.UpdatedAt)
		if err != nil {
			return nil, err
		}
		distributions = append(distributions, distribution)
	}
	
	return distributions, rows.Err()
}

func (r *Repository) UpdateSettlementDistributionStatus(id string, status string) error {
	query := `
		UPDATE settlement_distributions 
		SET status = $1, updated_at = $2
		WHERE id = $3
	`
	_, err := r.db.Exec(query, status, time.Now(), id)
	return err
}

func (r *Repository) GetMarketTransactions(marketID string) ([]MarketTransaction, error) {
	query := `
		SELECT user_id, option_id, amount, price
		FROM transactions 
		WHERE market_id = $1
		ORDER BY timestamp DESC
	`
	
	rows, err := r.db.Query(query, marketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var transactions []MarketTransaction
	for rows.Next() {
		var tx MarketTransaction
		err := rows.Scan(&tx.UserID, &tx.OptionID, &tx.Amount, &tx.Price)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}
	
	return transactions, rows.Err()
}

type MarketTransaction struct {
	UserID   string
	OptionID string
	Amount   float64
	Price    float64
}