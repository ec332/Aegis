package main

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
    "time"

    market "github.com/aegis/proto/gen/market"
    "github.com/redis/go-redis/v9"
    "github.com/stretchr/testify/require"
    "google.golang.org/grpc"
    miniredis "github.com/alicebob/miniredis/v2"
)

type fakeMarketClient struct{
    market.MarketServiceClient
    get func(ctx context.Context, in *market.GetMarketRequest) (*market.GetMarketResponse, error)
    list func(ctx context.Context, in *market.ListMarketsRequest) (*market.ListMarketsResponse, error)
    update func(ctx context.Context, in *market.UpdateMarketRequest) (*market.UpdateMarketResponse, error)
}
func (f *fakeMarketClient) GetMarket(ctx context.Context, in *market.GetMarketRequest, opts ...grpc.CallOption) (*market.GetMarketResponse, error) { return f.get(ctx, in) }
func (f *fakeMarketClient) ListMarkets(ctx context.Context, in *market.ListMarketsRequest, opts ...grpc.CallOption) (*market.ListMarketsResponse, error) { return f.list(ctx, in) }
func (f *fakeMarketClient) UpdateMarket(ctx context.Context, in *market.UpdateMarketRequest, opts ...grpc.CallOption) (*market.UpdateMarketResponse, error) { return f.update(ctx, in) }

func newTestGateway(t *testing.T) *APIGateway {
    t.Helper()
    mr, err := miniredis.Run()
    require.NoError(t, err)
    t.Cleanup(mr.Close)
    r := redis.NewClient(&redis.Options{Addr: mr.Addr()})
    g := &APIGateway{redis: r}
    return g
}

func TestCacheHelpers(t *testing.T) {
    g := newTestGateway(t)
    ctx := context.Background()
    key := "test:json:key"
    data := map[string]string{"a":"b"}
    g.setCacheJSON(ctx, key, data, 2*time.Second)
    var out map[string]string
    ok := g.getCacheJSON(ctx, key, &out)
    require.True(t, ok)
    require.Equal(t, data, out)
}

func TestGetMarketCaching(t *testing.T) {
    g := newTestGateway(t)
    // fake client
    mid := "m-1"
    g.marketStub = &fakeMarketClient{get: func(ctx context.Context, in *market.GetMarketRequest) (*market.GetMarketResponse, error) {
        return &market.GetMarketResponse{Market: &market.Market{Id: mid, Question: "Q", Description: "D", Status: "active"}}, nil
    }}
    req := httptest.NewRequest("GET", "/api/markets/"+mid, nil)
    w := httptest.NewRecorder()
    g.getMarket(context.Background(), w, req)
    res := w.Result()
    require.Equal(t, http.StatusOK, res.StatusCode)
    // verify cached
    raw, err := g.redis.Get(context.Background(), "market:"+mid+":summary").Bytes()
    require.NoError(t, err)
    var cached map[string]interface{}
    require.NoError(t, json.Unmarshal(raw, &cached))
    require.Equal(t, mid, cached["market"].(map[string]interface{})["id"])
}

func TestUpdateMarketInvalidation(t *testing.T) {
    g := newTestGateway(t)
    mid := "m-2"
    // seed cache entries
    g.setCacheJSON(context.Background(), "market:"+mid+":summary", map[string]string{"x":"y"}, 60*time.Second)
    g.setCacheJSON(context.Background(), "markets:list:page:1:20", map[string]string{"p":"1"}, 30*time.Second)
    // fake update
    g.marketStub = &fakeMarketClient{update: func(ctx context.Context, in *market.UpdateMarketRequest) (*market.UpdateMarketResponse, error) {
        return &market.UpdateMarketResponse{Market: &market.Market{Id: mid, Question: "Q"}}, nil
    }}
    body := strings.NewReader(`{"status":"active"}`)
    req := httptest.NewRequest("PUT", "/api/markets/"+mid, body)
    w := httptest.NewRecorder()
    g.updateMarket(context.Background(), w, req)
    require.Equal(t, http.StatusOK, w.Result().StatusCode)
    // keys should be gone or expired
    _, err1 := g.redis.Get(context.Background(), "market:"+mid+":summary").Result()
    require.Error(t, err1)
}
