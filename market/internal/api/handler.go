package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"github.com/ec332/aegis/market/internal/service"
	"github.com/ec332/aegis/market/pkg/models"
	"github.com/go-chi/chi/v5"
)

// CreateMarket handles POST /markets
func CreateMarket(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.CreateMarketRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		market, err := svc.CreateMarket(r.Context(), req)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Failed to create market", err)
			return
		}

		respondJSON(w, http.StatusCreated, market)
	}
}

// ListMarkets handles GET /markets
func ListMarkets(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Optional status filter
		var status *models.MarketStatus
		statusParam := r.URL.Query().Get("status")
		if statusParam != "" {
			s := models.MarketStatus(statusParam)
			status = &s
		}

		markets, err := svc.ListMarkets(r.Context(), status)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to list markets", err)
			return
		}

		response := models.MarketListResponse{
			Markets: markets,
			Total:   len(markets),
		}

		respondJSON(w, http.StatusOK, response)
	}
}

// GetMarket handles GET /markets/:marketId
func GetMarket(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		marketID := chi.URLParam(r, "marketId")
		if marketID == "" {
			respondError(w, http.StatusBadRequest, "Market ID is required", nil)
			return
		}

		market, err := svc.GetMarket(r.Context(), marketID)
		if err != nil {
			respondError(w, http.StatusNotFound, "Market not found", err)
			return
		}

		respondJSON(w, http.StatusOK, market)
	}
}

// UpdateMarket handles PUT /markets/:marketId
func UpdateMarket(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		marketID := chi.URLParam(r, "marketId")
		if marketID == "" {
			respondError(w, http.StatusBadRequest, "Market ID is required", nil)
			return
		}

		var req models.UpdateMarketRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		market, err := svc.UpdateMarket(r.Context(), marketID, req)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Failed to update market", err)
			return
		}

		respondJSON(w, http.StatusOK, market)
	}
}

// StreamLiquidityUpdates handles GET /markets/:marketId/stream (SSE for liquidity pool updates)
func StreamLiquidityUpdates(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		marketID := chi.URLParam(r, "marketId")
		if marketID == "" {
			respondError(w, http.StatusBadRequest, "Market ID is required", nil)
			return
		}

		// Verify market exists
		_, err := svc.GetMarket(r.Context(), marketID)
		if err != nil {
			respondError(w, http.StatusNotFound, "Market not found", err)
			return
		}

		// Set SSE headers
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Get flusher
		flusher, ok := w.(http.Flusher)
		if !ok {
			respondError(w, http.StatusInternalServerError, "Streaming not supported", nil)
			return
		}

		// Subscribe to Redis updates
		updatesCh, err := svc.SubscribeToLiquidityUpdates(r.Context(), marketID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to subscribe to updates", err)
			return
		}

		// Send initial connection message
		fmt.Fprintf(w, "event: connected\ndata: {\"market_id\":\"%s\",\"timestamp\":\"%s\"}\n\n", marketID, time.Now().Format(time.RFC3339))
		flusher.Flush()

		// Keepalive ticker
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case update, ok := <-updatesCh:
				if !ok {
					return
				}
				data, err := json.Marshal(update)
				if err != nil {
					fmt.Printf("Error marshaling update: %v\n", err)
					continue
				}
				fmt.Fprintf(w, "event: liquidity-update\ndata: %s\n\n", data)
				flusher.Flush()
			case <-ticker.C:
				// Keepalive ping
				fmt.Fprintf(w, "event: ping\ndata: {\"timestamp\":\"%s\"}\n\n", time.Now().Format(time.RFC3339))
				flusher.Flush()
			}
		}
	}
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		fmt.Printf("Error encoding JSON response: %v\n", err)
	}
}

func respondError(w http.ResponseWriter, status int, message string, err error) {
	errResp := models.ErrorResponse{
		Error: message,
	}
	if err != nil {
		errResp.Message = err.Error()
	}
	respondJSON(w, status, errResp)
}
