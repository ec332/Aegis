package service

import (
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"settlement-service/internal/repository"
	"settlement-service/pkg/models"
)

type SettlementService struct {
	repo *repository.Repository
}

func NewSettlementService(repo *repository.Repository) *SettlementService {
	return &SettlementService{repo: repo}
}

func (s *SettlementService) CreateSettlement(req models.SettlementRequest) (*models.SettlementResponse, error) {
	// Check if settlement already exists for this market
	existing, err := s.repo.GetSettlementByMarketID(req.MarketID)
	if err == nil && existing != nil {
		return &models.SettlementResponse{
			Settlement: existing,
			Message:    "Settlement already exists for this market",
		}, nil
	}

	// Get market transactions to calculate pools
	transactions, err := s.repo.GetMarketTransactions(req.MarketID)
	if err != nil {
		return nil, fmt.Errorf("failed to get market transactions: %v", err)
	}

	// Calculate total pool and winning pool
	totalPool := 0.0
	winningPool := 0.0
	
	for _, tx := range transactions {
		totalPool += tx.Amount
		if tx.OptionID == req.WinningOptionID {
			winningPool += tx.Amount
		}
	}

	// Create settlement
	settlement := &models.Settlement{
		ID:              uuid.New().String(),
		MarketID:        req.MarketID,
		WinningOptionID: req.WinningOptionID,
		TotalPool:       totalPool,
		WinningPool:     winningPool,
		Status:          models.SettlementStatusPending,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.repo.CreateSettlement(settlement); err != nil {
		return nil, fmt.Errorf("failed to create settlement: %v", err)
	}

	// Process distributions
	if err := s.processDistributions(settlement, transactions); err != nil {
		return nil, fmt.Errorf("failed to process distributions: %v", err)
	}

	return &models.SettlementResponse{
		Settlement: settlement,
		Message:    "Settlement created successfully",
	}, nil
}

func (s *SettlementService) GetSettlement(id string) (*models.Settlement, error) {
	return s.repo.GetSettlement(id)
}

func (s *SettlementService) GetSettlementByMarketID(marketID string) (*models.Settlement, error) {
	return s.repo.GetSettlementByMarketID(marketID)
}

func (s *SettlementService) CompleteSettlement(id string) error {
	settlement, err := s.repo.GetSettlement(id)
	if err != nil {
		return fmt.Errorf("settlement not found: %v", err)
	}

	if settlement.Status != models.SettlementStatusPending {
		return fmt.Errorf("settlement cannot be completed in current status: %s", settlement.Status)
	}

	// Get all distributions for this settlement
	distributions, err := s.repo.GetSettlementDistributions(id)
	if err != nil {
		return fmt.Errorf("failed to get distributions: %v", err)
	}

	// Process each distribution (in real system, this would trigger wallet service calls)
	for _, distribution := range distributions {
		if err := s.processDistribution(distribution); err != nil {
			log.Printf("Failed to process distribution %s: %v", distribution.ID, err)
			continue
		}
	}

	// Update settlement status
	if err := s.repo.UpdateSettlementStatus(id, models.SettlementStatusCompleted); err != nil {
		return fmt.Errorf("failed to update settlement status: %v", err)
	}

	return nil
}

func (s *SettlementService) processDistributions(settlement *models.Settlement, transactions []repository.MarketTransaction) error {
	// Group transactions by user
	userTransactions := make(map[string][]repository.MarketTransaction)
	for _, tx := range transactions {
		userTransactions[tx.UserID] = append(userTransactions[tx.UserID], tx)
	}

	// Calculate distribution for each user
	for userID, userTxs := range userTransactions {
		userWinningAmount := 0.0
		totalUserAmount := 0.0

		for _, tx := range userTxs {
			totalUserAmount += tx.Amount
			if tx.OptionID == settlement.WinningOptionID {
				userWinningAmount += tx.Amount
			}
		}

		// Calculate payout: user's winning amount / total winning pool * total pool
		if userWinningAmount > 0 && settlement.WinningPool > 0 {
			payoutAmount := (userWinningAmount / settlement.WinningPool) * settlement.TotalPool

			distribution := &models.SettlementDistribution{
				ID:           uuid.New().String(),
				SettlementID: settlement.ID,
				UserID:       userID,
				Amount:       payoutAmount,
				Status:       "pending",
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}

			if err := s.repo.CreateSettlementDistribution(distribution); err != nil {
				return fmt.Errorf("failed to create distribution for user %s: %v", userID, err)
			}
		}
	}

	return nil
}

func (s *SettlementService) processDistribution(distribution *models.SettlementDistribution) error {
	// In a real system, this would call the wallet service to credit the user's wallet
	log.Printf("Processing distribution %s: User %s should receive %.8f", 
		distribution.ID, distribution.UserID, distribution.Amount)

	// Update distribution status
	if err := s.repo.UpdateSettlementDistributionStatus(distribution.ID, "completed"); err != nil {
		return fmt.Errorf("failed to update distribution status: %v", err)
	}

	return nil
}