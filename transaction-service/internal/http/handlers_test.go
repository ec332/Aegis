package http

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/go-chi/chi/v5"
    "go.uber.org/zap"

    "aegis/internal/store/memory"
)

func TestCreateAndList(t *testing.T) {
    repo := memory.New()
    h := NewWithRepo(repo, zap.NewNop())
    r := chi.NewRouter()
    h.RegisterRoutes(r)

    body := map[string]string{
        "user_id":          "00000000-0000-0000-0000-000000000001",
        "market_id":        "00000000-0000-0000-0000-000000000002",
        "option_id":        "00000000-0000-0000-0000-000000000003",
        "transaction_type": "BUY",
        "number_of_shares": "10",
        "price_per_share":  "1.23",
    }
    b, _ := json.Marshal(body)
    req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewReader(b))
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)
    if w.Code != http.StatusCreated {
        t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
    }

    req2 := httptest.NewRequest(http.MethodGet, "/transactions", nil)
    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, req2)
    if w2.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
    }
}

func TestCrudAndErrors(t *testing.T) {
    repo := memory.New()
    h := NewWithRepo(repo, zap.NewNop())
    r := chi.NewRouter()
    h.RegisterRoutes(r)

    body := map[string]string{
        "user_id":          "00000000-0000-0000-0000-000000000001",
        "market_id":        "00000000-0000-0000-0000-000000000002",
        "option_id":        "00000000-0000-0000-0000-000000000003",
        "transaction_type": "BUY",
        "number_of_shares": "10",
        "price_per_share":  "1.23",
    }
    b, _ := json.Marshal(body)
    w := httptest.NewRecorder()
    r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewReader(b)))
    if w.Code != http.StatusCreated { t.Fatalf("expected 201, got %d", w.Code) }
    var created map[string]any
    _ = json.Unmarshal(w.Body.Bytes(), &created)
    id := created["id"].(string)

    w2 := httptest.NewRecorder()
    r.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/transactions/"+id, nil))
    if w2.Code != http.StatusOK { t.Fatalf("expected 200, got %d", w2.Code) }

    upd := map[string]string{
        "user_id":          body["user_id"],
        "market_id":        body["market_id"],
        "option_id":        body["option_id"],
        "transaction_type": "SELL",
        "number_of_shares": "5",
        "price_per_share":  "2.00",
    }
    ub, _ := json.Marshal(upd)
    w3 := httptest.NewRecorder()
    r.ServeHTTP(w3, httptest.NewRequest(http.MethodPut, "/transactions/"+id, bytes.NewReader(ub)))
    if w3.Code != http.StatusOK { t.Fatalf("expected 200, got %d", w3.Code) }

    w4 := httptest.NewRecorder()
    r.ServeHTTP(w4, httptest.NewRequest(http.MethodDelete, "/transactions/"+id, nil))
    if w4.Code != http.StatusNoContent { t.Fatalf("expected 204, got %d", w4.Code) }

    w5 := httptest.NewRecorder()
    r.ServeHTTP(w5, httptest.NewRequest(http.MethodGet, "/transactions/"+id, nil))
    if w5.Code != http.StatusNotFound { t.Fatalf("expected 404, got %d", w5.Code) }

    w6 := httptest.NewRecorder()
    r.ServeHTTP(w6, httptest.NewRequest(http.MethodGet, "/transactions/not-a-uuid", nil))
    if w6.Code != http.StatusNotFound { t.Fatalf("expected 404, got %d", w6.Code) }

    w7 := httptest.NewRecorder()
    r.ServeHTTP(w7, httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewReader([]byte("bad"))))
    if w7.Code != http.StatusBadRequest { t.Fatalf("expected 400, got %d", w7.Code) }
}