package service

import (
    "context"
    "fmt"
    "time"
    "strings"

    "aegis/wallet/internal/repository"
    "aegis/wallet/pkg/models"

    "github.com/google/uuid"
    "go.uber.org/zap"
    "github.com/ethereum/go-ethereum/accounts"
    "github.com/ethereum/go-ethereum/common/hexutil"
    "github.com/ethereum/go-ethereum/crypto"
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

// RequestNonce generates and persists a nonce for the given wallet address
func (s *Service) RequestNonce(ctx context.Context, wallet string) (string, error) {
    if wallet == "" {
        return "", fmt.Errorf("wallet is required")
    }
    nonce := fmt.Sprintf("Login verification: %s", uuid.New().String())
    _, err := s.repo.GetUserByWalletAddress(ctx, wallet)
    if err != nil {
        now := time.Now()
        userID := uuid.New().String()
        u := &models.User{ID: userID, WalletAddress: wallet, Balance: 0, Nonce: nonce, Role: models.UserRoleUser, CreatedAt: now, UpdatedAt: now}
        if err2 := s.repo.CreateUser(ctx, u); err2 != nil {
            return "", fmt.Errorf("create user: %w", err2)
        }
        return nonce, nil
    }
    if err := s.repo.SetNonceByWallet(ctx, wallet, nonce); err != nil {
        return "", fmt.Errorf("set nonce: %w", err)
    }
    return nonce, nil
}

// VerifySignature verifies an Ethereum signature and returns JWT if valid
func (s *Service) VerifySignature(ctx context.Context, wallet string, signature string, tokenGen func(wallet string) (string, error)) (string, error) {
    if wallet == "" || signature == "" {
        return "", fmt.Errorf("wallet and signature are required")
    }
    user, err := s.repo.GetUserByWalletAddress(ctx, wallet)
    if err != nil {
        return "", fmt.Errorf("user not found")
    }
    msg := []byte(user.Nonce)
    hash := accounts.TextHash(msg)
    sig, err := hexutil.Decode(signature)
    if err != nil {
        return "", fmt.Errorf("invalid signature")
    }
    if len(sig) != 65 {
        return "", fmt.Errorf("invalid signature length")
    }
    if sig[64] >= 27 {
        sig[64] -= 27
    }
    pubKey, err := crypto.SigToPub(hash, sig)
    if err != nil {
        return "", fmt.Errorf("recover failed")
    }
    recovered := crypto.PubkeyToAddress(*pubKey).Hex()
    if !equalEthAddr(recovered, wallet) {
        return "", fmt.Errorf("signature mismatch")
    }
    if err := s.repo.SetLastLoginByWallet(ctx, wallet, time.Now()); err != nil {
        return "", fmt.Errorf("update last_login: %w", err)
    }
    token, err := tokenGen(wallet)
    if err != nil {
        return "", fmt.Errorf("token generation failed")
    }
    return token, nil
}

func equalEthAddr(a, b string) bool {
    aa := strings.ToLower(strings.TrimSpace(a))
    bb := strings.ToLower(strings.TrimSpace(b))
    if !strings.HasPrefix(aa, "0x") {
        aa = "0x" + aa
    }
    if !strings.HasPrefix(bb, "0x") {
        bb = "0x" + bb
    }
    return aa == bb
}
