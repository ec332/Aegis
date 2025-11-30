package models

import (
	"time"
)

// UserRole represents the role of a user in the system
type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

// User represents a user in the system
type User struct {
	ID            string     `json:"id"`
	WalletAddress string     `json:"wallet_address"`
	Balance       float64    `json:"balance"`
	Nonce         string     `json:"nonce"`
	Role          UserRole   `json:"role"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastLogin     *time.Time `json:"last_login,omitempty"`
}

// CreateUserRequest represents the payload for creating a new user
type CreateUserRequest struct {
	WalletAddress string  `json:"wallet_address"`
	Balance       float64 `json:"balance"`
}

// UpdateUserRequest represents the payload for updating a user
type UpdateUserRequest struct {
    Balance *float64  `json:"balance,omitempty"`
    Nonce   *string   `json:"nonce,omitempty"`
    Role    *UserRole `json:"role,omitempty"`
}

type WalletAccount struct {
    ID        string    `json:"id"`
    UserID    string    `json:"user_id"`
    Currency  string    `json:"currency"`
    Balance   float64   `json:"balance"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type WalletTransaction struct {
    ID           string    `json:"id"`
    WalletID     string    `json:"wallet_id"`
    Type         string    `json:"type"`
    Amount       float64   `json:"amount"`
    BalanceAfter float64   `json:"balance_after"`
    Description  string    `json:"description"`
    Status       string    `json:"status"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}
