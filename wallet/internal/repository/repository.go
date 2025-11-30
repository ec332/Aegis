package repository

import (
    "context"
    "database/sql"
    "fmt"
    "time"

    "aegis/wallet/pkg/models"

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

// CreateUser creates a new user
func (r *Repository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
        INSERT INTO users (id, wallet_address, balance, nonce, role, created_at, updated_at, last_login)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `
	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.WalletAddress, user.Balance, user.Nonce, user.Role,
		user.CreatedAt, user.UpdatedAt, user.LastLogin,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// GetUser retrieves a user by ID
func (r *Repository) GetUser(ctx context.Context, userID string) (*models.User, error) {
	user := &models.User{}
	query := `
        SELECT id, wallet_address, balance, nonce, role, created_at, updated_at, last_login
        FROM users
        WHERE id = $1
    `
	var lastLogin sql.NullTime
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.ID, &user.WalletAddress, &user.Balance, &user.Nonce, &user.Role,
		&user.CreatedAt, &user.UpdatedAt, &lastLogin,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("query user: %w", err)
	}
	if lastLogin.Valid {
		t := lastLogin.Time
		user.LastLogin = &t
	}
	return user, nil
}

// GetUserByWalletAddress retrieves a user by wallet address
func (r *Repository) GetUserByWalletAddress(ctx context.Context, walletAddress string) (*models.User, error) {
	user := &models.User{}
	query := `
        SELECT id, wallet_address, balance, nonce, role, created_at, updated_at, last_login
        FROM users
        WHERE wallet_address = $1
    `
	var lastLogin sql.NullTime
	err := r.db.QueryRowContext(ctx, query, walletAddress).Scan(
		&user.ID, &user.WalletAddress, &user.Balance, &user.Nonce, &user.Role,
		&user.CreatedAt, &user.UpdatedAt, &lastLogin,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("query user by wallet: %w", err)
	}
	if lastLogin.Valid {
		t := lastLogin.Time
		user.LastLogin = &t
	}
	return user, nil
}

// UpdateUser updates user fields
func (r *Repository) UpdateUser(ctx context.Context, userID string, updates models.UpdateUserRequest) error {
	query := "UPDATE users SET updated_at = $1"
	args := []interface{}{time.Now()}
	argCount := 2

	if updates.Balance != nil {
		query += fmt.Sprintf(", balance = $%d", argCount)
		args = append(args, *updates.Balance)
		argCount++
	}
	if updates.Nonce != nil {
		query += fmt.Sprintf(", nonce = $%d", argCount)
		args = append(args, *updates.Nonce)
		argCount++
	}
	if updates.Role != nil {
		query += fmt.Sprintf(", role = $%d", argCount)
		args = append(args, *updates.Role)
		argCount++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, userID)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// InitSchema initializes the database schema
func (r *Repository) InitSchema(ctx context.Context) error {
	schema := `
    CREATE TABLE IF NOT EXISTS users (
        id UUID PRIMARY KEY,
        wallet_address VARCHAR(255) NOT NULL UNIQUE,
        balance DECIMAL(20, 8) NOT NULL DEFAULT 0,
        nonce TEXT NOT NULL,
        role VARCHAR(50) NOT NULL DEFAULT 'user',
        created_at TIMESTAMP NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
        last_login TIMESTAMP NULL
    );

	CREATE INDEX IF NOT EXISTS idx_users_wallet_address ON users(wallet_address);
	CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);
    `

	_, err := r.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("init schema: %w", err)
	}

	return nil
}

// SetNonceByWallet updates nonce for a user identified by wallet address
func (r *Repository) SetNonceByWallet(ctx context.Context, walletAddress string, nonce string) error {
	query := `UPDATE users SET nonce = $1, updated_at = NOW() WHERE wallet_address = $2`
	res, err := r.db.ExecContext(ctx, query, nonce, walletAddress)
	if err != nil {
		return fmt.Errorf("update nonce: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// SetLastLoginByWallet updates last_login for a user identified by wallet address
func (r *Repository) SetLastLoginByWallet(ctx context.Context, walletAddress string, t time.Time) error {
	query := `UPDATE users SET last_login = $1, updated_at = NOW() WHERE wallet_address = $2`
	res, err := r.db.ExecContext(ctx, query, t, walletAddress)
	if err != nil {
		return fmt.Errorf("update last_login: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *Repository) GetWalletAccountByUserCurrency(ctx context.Context, userID string, currency string) (*models.WalletAccount, error) {
    acc := &models.WalletAccount{}
    query := `SELECT id, user_id, currency, balance, status, created_at, updated_at FROM wallet_accounts WHERE user_id = $1 AND currency = $2`
    err := r.db.QueryRowContext(ctx, query, userID, currency).Scan(
        &acc.ID, &acc.UserID, &acc.Currency, &acc.Balance, &acc.Status, &acc.CreatedAt, &acc.UpdatedAt,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("not found")
        }
        return nil, fmt.Errorf("query wallet account: %w", err)
    }
    return acc, nil
}

func (r *Repository) CreateWalletAccount(ctx context.Context, acc *models.WalletAccount) (*models.WalletAccount, error) {
    query := `INSERT INTO wallet_accounts (id, user_id, currency, balance, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, user_id, currency, balance, status, created_at, updated_at`
    var out models.WalletAccount
    err := r.db.QueryRowContext(ctx, query, acc.ID, acc.UserID, acc.Currency, acc.Balance, acc.Status, acc.CreatedAt, acc.UpdatedAt).Scan(
        &out.ID, &out.UserID, &out.Currency, &out.Balance, &out.Status, &out.CreatedAt, &out.UpdatedAt,
    )
    if err != nil {
        return nil, fmt.Errorf("insert wallet account: %w", err)
    }
    return &out, nil
}

func (r *Repository) GetWalletAccount(ctx context.Context, id string) (*models.WalletAccount, error) {
    acc := &models.WalletAccount{}
    query := `SELECT id, user_id, currency, balance, status, created_at, updated_at FROM wallet_accounts WHERE id = $1`
    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &acc.ID, &acc.UserID, &acc.Currency, &acc.Balance, &acc.Status, &acc.CreatedAt, &acc.UpdatedAt,
    )
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, fmt.Errorf("not found")
        }
        return nil, fmt.Errorf("query wallet account by id: %w", err)
    }
    return acc, nil
}

func (r *Repository) UpdateWalletBalanceAndCreateTransaction(ctx context.Context, walletID string, amount float64, txType string, description string) (*models.WalletTransaction, float64, error) {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, 0, fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback()
    var currentBalance float64
    err = tx.QueryRowContext(ctx, "SELECT balance FROM wallet_accounts WHERE id = $1 FOR UPDATE", walletID).Scan(&currentBalance)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, 0, fmt.Errorf("not found")
        }
        return nil, 0, fmt.Errorf("get balance: %w", err)
    }
    newBalance := currentBalance + amount
    if newBalance < 0 {
        return nil, 0, fmt.Errorf("insufficient funds")
    }
    _, err = tx.ExecContext(ctx, "UPDATE wallet_accounts SET balance = $1, updated_at = $2 WHERE id = $3", newBalance, time.Now(), walletID)
    if err != nil {
        return nil, 0, fmt.Errorf("update balance: %w", err)
    }
    id := fmt.Sprintf("t-%d", time.Now().UnixNano())
    now := time.Now()
    _, err = tx.ExecContext(ctx, `INSERT INTO wallet_transactions (id, wallet_id, type, amount, balance_after, description, status, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, id, walletID, txType, amount, newBalance, description, "completed", now)
    if err != nil {
        return nil, 0, fmt.Errorf("insert transaction: %w", err)
    }
    if err = tx.Commit(); err != nil {
        return nil, 0, fmt.Errorf("commit: %w", err)
    }
    return &models.WalletTransaction{ID: id, WalletID: walletID, Type: txType, Amount: amount, BalanceAfter: newBalance, Description: description, Status: "completed", CreatedAt: now, UpdatedAt: now}, newBalance, nil
}
