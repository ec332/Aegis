//go:build integration
// +build integration

package postgres

import (
    "context"
    "os"
    "testing"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
    "github.com/shopspring/decimal"
    "aegis/internal/model"
)

func TestRepoIntegration(t *testing.T) {
    dsn := os.Getenv("PG_DSN")
    if dsn == "" {
        host := getenv("APP_DB_HOST", "localhost")
        port := getenv("APP_DB_PORT", "5432")
        name := getenv("APP_DB_NAME", "transaction")
        user := getenv("APP_DB_USER", "postgres")
        pass := getenv("APP_DB_PASSWORD", "postgres")
        dsn = "postgres://"+user+":"+pass+"@"+host+":"+port+"/"+name+"?sslmode=disable"
    }
    cfg, err := pgxpool.ParseConfig(dsn)
    if err != nil { t.Fatal(err) }
    pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
    if err != nil { t.Fatal(err) }
    defer pool.Close()
    repo := New(pool)

    ctx := context.Background()
    tx := model.Transaction{
        ID: uuid.New(),
        UserID: uuid.New(),
        MarketID: uuid.New(),
        OptionID: uuid.New(),
        TransactionType: "BUY",
        NumberOfShares: decimal.RequireFromString("1.00"),
        PricePerShare: decimal.RequireFromString("1.23"),
        CreatedAt: time.Now().UTC(),
    }
    created, err := repo.Insert(ctx, tx)
    if err != nil { t.Fatal(err) }
    got, err := repo.FindByID(ctx, created.ID)
    if err != nil { t.Fatal(err) }
    if got.ID != created.ID { t.Fatal("id mismatch") }
    n, err := repo.DeleteByID(ctx, created.ID)
    if err != nil { t.Fatal(err) }
    if n != 1 { t.Fatalf("expected 1, got %d", n) }
}

func getenv(k, def string) string {
    v := os.Getenv(k)
    if v == "" { return def }
    return v
}