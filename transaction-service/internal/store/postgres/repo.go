package postgres

import (
    "context"
    "errors"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/shopspring/decimal"

    "transaction-service/internal/model"
    "transaction-service/internal/store"
)

type Repository struct {
    pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

func (r *Repository) FindAll(ctx context.Context) ([]model.Transaction, error) {
    rows, err := r.pool.Query(ctx, `SELECT id, user_id, market_id, option_id, transaction_type, number_of_shares, price_per_share, created_at FROM transactions ORDER BY created_at DESC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var out []model.Transaction
    for rows.Next() {
        var t model.Transaction
        var id, userID, marketID, optionID uuid.UUID
        var typ string
        var shares, price decimal.Decimal
        var created time.Time
        if err := rows.Scan(&id, &userID, &marketID, &optionID, &typ, &shares, &price, &created); err != nil {
            return nil, err
        }
        t.ID = id
        t.UserID = userID
        t.MarketID = marketID
        t.OptionID = optionID
        t.TransactionType = typ
        t.NumberOfShares = shares
        t.PricePerShare = price
        t.CreatedAt = created
        out = append(out, t)
    }
    return out, rows.Err()
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (model.Transaction, error) {
    var t model.Transaction
    row := r.pool.QueryRow(ctx, `SELECT id, user_id, market_id, option_id, transaction_type, number_of_shares, price_per_share, created_at FROM transactions WHERE id=$1`, id)
    var tid, userID, marketID, optionID uuid.UUID
    var typ string
    var shares, price decimal.Decimal
    var created time.Time
    err := row.Scan(&tid, &userID, &marketID, &optionID, &typ, &shares, &price, &created)
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return model.Transaction{}, err
        }
        return model.Transaction{}, err
    }
    t.ID = tid
    t.UserID = userID
    t.MarketID = marketID
    t.OptionID = optionID
    t.TransactionType = typ
    t.NumberOfShares = shares
    t.PricePerShare = price
    t.CreatedAt = created
    return t, nil
}

func (r *Repository) Insert(ctx context.Context, t model.Transaction) (model.Transaction, error) {
    var out model.Transaction
    row := r.pool.QueryRow(ctx, `INSERT INTO transactions (id, user_id, market_id, option_id, transaction_type, number_of_shares, price_per_share, created_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id, user_id, market_id, option_id, transaction_type, number_of_shares, price_per_share, created_at`,
        t.ID, t.UserID, t.MarketID, t.OptionID, t.TransactionType, t.NumberOfShares, t.PricePerShare, t.CreatedAt,
    )
    var tid, userID, marketID, optionID uuid.UUID
    var typ string
    var shares, price decimal.Decimal
    var created time.Time
    if err := row.Scan(&tid, &userID, &marketID, &optionID, &typ, &shares, &price, &created); err != nil {
        return model.Transaction{}, err
    }
    out.ID = tid
    out.UserID = userID
    out.MarketID = marketID
    out.OptionID = optionID
    out.TransactionType = typ
    out.NumberOfShares = shares
    out.PricePerShare = price
    out.CreatedAt = created
    return out, nil
}

func (r *Repository) Update(ctx context.Context, t model.Transaction) (model.Transaction, error) {
    var out model.Transaction
    row := r.pool.QueryRow(ctx, `UPDATE transactions SET user_id=$2, market_id=$3, option_id=$4, transaction_type=$5, number_of_shares=$6, price_per_share=$7, created_at=$8 WHERE id=$1 RETURNING id, user_id, market_id, option_id, transaction_type, number_of_shares, price_per_share, created_at`,
        t.ID, t.UserID, t.MarketID, t.OptionID, t.TransactionType, t.NumberOfShares, t.PricePerShare, t.CreatedAt,
    )
    var tid, userID, marketID, optionID uuid.UUID
    var typ string
    var shares, price decimal.Decimal
    var created time.Time
    if err := row.Scan(&tid, &userID, &marketID, &optionID, &typ, &shares, &price, &created); err != nil {
        return model.Transaction{}, err
    }
    out.ID = tid
    out.UserID = userID
    out.MarketID = marketID
    out.OptionID = optionID
    out.TransactionType = typ
    out.NumberOfShares = shares
    out.PricePerShare = price
    out.CreatedAt = created
    return out, nil
}

func (r *Repository) DeleteByID(ctx context.Context, id uuid.UUID) (int64, error) {
    ct, err := r.pool.Exec(ctx, `DELETE FROM transactions WHERE id=$1`, id)
    if err != nil {
        return 0, err
    }
    return ct.RowsAffected(), nil
}

func (r *Repository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Transaction, error) {
    rows, err := r.pool.Query(ctx, `SELECT id, user_id, market_id, option_id, transaction_type, number_of_shares, price_per_share, created_at FROM transactions WHERE user_id=$1 ORDER BY created_at DESC`, userID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var out []model.Transaction
    for rows.Next() {
        var t model.Transaction
        var id, userID, marketID, optionID uuid.UUID
        var typ string
        var shares, price decimal.Decimal
        var created time.Time
        if err := rows.Scan(&id, &userID, &marketID, &optionID, &typ, &shares, &price, &created); err != nil {
            return nil, err
        }
        t.ID = id
        t.UserID = userID
        t.MarketID = marketID
        t.OptionID = optionID
        t.TransactionType = typ
        t.NumberOfShares = shares
        t.PricePerShare = price
        t.CreatedAt = created
        out = append(out, t)
    }
    return out, rows.Err()
}

func (r *Repository) FindByMarketID(ctx context.Context, marketID uuid.UUID) ([]model.Transaction, error) {
    rows, err := r.pool.Query(ctx, `SELECT id, user_id, market_id, option_id, transaction_type, number_of_shares, price_per_share, created_at FROM transactions WHERE market_id=$1 ORDER BY created_at DESC`, marketID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var out []model.Transaction
    for rows.Next() {
        var t model.Transaction
        var id, userID, marketID, optionID uuid.UUID
        var typ string
        var shares, price decimal.Decimal
        var created time.Time
        if err := rows.Scan(&id, &userID, &marketID, &optionID, &typ, &shares, &price, &created); err != nil {
            return nil, err
        }
        t.ID = id
        t.UserID = userID
        t.MarketID = marketID
        t.OptionID = optionID
        t.TransactionType = typ
        t.NumberOfShares = shares
        t.PricePerShare = price
        t.CreatedAt = created
        out = append(out, t)
    }
    return out, rows.Err()
}

var _ store.Repository = (*Repository)(nil)