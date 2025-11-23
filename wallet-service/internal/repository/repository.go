package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"wallet-service/pkg/models"
)

// Repository handles database operations for wallet service
type Repository struct {
	db *sql.DB
}

// New creates a new repository instance
func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// CreateWalletAccount creates a new wallet account for a user
func (r *Repository) CreateWalletAccount(ctx context.Context, account *models.WalletAccount) error {
	query := `
		INSERT INTO wallet_accounts (id, user_id, currency, balance, available, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.ExecContext(ctx, query,
		account.ID, account.UserID, account.Currency, account.Balance, account.Available,
		account.CreatedAt, account.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert wallet account: %w", err)
	}
	return nil
}

// GetWalletAccount retrieves a wallet account by ID
func (r *Repository) GetWalletAccount(ctx context.Context, accountID string) (*models.WalletAccount, error) {
	account := &models.WalletAccount{}
	query := `
		SELECT id, user_id, currency, balance, available, created_at, updated_at
		FROM wallet_accounts
		WHERE id = $1
	`
	err := r.db.QueryRowContext(ctx, query, accountID).Scan(
		&account.ID, &account.UserID, &account.Currency, &account.Balance, &account.Available,
		&account.CreatedAt, &account.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("wallet account not found")
		}
		return nil, fmt.Errorf("query wallet account: %w", err)
	}
	return account, nil
}

// GetWalletAccountByUserID retrieves a wallet account by user ID and currency
func (r *Repository) GetWalletAccountByUserID(ctx context.Context, userID string, currency models.Currency) (*models.WalletAccount, error) {
	account := &models.WalletAccount{}
	query := `
		SELECT id, user_id, currency, balance, available, created_at, updated_at
		FROM wallet_accounts
		WHERE user_id = $1 AND currency = $2
	`
	err := r.db.QueryRowContext(ctx, query, userID, currency).Scan(
		&account.ID, &account.UserID, &account.Currency, &account.Balance, &account.Available,
		&account.CreatedAt, &account.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("wallet account not found")
		}
		return nil, fmt.Errorf("query wallet account by user: %w", err)
	}
	return account, nil
}

// UpdateWalletBalance updates the balance and available amount of a wallet account
func (r *Repository) UpdateWalletBalance(ctx context.Context, accountID string, balance, available float64) error {
	query := `
		UPDATE wallet_accounts
		SET balance = $1, available = $2, updated_at = $3
		WHERE id = $4
	`
	result, err := r.db.ExecContext(ctx, query, balance, available, time.Now(), accountID)
	if err != nil {
		return fmt.Errorf("update wallet balance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("wallet account not found")
	}

	return nil
}

// CreateWalletTransaction creates a new wallet transaction
func (r *Repository) CreateWalletTransaction(ctx context.Context, transaction *models.WalletTransaction) error {
	query := `
		INSERT INTO wallet_transactions (id, wallet_id, market_id, type, amount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.ExecContext(ctx, query,
		transaction.ID, transaction.WalletID, transaction.MarketID, transaction.Type, transaction.Amount, transaction.Status,
		transaction.CreatedAt, transaction.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert wallet transaction: %w", err)
	}
	return nil
}

// GetWalletTransactions retrieves wallet transactions by wallet ID
func (r *Repository) GetWalletTransactions(ctx context.Context, walletID string) ([]models.WalletTransaction, error) {
	query := `
		SELECT id, wallet_id, market_id, type, amount, status, created_at, updated_at
		FROM wallet_transactions
		WHERE wallet_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, walletID)
	if err != nil {
		return nil, fmt.Errorf("query wallet transactions: %w", err)
	}
	defer rows.Close()

	transactions := []models.WalletTransaction{}
	for rows.Next() {
		transaction := models.WalletTransaction{}
		err := rows.Scan(
			&transaction.ID, &transaction.WalletID, &transaction.MarketID, &transaction.Type, &transaction.Amount, &transaction.Status,
			&transaction.CreatedAt, &transaction.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan wallet transaction: %w", err)
		}
		transactions = append(transactions, transaction)
	}

	return transactions, nil
}

// UpdateTransactionStatus updates the status of a wallet transaction
func (r *Repository) UpdateTransactionStatus(ctx context.Context, transactionID string, status models.WalletTransactionStatus) error {
	query := `
		UPDATE wallet_transactions
		SET status = $1, updated_at = $2
		WHERE id = $3
	`
	result, err := r.db.ExecContext(ctx, query, status, time.Now(), transactionID)
	if err != nil {
		return fmt.Errorf("update transaction status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("wallet transaction not found")
	}

	return nil
}

// InitSchema initializes the wallet database schema
func (r *Repository) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS wallet_accounts (
		id UUID PRIMARY KEY,
		user_id UUID NOT NULL,
		currency VARCHAR(10) NOT NULL DEFAULT 'USDC',
		balance DECIMAL(20, 8) NOT NULL DEFAULT 0,
		available DECIMAL(20, 8) NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
		UNIQUE(user_id, currency)
	);

	CREATE INDEX IF NOT EXISTS idx_wallet_accounts_user_id ON wallet_accounts(user_id);
	CREATE INDEX IF NOT EXISTS idx_wallet_accounts_currency ON wallet_accounts(currency);

	CREATE TABLE IF NOT EXISTS wallet_transactions (
		id UUID PRIMARY KEY,
		wallet_id UUID NOT NULL REFERENCES wallet_accounts(id),
		market_id UUID,
		type VARCHAR(50) NOT NULL,
		amount DECIMAL(20, 8) NOT NULL,
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_wallet_transactions_wallet_id ON wallet_transactions(wallet_id);
	CREATE INDEX IF NOT EXISTS idx_wallet_transactions_market_id ON wallet_transactions(market_id);
	CREATE INDEX IF NOT EXISTS idx_wallet_transactions_type ON wallet_transactions(type);
	CREATE INDEX IF NOT EXISTS idx_wallet_transactions_status ON wallet_transactions(status);
	`

	_, err := r.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("init wallet schema: %w", err)
	}

	return nil
}