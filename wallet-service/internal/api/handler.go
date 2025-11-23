package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"wallet-service/internal/service"
	"wallet-service/pkg/models"
	"github.com/go-chi/chi/v5"
)

// CreateWalletAccount handles POST /wallet-accounts
func CreateWalletAccount(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req models.CreateWalletAccountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		account, err := svc.CreateWalletAccount(r.Context(), req)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Failed to create wallet account", err)
			return
		}

		respondJSON(w, http.StatusCreated, account)
	}
}

// GetWalletAccount handles GET /wallet-accounts/{accountId}
func GetWalletAccount(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accountID := chi.URLParam(r, "accountId")
		if accountID == "" {
			respondError(w, http.StatusBadRequest, "Account ID is required", nil)
			return
		}

		account, err := svc.GetWalletAccount(r.Context(), accountID)
		if err != nil {
			respondError(w, http.StatusNotFound, "Wallet account not found", err)
			return
		}

		respondJSON(w, http.StatusOK, account)
	}
}

// GetWalletAccountByUserID handles GET /wallet-accounts/user/{userId}
func GetWalletAccountByUserID(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := chi.URLParam(r, "userId")
		if userID == "" {
			respondError(w, http.StatusBadRequest, "User ID is required", nil)
			return
		}

		currency := r.URL.Query().Get("currency")
		if currency == "" {
			currency = string(models.CurrencyUSDC) // Default to USDC
		}

		account, err := svc.GetWalletAccountByUserID(r.Context(), userID, models.Currency(currency))
		if err != nil {
			respondError(w, http.StatusNotFound, "Wallet account not found", err)
			return
		}

		respondJSON(w, http.StatusOK, account)
	}
}

// Deposit handles POST /wallet-accounts/{accountId}/deposit
func Deposit(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accountID := chi.URLParam(r, "accountId")
		if accountID == "" {
			respondError(w, http.StatusBadRequest, "Account ID is required", nil)
			return
		}

		var req models.DepositRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		transaction, err := svc.Deposit(r.Context(), accountID, req)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Failed to process deposit", err)
			return
		}

		respondJSON(w, http.StatusCreated, transaction)
	}
}

// Withdrawal handles POST /wallet-accounts/{accountId}/withdrawal
func Withdrawal(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accountID := chi.URLParam(r, "accountId")
		if accountID == "" {
			respondError(w, http.StatusBadRequest, "Account ID is required", nil)
			return
		}

		var req models.WithdrawalRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		transaction, err := svc.Withdrawal(r.Context(), accountID, req)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Failed to process withdrawal", err)
			return
		}

		respondJSON(w, http.StatusCreated, transaction)
	}
}

// DebitWallet handles POST /wallet-accounts/{accountId}/debit
func DebitWallet(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accountID := chi.URLParam(r, "accountId")
		if accountID == "" {
			respondError(w, http.StatusBadRequest, "Account ID is required", nil)
			return
		}

		var req models.DebitWalletRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		transaction, err := svc.DebitWallet(r.Context(), accountID, req)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Failed to debit wallet", err)
			return
		}

		respondJSON(w, http.StatusCreated, transaction)
	}
}

// CreditWallet handles POST /wallet-accounts/{accountId}/credit
func CreditWallet(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accountID := chi.URLParam(r, "accountId")
		if accountID == "" {
			respondError(w, http.StatusBadRequest, "Account ID is required", nil)
			return
		}

		var req models.CreditWalletRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body", err)
			return
		}

		transaction, err := svc.CreditWallet(r.Context(), accountID, req)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Failed to credit wallet", err)
			return
		}

		respondJSON(w, http.StatusCreated, transaction)
	}
}

// GetWalletTransactions handles GET /wallet-accounts/{accountId}/transactions
func GetWalletTransactions(svc *service.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accountID := chi.URLParam(r, "accountId")
		if accountID == "" {
			respondError(w, http.StatusBadRequest, "Account ID is required", nil)
			return
		}

		transactions, err := svc.GetWalletTransactions(r.Context(), accountID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to get wallet transactions", err)
			return
		}

		respondJSON(w, http.StatusOK, transactions)
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