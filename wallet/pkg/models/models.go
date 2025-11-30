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
    ID            string    `json:"id"`
    WalletAddress string    `json:"wallet_address"`
    Balance       float64   `json:"balance"`
    Nonce         string    `json:"nonce"`
    Role          UserRole  `json:"role"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
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
