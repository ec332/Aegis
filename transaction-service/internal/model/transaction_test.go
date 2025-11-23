package model

import (
    "testing"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
)

func TestValidateCreate(t *testing.T) {
    ok := Transaction{
        UserID: uuid.New(),
        MarketID: uuid.New(),
        OptionID: uuid.New(),
        TransactionType: "BUY",
        NumberOfShares: decimal.RequireFromString("1.00"),
        PricePerShare: decimal.RequireFromString("2.00"),
    }
    if err := ValidateCreate(&ok); err != nil { t.Fatalf("unexpected error: %v", err) }

    bad := Transaction{}
    if err := ValidateCreate(&bad); err == nil { t.Fatalf("expected error") }
}

func TestValidateUpdate(t *testing.T) {
    ok := Transaction{
        TransactionType: "SELL",
        NumberOfShares: decimal.RequireFromString("5.00"),
        PricePerShare: decimal.RequireFromString("3.00"),
    }
    if err := ValidateUpdate(&ok); err != nil { t.Fatalf("unexpected error: %v", err) }

    bad := Transaction{}
    if err := ValidateUpdate(&bad); err == nil { t.Fatalf("expected error") }
}