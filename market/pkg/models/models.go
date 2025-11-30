package models

import (
	"time"
)

// MarketStatus represents the current state of a market
type MarketStatus string

const (
	MarketStatusDraft     MarketStatus = "draft"
	MarketStatusActive    MarketStatus = "active"
	MarketStatusHidden    MarketStatus = "hidden"
	MarketStatusResolving MarketStatus = "resolving"
	MarketStatusResolved  MarketStatus = "resolved"
)

// Market
type Market struct {
	ID                 string          `json:"id"`
	Title              string          `json:"title"`
	Description        string          `json:"description"`
	Status             MarketStatus    `json:"status"`
	ResolutionDatetime *time.Time      `json:"resolution_datetime,omitempty"`
	WinningOptionID    *string         `json:"winning_option_id,omitempty"`
	LiquidityParameter float64         `json:"liquidity_parameter"`
	Options            []Option        `json:"options,omitempty"`
	LiquidityPools     []LiquidityPool `json:"liquidity_pools,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

// Each market will have options
type Option struct {
	ID        string    `json:"id"`
	MarketID  string    `json:"market_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

// LiquidityPool tracks liquidity for each option in a market
type LiquidityPool struct {
	ID            string    `json:"id"`
	MarketID      string    `json:"market_id"`
	OptionID      string    `json:"option_id"`
	ShareQuantity float64   `json:"share_quantity"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// CreateMarketRequest represents the payload for creating a new market
type CreateMarketRequest struct {
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	ResolutionDatetime *time.Time `json:"resolution_datetime,omitempty"`
	LiquidityParameter float64    `json:"liquidity_parameter,omitempty"`
	Options            []string   `json:"options"`
}

// UpdateMarketRequest represents the payload for updating a market
type UpdateMarketRequest struct {
	Status             *MarketStatus `json:"status,omitempty"`
	WinningOptionID    *string       `json:"winning_option_id,omitempty"`
	ResolutionDatetime *time.Time    `json:"resolution_datetime,omitempty"`
}

// LiquidityUpdate represents a liquidity pool update published to Redis
type LiquidityUpdate struct {
	MarketID       string          `json:"market_id"`
	LiquidityPools []LiquidityPool `json:"liquidity_pools"`
	Timestamp      time.Time       `json:"timestamp"`
}

// Response for market listing
type MarketListResponse struct {
	Markets []Market `json:"markets"`
	Total   int      `json:"total"`
}

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

// Error Response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// Position represents a user's holdings in a market option
type Position struct {
	ID        string    `json:"id"`
	MarketID  string    `json:"market_id"`
	OptionID  string    `json:"option_id"`
	UserID    string    `json:"user_id"`
	Shares    float64   `json:"shares"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CostCalculationRequest represents the payload for calculating buy/sell costs
type CostCalculationRequest struct {
	Amount float64 `json:"amount"`
}

// CostCalculationResponse represents the response for cost calculation
type CostCalculationResponse struct {
	Cost float64 `json:"cost"`
}
