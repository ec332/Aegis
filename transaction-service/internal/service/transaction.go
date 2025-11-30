package service

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/shopspring/decimal"

    "transaction-service/internal/model"
    "transaction-service/internal/store"
)

type TransactionService struct { repo store.Repository }

func NewTransactionService(repo store.Repository) *TransactionService {
    return &TransactionService{repo: repo}
}

func (s *TransactionService) FindAll(ctx context.Context) ([]model.Transaction, error) {
    return s.repo.FindAll(ctx)
}

func (s *TransactionService) FindByID(ctx context.Context, id uuid.UUID) (model.Transaction, error) {
    return s.repo.FindByID(ctx, id)
}

func (s *TransactionService) Create(ctx context.Context, t model.Transaction) (model.Transaction, error) {
    if err := model.ValidateCreate(&t); err != nil {
        return model.Transaction{}, err
    }
    if t.ID == uuid.Nil {
        t.ID = uuid.New()
    }
    if t.CreatedAt.IsZero() {
        t.CreatedAt = time.Now().UTC()
    }
    created, err := s.repo.Insert(ctx, t)
    if err != nil {
        return model.Transaction{}, err
    }
    delta := t.NumberOfShares.InexactFloat64()
    if t.TransactionType == "SELL" {
        delta = -delta
    }
    _ = s.repo.AdjustMarketLiquidity(ctx, t.MarketID, t.OptionID, delta)
    return created, nil
}

func (s *TransactionService) Update(ctx context.Context, id uuid.UUID, t model.Transaction) (model.Transaction, error) {
    if err := model.ValidateUpdate(&t); err != nil {
        return model.Transaction{}, err
    }
    t.ID = id
    if t.CreatedAt.IsZero() {
        existing, err := s.repo.FindByID(ctx, id)
        if err == nil {
            t.CreatedAt = existing.CreatedAt
        } else {
            t.CreatedAt = time.Now().UTC()
        }
    }
    return s.repo.Update(ctx, t)
}

func (s *TransactionService) DeleteByID(ctx context.Context, id uuid.UUID) (int64, error) {
    return s.repo.DeleteByID(ctx, id)
}

func (s *TransactionService) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Transaction, error) {
    return s.repo.FindByUserID(ctx, userID)
}

func (s *TransactionService) FindByMarketID(ctx context.Context, marketID uuid.UUID) ([]model.Transaction, error) {
    return s.repo.FindByMarketID(ctx, marketID)
}

// helpers to parse decimals
func MustDecimalFromString(s string) decimal.Decimal {
    d, _ := decimal.NewFromString(s)
    return d
}
