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

// CreateUser handles POST /users
func CreateUser(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		user, err := svc.CreateUser(r.Context(), req)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Failed to create user", err)
			return
		}

		respondJSON(w, http.StatusCreated, user)
	}
}

// GetUser handles GET /users/{userId}
func GetUser(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "userId")
		if userID == "" {
			respondError(w, http.StatusBadRequest, "User ID is required", nil)
			return
		}

		user, err := svc.GetUser(r.Context(), userID)
		if err != nil {
			respondError(w, http.StatusNotFound, "User not found", err)
			return
		}

		respondJSON(w, http.StatusOK, user)
	}
}

// GetUserByWallet handles GET /users/wallet/{walletAddress}
func GetUserByWallet(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walletAddress := chi.URLParam(r, "walletAddress")
		if walletAddress == "" {
			respondError(w, http.StatusBadRequest, "Wallet address is required", nil)
			return
		}

		user, err := svc.GetUserByWalletAddress(r.Context(), walletAddress)
		if err != nil {
			respondError(w, http.StatusNotFound, "User not found", err)
			return
		}

		respondJSON(w, http.StatusOK, user)
	}
}

// UpdateUser handles PUT /users/{userId}
func UpdateUser(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "userId")
		if userID == "" {
			respondError(w, http.StatusBadRequest, "User ID is required", nil)
			return
		}

		var req models.UpdateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		user, err := svc.UpdateUser(r.Context(), userID, req)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Failed to update user", err)
			return
		}

		respondJSON(w, http.StatusOK, user)
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
					// Note: In a real implementation, you would want to log this error
				// but since this is a streaming response, we can't easily access the logger here
				// For now, we'll skip the error silently to avoid breaking the stream
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

// GetMarketPrices handles GET /markets/:marketId/prices
func GetMarketPrices(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		marketID := chi.URLParam(r, "marketId")
		if marketID == "" {
			respondError(w, http.StatusBadRequest, "Market ID is required", nil)
			return
		}

		prices, err := svc.GetMarketPrices(r.Context(), marketID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to get market prices", err)
			return
		}

		respondJSON(w, http.StatusOK, prices)
	}
}

// CalculateBuyCost handles POST /markets/:marketId/options/:optionId/cost/buy
func CalculateBuyCost(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		marketID := chi.URLParam(r, "marketId")
		optionID := chi.URLParam(r, "optionId")
		if marketID == "" || optionID == "" {
			respondError(w, http.StatusBadRequest, "Market ID and Option ID are required", nil)
			return
		}

		var req models.CostCalculationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		cost, err := svc.CalculateBuyCost(r.Context(), marketID, optionID, req.Amount)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Failed to calculate buy cost", err)
			return
		}

		respondJSON(w, http.StatusOK, models.CostCalculationResponse{Cost: cost})
	}
}

// CalculateSellCost handles POST /markets/:marketId/options/:optionId/cost/sell
func CalculateSellCost(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		marketID := chi.URLParam(r, "marketId")
		optionID := chi.URLParam(r, "optionId")
		if marketID == "" || optionID == "" {
			respondError(w, http.StatusBadRequest, "Market ID and Option ID are required", nil)
			return
		}

		var req models.CostCalculationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		cost, err := svc.CalculateSellCost(r.Context(), marketID, optionID, req.Amount)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Failed to calculate sell cost", err)
			return
		}

		respondJSON(w, http.StatusOK, models.CostCalculationResponse{Cost: cost})
	}
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Note: In a real implementation, you would want to log this error
		// but since this is a generic response function, we don't have access to logger here
		// Consider adding logger parameter or using a global logger in production
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
