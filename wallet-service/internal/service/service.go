package service

import (
	"context"
	"fmt"
	"time"

	"wallet-service/internal/repository"
	"wallet-service/pkg/models"
	"github.com/google/uuid"
)

// Service handles business logic for wallet operations
type Service struct {
	repo *repository.Repository
}

// New creates a new service instance
func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

// CreateWalletAccount creates a new wallet account for a user
func (s *Service) CreateWalletAccount(ctx context.Context, req models.CreateWalletAccountRequest) (*models.WalletAccount, error) {
	if req.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if req.Currency == "" {
		req.Currency = models.CurrencyUSDC // Default to USDC
	}

	// Check if wallet account already exists
	existingAccount, err := s.repo.GetWalletAccountByUserID(ctx, req.UserID, req.Currency)
	if err == nil && existingAccount != nil {
		return nil, fmt.Errorf("wallet account already exists for user %s with currency %s", req.UserID, req.Currency)
	}

	now := time.Now()
	accountID := uuid.New().String()

	account := &models.WalletAccount{
		ID:        accountID,
		UserID:    req.UserID,
		Currency:  req.Currency,
		Balance:   0,
		Available: 0,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.CreateWalletAccount(ctx, account); err != nil {
		return nil, fmt.Errorf("create wallet account: %w", err)
	}

	return account, nil
}

// GetWalletAccount retrieves a wallet account by ID
func (s *Service) GetWalletAccount(ctx context.Context, accountID string) (*models.WalletAccount, error) {
	account, err := s.repo.GetWalletAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("get wallet account: %w", err)
	}
	return account, nil
}

// GetWalletAccountByUserID retrieves a wallet account by user ID and currency
func (s *Service) GetWalletAccountByUserID(ctx context.Context, userID string, currency models.Currency) (*models.WalletAccount, error) {
	account, err := s.repo.GetWalletAccountByUserID(ctx, userID, currency)
	if err != nil {
		return nil, fmt.Errorf("get wallet account by user: %w", err)
	}
	return account, nil
}

// Deposit handles depositing funds to a wallet account
func (s *Service) Deposit(ctx context.Context, accountID string, req models.DepositRequest) (*models.WalletTransaction, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("deposit amount must be positive")
	}

	// Get the wallet account
	account, err := s.repo.GetWalletAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("get wallet account: %w", err)
	}

	// Create transaction
	transaction := &models.WalletTransaction{
		ID:        uuid.New().String(),
		WalletID:  accountID,
		MarketID:  nil,
		Type:      models.WalletTransactionTypeDeposit,
		Amount:    req.Amount,
		Status:    models.WalletTransactionStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateWalletTransaction(ctx, transaction); err != nil {
		return nil, fmt.Errorf("create wallet transaction: %w", err)
	}

	// Update account balance
	newBalance := account.Balance + req.Amount
	newAvailable := account.Available + req.Amount
	if err := s.repo.UpdateWalletBalance(ctx, accountID, newBalance, newAvailable); err != nil {
		return nil, fmt.Errorf("update wallet balance: %w", err)
	}

	// Update transaction status to completed
	if err := s.repo.UpdateTransactionStatus(ctx, transaction.ID, models.WalletTransactionStatusCompleted); err != nil {
		return nil, fmt.Errorf("update transaction status: %w", err)
	}

	transaction.Status = models.WalletTransactionStatusCompleted
	return transaction, nil
}

