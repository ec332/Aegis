package service

import (
	"context"
	"fmt"
	"time"

	"aegis/wallet/internal/repository"
	"aegis/wallet/pkg/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Service handles business logic for wallet operations
type Service struct {
	repo   *repository.Repository
	logger *zap.Logger
}

// New creates a new service instance
func New(repo *repository.Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

// CreateUser creates a new user
func (s *Service) CreateUser(ctx context.Context, req models.CreateUserRequest) (*models.User, error) {
	if req.WalletAddress == "" {
		return nil, fmt.Errorf("wallet_address is required")
	}
	if req.Balance < 0 {
		return nil, fmt.Errorf("balance cannot be negative")
	}

	now := time.Now()
	userID := uuid.New().String()
	nonce := uuid.New().String() // Generate a random nonce

	user := &models.User{
		ID:            userID,
		WalletAddress: req.WalletAddress,
		Balance:       req.Balance,
		Nonce:         nonce,
		Role:          models.UserRoleUser, // Default to user role
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return user, nil
}

// GetUser retrieves a user by ID
func (s *Service) GetUser(ctx context.Context, userID string) (*models.User, error) {
	user, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return user, nil
}

// GetUserByWalletAddress retrieves a user by wallet address
func (s *Service) GetUserByWalletAddress(ctx context.Context, walletAddress string) (*models.User, error) {
	user, err := s.repo.GetUserByWalletAddress(ctx, walletAddress)
	if err != nil {
		return nil, fmt.Errorf("get user by wallet: %w", err)
	}
	return user, nil
}

// UpdateUser updates a user and returns the updated user
func (s *Service) UpdateUser(ctx context.Context, userID string, req models.UpdateUserRequest) (*models.User, error) {
	if err := s.repo.UpdateUser(ctx, userID, req); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	// Return the updated user
	return s.GetUser(ctx, userID)
}
