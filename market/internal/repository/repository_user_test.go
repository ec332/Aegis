package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ec332/aegis/market/pkg/models"
	"github.com/google/uuid"
)

func TestUserOperations(t *testing.T) {
	// Setup test database connection
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repo := New(db)
	ctx := context.Background()

	// Initialize schema
	if err := repo.InitSchema(ctx); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	t.Run("CreateUser", func(t *testing.T) {
		user := &models.User{
			ID:            uuid.New().String(),
			WalletAddress: "0x742d35Cc6634C0532925a3b8D0e0C8d4A3F8d4e2",
			Balance:       1000.50,
			Nonce:         uuid.New().String(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.CreateUser(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	})

	t.Run("GetUser", func(t *testing.T) {
		// Create a test user first
		user := &models.User{
			ID:            uuid.New().String(),
			WalletAddress: "0x1234567890123456789012345678901234567890",
			Balance:       500.25,
			Nonce:         uuid.New().String(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.CreateUser(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Test GetUser
		retrievedUser, err := repo.GetUser(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if retrievedUser.ID != user.ID {
			t.Errorf("Expected user ID %s, got %s", user.ID, retrievedUser.ID)
		}
		if retrievedUser.WalletAddress != user.WalletAddress {
			t.Errorf("Expected wallet address %s, got %s", user.WalletAddress, retrievedUser.WalletAddress)
		}
		if retrievedUser.Balance != user.Balance {
			t.Errorf("Expected balance %f, got %f", user.Balance, retrievedUser.Balance)
		}
	})

	t.Run("GetUserByWalletAddress", func(t *testing.T) {
		// Create a test user first
		user := &models.User{
			ID:            uuid.New().String(),
			WalletAddress: "0x9876543210987654321098765432109876543210",
			Balance:       750.75,
			Nonce:         uuid.New().String(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.CreateUser(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Test GetUserByWalletAddress
		retrievedUser, err := repo.GetUserByWalletAddress(ctx, user.WalletAddress)
		if err != nil {
			t.Fatalf("Failed to get user by wallet address: %v", err)
		}

		if retrievedUser.ID != user.ID {
			t.Errorf("Expected user ID %s, got %s", user.ID, retrievedUser.ID)
		}
		if retrievedUser.WalletAddress != user.WalletAddress {
			t.Errorf("Expected wallet address %s, got %s", user.WalletAddress, retrievedUser.WalletAddress)
		}
	})

	t.Run("UpdateUser", func(t *testing.T) {
		// Create a test user first
		user := &models.User{
			ID:            uuid.New().String(),
			WalletAddress: "0x1111111111111111111111111111111111111111",
			Balance:       100.00,
			Nonce:         uuid.New().String(),
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}

		err := repo.CreateUser(ctx, user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Update user
		newBalance := 200.50
		newNonce := uuid.New().String()
		updates := models.UpdateUserRequest{
			Balance: &newBalance,
			Nonce:   &newNonce,
		}

		err = repo.UpdateUser(ctx, user.ID, updates)
		if err != nil {
			t.Fatalf("Failed to update user: %v", err)
		}

		// Verify update
		updatedUser, err := repo.GetUser(ctx, user.ID)
		if err != nil {
			t.Fatalf("Failed to get updated user: %v", err)
		}

		if updatedUser.Balance != newBalance {
			t.Errorf("Expected balance %f, got %f", newBalance, updatedUser.Balance)
		}
		if updatedUser.Nonce != newNonce {
			t.Errorf("Expected nonce %s, got %s", newNonce, updatedUser.Nonce)
		}
	})

	t.Run("GetNonExistentUser", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		_, err := repo.GetUser(ctx, nonExistentID)
		if err == nil {
			t.Error("Expected error for non-existent user, got nil")
		}
	})

	t.Run("GetNonExistentUserByWallet", func(t *testing.T) {
		nonExistentWallet := "0x9999999999999999999999999999999999999999"
		_, err := repo.GetUserByWalletAddress(ctx, nonExistentWallet)
		if err == nil {
			t.Error("Expected error for non-existent user wallet, got nil")
		}
	})

	t.Run("UpdateNonExistentUser", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		newBalance := 50.00
		updates := models.UpdateUserRequest{
			Balance: &newBalance,
		}
		err := repo.UpdateUser(ctx, nonExistentID, updates)
		if err == nil {
			t.Error("Expected error for updating non-existent user, got nil")
		}
	})
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Use environment variables for test database connection
	dbHost := os.Getenv("TEST_DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("TEST_DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbName := os.Getenv("TEST_DB_NAME")
	if dbName == "" {
		dbName = "transaction_test"
	}
	dbUser := os.Getenv("TEST_DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPassword := os.Getenv("TEST_DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "postgres"
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	cleanup := func() {
		// Clean up test data
		db.Exec("DROP TABLE IF EXISTS users CASCADE")
		db.Exec("DROP TABLE IF EXISTS liquidity_pool CASCADE")
		db.Exec("DROP TABLE IF EXISTS options CASCADE")
		db.Exec("DROP TABLE IF EXISTS markets CASCADE")
		db.Close()
	}

	return db, cleanup
}