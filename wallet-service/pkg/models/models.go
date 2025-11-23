package models

import (
	"time"
)

// Currency represents the type of currency
type Currency string

const (
	CurrencyUSDC Currency = "USDC"
)

// WalletAccount represents a user's wallet account
type WalletAccount struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Currency  Currency  `json:"currency"`
	Balance   float64   `json:"balance"`
	Available float64   `json:"available"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WalletTransaction represents a transaction in the wallet
type WalletTransaction struct {
	ID         string                   `json:"id"`
	WalletID   string                   `json:"wallet_id"`
	MarketID   *string                  `json:"market_id,omitempty"`
	Type       WalletTransactionType    `json:"type"`
	Amount     float64                  `json:"amount"`
	Status     WalletTransactionStatus  `json:"status"`
	CreatedAt  time.Time                `json:"created_at"`
	UpdatedAt  time.Time                `json:"updated_at"`
}

// WalletTransactionType represents the type of wallet transaction
type WalletTransactionType string

const (
	WalletTransactionTypeDeposit   WalletTransactionType = "deposit"
	WalletTransactionTypeDebit     WalletTransactionType = "debit"
	WalletTransactionTypeWithdrawal WalletTransactionType = "withdrawal"
	WalletTransactionTypeSettlement WalletTransactionType = "settlement"
)

// WalletTransactionStatus represents the status of a wallet transaction
type WalletTransactionStatus string

const (
	WalletTransactionStatusPending   WalletTransactionStatus = "pending"
	WalletTransactionStatusCompleted WalletTransactionStatus = "completed"
	WalletTransactionStatusFailed    WalletTransactionStatus = "failed"
)

// CreateWalletAccountRequest represents the payload for creating a wallet account
type CreateWalletAccountRequest struct {
	UserID   string   `json:"user_id"`
	Currency Currency `json:"currency"`
}

// CreateWalletTransactionRequest represents the payload for creating a wallet transaction
type CreateWalletTransactionRequest struct {
	WalletID string                `json:"wallet_id"`
	MarketID *string                 `json:"market_id,omitempty"`
	Type     WalletTransactionType `json:"type"`
	Amount   float64               `json:"amount"`
}

// UpdateWalletBalanceRequest represents the payload for updating wallet balance
type UpdateWalletBalanceRequest struct {
	Amount float64 `json:"amount"`
}

// DebitWalletRequest represents the payload for debiting a wallet
type DebitWalletRequest struct {
	MarketID string  `json:"market_id"`
	Amount   float64 `json:"amount"`
}

// CreditWalletRequest represents the payload for crediting a wallet
type CreditWalletRequest struct {
	MarketID string  `json:"market_id"`
	Amount   float64 `json:"amount"`
}

// DepositRequest represents the payload for depositing funds
type DepositRequest struct {
	Amount float64 `json:"amount"`
}

// WithdrawalRequest represents the payload for withdrawing funds
type WithdrawalRequest struct {
	Amount float64 `json:"amount"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}