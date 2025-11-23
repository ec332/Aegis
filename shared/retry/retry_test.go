package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestExecute_Success(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultConfig()
	config.MaxAttempts = 3

	attemptCount := 0
	fn := func(ctx context.Context) error {
		attemptCount++
		return nil
	}

	ctx := context.Background()
	err := Execute(ctx, config, logger, fn)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt, got: %d", attemptCount)
	}
}

func TestExecute_RetrySuccess(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultConfig()
	config.MaxAttempts = 3
	config.InitialDelay = 10 * time.Millisecond

	attemptCount := 0
	fn := func(ctx context.Context) error {
		attemptCount++
		if attemptCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	ctx := context.Background()
	err := Execute(ctx, config, logger, fn)

	if err != nil {
		t.Errorf("Expected no error after retry, got: %v", err)
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got: %d", attemptCount)
	}
}

func TestExecute_MaxRetriesExceeded(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultConfig()
	config.MaxAttempts = 2
	config.InitialDelay = 10 * time.Millisecond

	attemptCount := 0
	testErr := errors.New("persistent error")
	fn := func(ctx context.Context) error {
		attemptCount++
		return testErr
	}

	ctx := context.Background()
	err := Execute(ctx, config, logger, fn)

	if err == nil {
		t.Error("Expected error after max retries")
	}

	if !errors.Is(err, ErrMaxRetriesExceeded) {
		t.Errorf("Expected ErrMaxRetriesExceeded, got: %v", err)
	}

	if attemptCount != 2 {
		t.Errorf("Expected 2 attempts, got: %d", attemptCount)
	}
}

func TestExecute_NonRetryableError(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultConfig()
	config.MaxAttempts = 3
	config.RetryableErrors = []error{errors.New("retryable")}

	attemptCount := 0
	nonRetryableErr := errors.New("non-retryable")
	fn := func(ctx context.Context) error {
		attemptCount++
		return nonRetryableErr
	}

	ctx := context.Background()
	err := Execute(ctx, config, logger, fn)

	if err != nonRetryableErr {
		t.Errorf("Expected non-retryable error, got: %v", err)
	}

	if attemptCount != 1 {
		t.Errorf("Expected 1 attempt for non-retryable error, got: %d", attemptCount)
	}
}

func TestExecute_ContextCancellation(t *testing.T) {
	logger := zap.NewNop()
	config := DefaultConfig()
	config.MaxAttempts = 5
	config.InitialDelay = 100 * time.Millisecond

	attemptCount := 0
	fn := func(ctx context.Context) error {
		attemptCount++
		return errors.New("error")
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	// Cancel context after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := Execute(ctx, config, logger, fn)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got: %v", err)
	}

	// Should have made at least 1 attempt but not all 5 due to cancellation
	if attemptCount >= 5 {
		t.Errorf("Expected fewer than 5 attempts due to cancellation, got: %d", attemptCount)
	}
}

func TestCalculateDelay(t *testing.T) {
	config := Config{
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        false,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 100 * time.Millisecond},
		{2, 200 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 800 * time.Millisecond},
		{5, 1 * time.Second}, // Should be capped at MaxDelay
	}

	for _, tt := range tests {
		delay := calculateDelay(tt.attempt, config)
		if delay != tt.expected {
			t.Errorf("Attempt %d: expected delay %v, got %v", tt.attempt, tt.expected, delay)
		}
	}
}

func TestIsRetryable(t *testing.T) {
	retryableErr := errors.New("retryable")
	nonRetryableErr := errors.New("non-retryable")
	
	config := Config{
		RetryableErrors: []error{retryableErr},
	}

	tests := []struct {
		err      error
		expected bool
	}{
		{nil, false},
		{retryableErr, true},
		{nonRetryableErr, false}, // When specific retryable errors are defined, others are not retryable
		{context.DeadlineExceeded, true},
	}

	for _, tt := range tests {
		result := isRetryable(tt.err, config.RetryableErrors)
		if result != tt.expected {
			t.Errorf("isRetryable(%v): expected %v, got %v", tt.err, tt.expected, result)
		}
	}
}