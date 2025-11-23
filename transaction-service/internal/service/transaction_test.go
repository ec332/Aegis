package service

import (
    "context"
    "testing"
    "time"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
    mem "aegis/internal/store/memory"
    "aegis/internal/model"
)

func TestCreateSetsDefaults(t *testing.T) {
    svc := NewTransactionService(mem.New())
    in := model.Transaction{
        UserID: uuid.New(),
        MarketID: uuid.New(),
        OptionID: uuid.New(),
        TransactionType: "BUY",
        NumberOfShares: decimal.RequireFromString("1"),
        PricePerShare: decimal.RequireFromString("1.23"),
    }
    out, err := svc.Create(context.Background(), in)
    if err != nil { t.Fatalf("unexpected: %v", err) }
    if out.ID == uuid.Nil { t.Fatalf("missing id") }
    if out.CreatedAt.IsZero() { t.Fatalf("missing created_at") }
}

func TestUpdatePreservesCreatedAt(t *testing.T) {
    svc := NewTransactionService(mem.New())
    base := model.Transaction{
        UserID: uuid.New(),
        MarketID: uuid.New(),
        OptionID: uuid.New(),
        TransactionType: "BUY",
        NumberOfShares: decimal.RequireFromString("1"),
        PricePerShare: decimal.RequireFromString("1.23"),
    }
    created, err := svc.Create(context.Background(), base)
    if err != nil { t.Fatalf("unexpected: %v", err) }
    upd := model.Transaction{
        UserID: created.UserID,
        MarketID: created.MarketID,
        OptionID: created.OptionID,
        TransactionType: "SELL",
        NumberOfShares: decimal.RequireFromString("2"),
        PricePerShare: decimal.RequireFromString("2.34"),
    }
    out, err := svc.Update(context.Background(), created.ID, upd)
    if err != nil { t.Fatalf("unexpected: %v", err) }
    if !out.CreatedAt.Equal(created.CreatedAt) { t.Fatalf("created_at changed") }
}

func TestDeleteByID(t *testing.T) {
    svc := NewTransactionService(mem.New())
    base := model.Transaction{
        UserID: uuid.New(),
        MarketID: uuid.New(),
        OptionID: uuid.New(),
        TransactionType: "BUY",
        NumberOfShares: decimal.RequireFromString("1"),
        PricePerShare: decimal.RequireFromString("1.23"),
        CreatedAt: time.Now().UTC(),
    }
    created, _ := svc.Create(context.Background(), base)
    n, err := svc.DeleteByID(context.Background(), created.ID)
    if err != nil { t.Fatalf("unexpected: %v", err) }
    if n != 1 { t.Fatalf("expected 1, got %d", n) }
}