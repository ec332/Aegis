package models

import (
	"time"
)

type SettlementStatus string

const (
	SettlementStatusPending   SettlementStatus = "pending"
	SettlementStatusCompleted SettlementStatus = "completed"
	SettlementStatusFailed    SettlementStatus = "failed"
)

type Settlement struct {
	ID           string           `json:"id"`
	MarketID     string           `json:"market_id"`
	WinningOptionID string        `json:"winning_option_id"`
	TotalPool    float64          `json:"total_pool"`
	WinningPool  float64          `json:"winning_pool"`
	Status       SettlementStatus `json:"status"`
	SettledAt    *time.Time       `json:"settled_at,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

type SettlementRequest struct {
	MarketID        string `json:"market_id"`
	WinningOptionID string `json:"winning_option_id"`
}

type SettlementResponse struct {
	Settlement *Settlement `json:"settlement"`
	Message    string      `json:"message"`
}

type SettlementDistribution struct {
	ID           string    `json:"id"`
	SettlementID string    `json:"settlement_id"`
	UserID       string    `json:"user_id"`
	Amount       float64   `json:"amount"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}