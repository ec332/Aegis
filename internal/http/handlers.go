package http

import (
    "context"
    "encoding/json"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/google/uuid"
    "github.com/jackc/pgx/v5/pgxpool"
    "go.uber.org/zap"

    "aegis/internal/model"
    "aegis/internal/service"
    storeiface "aegis/internal/store"
    storepg "aegis/internal/store/postgres"
)

type Handler struct { svc *service.TransactionService; logger *zap.Logger }

func New(pool *pgxpool.Pool, logger *zap.Logger) *Handler {
    repo := storepg.New(pool)
    return &Handler{svc: service.NewTransactionService(repo), logger: logger}
}

func NewWithRepo(repo storeiface.Repository, logger *zap.Logger) *Handler {
    return &Handler{svc: service.NewTransactionService(repo), logger: logger}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
    r.Route("/transactions", func(r chi.Router) {
        r.Get("/", h.List)
        r.Post("/", h.Create)
        r.Get("/{id}", h.GetOne)
        r.Put("/{id}", h.Update)
        r.Delete("/{id}", h.Delete)
    })
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    items, err := h.svc.FindAll(ctx)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err)
        return
    }
    writeJSON(w, http.StatusOK, items)
}

func (h *Handler) GetOne(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    idStr := chi.URLParam(r, "id")
    id, err := uuid.Parse(idStr)
    if err != nil {
        writeError(w, http.StatusNotFound, err)
        return
    }
    t, err := h.svc.FindByID(ctx, id)
    if err != nil {
        writeError(w, http.StatusNotFound, err)
        return
    }
    writeJSON(w, http.StatusOK, t)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    var body struct {
        UserID          string `json:"user_id"`
        MarketID        string `json:"market_id"`
        OptionID        string `json:"option_id"`
        TransactionType string `json:"transaction_type"`
        NumberOfShares  string `json:"number_of_shares"`
        PricePerShare   string `json:"price_per_share"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        writeError(w, http.StatusBadRequest, err)
        return
    }
    t := model.Transaction{
        UserID:          uuid.MustParse(body.UserID),
        MarketID:        uuid.MustParse(body.MarketID),
        OptionID:        uuid.MustParse(body.OptionID),
        TransactionType: body.TransactionType,
        NumberOfShares:  service.MustDecimalFromString(body.NumberOfShares),
        PricePerShare:   service.MustDecimalFromString(body.PricePerShare),
    }
    out, err := h.svc.Create(context.Background(), t)
    if err != nil {
        writeError(w, http.StatusBadRequest, err)
        return
    }
    writeJSON(w, http.StatusCreated, out)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := uuid.Parse(idStr)
    if err != nil {
        writeError(w, http.StatusNotFound, err)
        return
    }
    var body struct {
        UserID          string `json:"user_id"`
        MarketID        string `json:"market_id"`
        OptionID        string `json:"option_id"`
        TransactionType string `json:"transaction_type"`
        NumberOfShares  string `json:"number_of_shares"`
        PricePerShare   string `json:"price_per_share"`
        CreatedAt       *time.Time `json:"created_at"`
    }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
        writeError(w, http.StatusBadRequest, err)
        return
    }
    var createdAt time.Time
    if body.CreatedAt != nil {
        createdAt = *body.CreatedAt
    }
    t := model.Transaction{
        UserID:          uuid.MustParse(body.UserID),
        MarketID:        uuid.MustParse(body.MarketID),
        OptionID:        uuid.MustParse(body.OptionID),
        TransactionType: body.TransactionType,
        NumberOfShares:  service.MustDecimalFromString(body.NumberOfShares),
        PricePerShare:   service.MustDecimalFromString(body.PricePerShare),
        CreatedAt:       createdAt,
    }
    out, err := h.svc.Update(context.Background(), id, t)
    if err != nil {
        writeError(w, http.StatusBadRequest, err)
        return
    }
    writeJSON(w, http.StatusOK, out)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
    idStr := chi.URLParam(r, "id")
    id, err := uuid.Parse(idStr)
    if err != nil {
        writeError(w, http.StatusNotFound, err)
        return
    }
    n, err := h.svc.DeleteByID(context.Background(), id)
    if err != nil {
        writeError(w, http.StatusInternalServerError, err)
        return
    }
    if n == 0 {
        writeJSON(w, http.StatusNotFound, map[string]any{})
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
    writeJSON(w, status, map[string]string{"error": err.Error()})
}