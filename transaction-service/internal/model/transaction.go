package model

import (
    "errors"
    "time"

    "github.com/google/uuid"
    "github.com/shopspring/decimal"
)

type Transaction struct {
    ID               uuid.UUID         `json:"id"`
    UserID           uuid.UUID         `json:"user_id"`
    MarketID         uuid.UUID         `json:"market_id"`
    OptionID         uuid.UUID         `json:"option_id"`
    TransactionType  string            `json:"transaction_type"`
    NumberOfShares   decimal.Decimal   `json:"number_of_shares"`
    PricePerShare    decimal.Decimal   `json:"price_per_share"`
    CreatedAt        time.Time         `json:"created_at"`
}

func ValidateCreate(t *Transaction) error {
    if t.UserID == uuid.Nil {
        return errors.New("user_id is required")
    }
    if t.MarketID == uuid.Nil {
        return errors.New("market_id is required")
    }
    if t.OptionID == uuid.Nil {
        return errors.New("option_id is required")
    }
    if t.TransactionType == "" {
        return errors.New("transaction_type is required")
    }
    if !t.NumberOfShares.IsPositive() {
        return errors.New("number_of_shares must be positive")
    }
    if !t.PricePerShare.IsPositive() {
        return errors.New("price_per_share must be positive")
    }
    return nil
}

func ValidateUpdate(t *Transaction) error {
    if t.TransactionType == "" {
        return errors.New("transaction_type is required")
    }
    if !t.NumberOfShares.IsPositive() {
        return errors.New("number_of_shares must be positive")
    }
    if !t.PricePerShare.IsPositive() {
        return errors.New("price_per_share must be positive")
    }
    return nil
}