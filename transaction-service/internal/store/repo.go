package store

import (
    "context"

    "github.com/google/uuid"

    "transaction-service/internal/model"
)

type Repository interface {
    FindAll(ctx context.Context) ([]model.Transaction, error)
    FindByID(ctx context.Context, id uuid.UUID) (model.Transaction, error)
    FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Transaction, error)
    FindByMarketID(ctx context.Context, marketID uuid.UUID) ([]model.Transaction, error)
    Insert(ctx context.Context, t model.Transaction) (model.Transaction, error)
    Update(ctx context.Context, t model.Transaction) (model.Transaction, error)
    DeleteByID(ctx context.Context, id uuid.UUID) (int64, error)
    AdjustMarketLiquidity(ctx context.Context, marketID, optionID uuid.UUID, deltaShares float64) error
}