// Withdrawal handles withdrawing funds from a wallet account
func (s *Service) Withdrawal(ctx context.Context, accountID string, req models.WithdrawalRequest) (*models.WalletTransaction, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("withdrawal amount must be positive")
	}

	// Get the wallet account
	account, err := s.repo.GetWalletAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("get wallet account: %w", err)
	}

	// Check if sufficient balance
	if account.Available < req.Amount {
		return nil, fmt.Errorf("insufficient available balance")
	}

	// Create transaction
	transaction := &models.WalletTransaction{
		ID:        uuid.New().String(),
		WalletID:  accountID,
		MarketID:  nil,
		Type:      models.WalletTransactionTypeWithdrawal,
		Amount:    req.Amount,
		Status:    models.WalletTransactionStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateWalletTransaction(ctx, transaction); err != nil {
		return nil, fmt.Errorf("create wallet transaction: %w", err)
	}

	// Update account balance
	newBalance := account.Balance - req.Amount
	newAvailable := account.Available - req.Amount
	if err := s.repo.UpdateWalletBalance(ctx, accountID, newBalance, newAvailable); err != nil {
		return nil, fmt.Errorf("update wallet balance: %w", err)
	}

	// Update transaction status to completed
	if err := s.repo.UpdateTransactionStatus(ctx, transaction.ID, models.WalletTransactionStatusCompleted); err != nil {
		return nil, fmt.Errorf("update transaction status: %w", err)
	}

	transaction.Status = models.WalletTransactionStatusCompleted
	return transaction, nil
}

// DebitWallet debits a wallet account for a market transaction
func (s *Service) DebitWallet(ctx context.Context, accountID string, req models.DebitWalletRequest) (*models.WalletTransaction, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("debit amount must be positive")
	}

	// Get the wallet account
	account, err := s.repo.GetWalletAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("get wallet account: %w", err)
	}

	// Check if sufficient available balance
	if account.Available < req.Amount {
		return nil, fmt.Errorf("insufficient available balance")
	}

	// Create transaction
	transaction := &models.WalletTransaction{
		ID:        uuid.New().String(),
		WalletID:  accountID,
		MarketID:  &req.MarketID,
		Type:      models.WalletTransactionTypeDebit,
		Amount:    req.Amount,
		Status:    models.WalletTransactionStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateWalletTransaction(ctx, transaction); err != nil {
		return nil, fmt.Errorf("create wallet transaction: %w", err)
	}

	// Update account balance (reduce available, keep balance until settlement)
	newAvailable := account.Available - req.Amount
	if err := s.repo.UpdateWalletBalance(ctx, accountID, account.Balance, newAvailable); err != nil {
		return nil, fmt.Errorf("update wallet balance: %w", err)
	}

	// Update transaction status to completed
	if err := s.repo.UpdateTransactionStatus(ctx, transaction.ID, models.WalletTransactionStatusCompleted); err != nil {
		return nil, fmt.Errorf("update transaction status: %w", err)
	}

	transaction.Status = models.WalletTransactionStatusCompleted
	return transaction, nil
}

// CreditWallet credits a wallet account for a market settlement
func (s *Service) CreditWallet(ctx context.Context, accountID string, req models.CreditWalletRequest) (*models.WalletTransaction, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("credit amount must be positive")
	}

	// Get the wallet account
	account, err := s.repo.GetWalletAccount(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("get wallet account: %w", err)
	}

	// Create transaction
	transaction := &models.WalletTransaction{
		ID:        uuid.New().String(),
		WalletID:  accountID,
		MarketID:  &req.MarketID,
		Type:      models.WalletTransactionTypeSettlement,
		Amount:    req.Amount,
		Status:    models.WalletTransactionStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.repo.CreateWalletTransaction(ctx, transaction); err != nil {
		return nil, fmt.Errorf("create wallet transaction: %w", err)
	}

	// Update account balance (increase both balance and available)
	newBalance := account.Balance + req.Amount
	newAvailable := account.Available + req.Amount
	if err := s.repo.UpdateWalletBalance(ctx, accountID, newBalance, newAvailable); err != nil {
		return nil, fmt.Errorf("update wallet balance: %w", err)
	}

	// Update transaction status to completed
	if err := s.repo.UpdateTransactionStatus(ctx, transaction.ID, models.WalletTransactionStatusCompleted); err != nil {
		return nil, fmt.Errorf("update transaction status: %w", err)
	}

	transaction.Status = models.WalletTransactionStatusCompleted
	return transaction, nil
}

// GetWalletTransactions retrieves wallet transactions by wallet ID
func (s *Service) GetWalletTransactions(ctx context.Context, walletID string) ([]models.WalletTransaction, error) {
	transactions, err := s.repo.GetWalletTransactions(ctx, walletID)
	if err != nil {
		return nil, fmt.Errorf("get wallet transactions: %w", err)
	}
	return transactions, nil
}