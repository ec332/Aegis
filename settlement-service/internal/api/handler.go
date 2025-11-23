package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"settlement-service/internal/service"
	"settlement-service/pkg/models"
)

type Handler struct {
	settlementService *service.SettlementService
}

func NewHandler(settlementService *service.SettlementService) *Handler {
	return &Handler{settlementService: settlementService}
}

func (h *Handler) CreateSettlement(w http.ResponseWriter, r *http.Request) {
	var req models.SettlementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	response, err := h.settlementService.CreateSettlement(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetSettlement(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Settlement ID is required", http.StatusBadRequest)
		return
	}

	settlement, err := h.settlementService.GetSettlement(id)
	if err != nil {
		if err.Error() == "settlement not found" {
			http.Error(w, "Settlement not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settlement)
}

func (h *Handler) GetSettlementByMarketID(w http.ResponseWriter, r *http.Request) {
	marketID := chi.URLParam(r, "marketId")
	if marketID == "" {
		http.Error(w, "Market ID is required", http.StatusBadRequest)
		return
	}

	settlement, err := h.settlementService.GetSettlementByMarketID(marketID)
	if err != nil {
		if err.Error() == "settlement not found" {
			http.Error(w, "Settlement not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settlement)
}

func (h *Handler) CompleteSettlement(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "Settlement ID is required", http.StatusBadRequest)
		return
	}

	if err := h.settlementService.CompleteSettlement(id); err != nil {
		if err.Error() == "settlement not found" {
			http.Error(w, "Settlement not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Settlement completed successfully",
	})
}