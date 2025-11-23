package memory

import (
    "context"
    "errors"
    "sync"
    "time"

    "github.com/google/uuid"

    "aegis/internal/model"
    "aegis/internal/store"
)

type Repository struct {
    mu   sync.RWMutex
    data map[uuid.UUID]model.Transaction
}

func New() *Repository { return &Repository{data: make(map[uuid.UUID]model.Transaction)} }

func (r *Repository) FindAll(ctx context.Context) ([]model.Transaction, error) {
    r.mu.RLock(); defer r.mu.RUnlock()
    out := make([]model.Transaction, 0, len(r.data))
    for _, t := range r.data {
        out = append(out, t)
    }
    return out, nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (model.Transaction, error) {
    r.mu.RLock(); defer r.mu.RUnlock()
    t, ok := r.data[id]
    if !ok { return model.Transaction{}, errors.New("not found") }
    return t, nil
}

func (r *Repository) Insert(ctx context.Context, t model.Transaction) (model.Transaction, error) {
    r.mu.Lock(); defer r.mu.Unlock()
    if t.ID == uuid.Nil { t.ID = uuid.New() }
    if t.CreatedAt.IsZero() { t.CreatedAt = time.Now().UTC() }
    r.data[t.ID] = t
    return t, nil
}

func (r *Repository) Update(ctx context.Context, t model.Transaction) (model.Transaction, error) {
    r.mu.Lock(); defer r.mu.Unlock()
    if _, ok := r.data[t.ID]; !ok { return model.Transaction{}, errors.New("not found") }
    r.data[t.ID] = t
    return t, nil
}

func (r *Repository) DeleteByID(ctx context.Context, id uuid.UUID) (int64, error) {
    r.mu.Lock(); defer r.mu.Unlock()
    if _, ok := r.data[id]; !ok { return 0, nil }
    delete(r.data, id)
    return 1, nil
}

var _ store.Repository = (*Repository)(nil)