package store

import (
    "context"

    "github.com/google/uuid"

    "aegis/internal/model"
)

type Repository interface {
    FindAll(ctx context.Context) ([]model.Transaction, error)
    FindByID(ctx context.Context, id uuid.UUID) (model.Transaction, error)
    Insert(ctx context.Context, t model.Transaction) (model.Transaction, error)
    Update(ctx context.Context, t model.Transaction) (model.Transaction, error)
    DeleteByID(ctx context.Context, id uuid.UUID) (int64, error)
}