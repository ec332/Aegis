package retry

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"

	"go.uber.org/zap"
)

type Config struct {
	MaxAttempts     int           // Maximum number of retry attempts
	InitialDelay    time.Duration // Initial delay between retries
	MaxDelay        time.Duration // Maximum delay between retries
	BackoffFactor   float64       // Exponential backoff factor
	Jitter          bool          // Add jitter to delays
	RetryableErrors []error       // Errors that should trigger retry
}

func DefaultConfig() Config {
	return Config{
		MaxAttempts:     3,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        30 * time.Second,
		BackoffFactor:   2.0,
		Jitter:          true,
		RetryableErrors: []error{},
	}
}

type Retryable func(ctx context.Context) error

func Execute(ctx context.Context, config Config, logger *zap.Logger, fn Retryable) error {
	var lastErr error
	
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay := calculateDelay(attempt, config)
			logger.Debug("retrying after delay",
				zap.Int("attempt", attempt+1),
				zap.Duration("delay", delay),
				zap.Error(lastErr))
			
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		
		err := fn(ctx)
		if err == nil {
			if attempt > 0 {
				logger.Info("retry succeeded",
					zap.Int("attempts", attempt+1))
			}
			return nil
		}
		
		lastErr = err
		
		// Check if error is retryable
		if !isRetryable(err, config.RetryableErrors) {
			logger.Debug("non-retryable error",
				zap.Error(err),
				zap.Int("attempt", attempt+1))
			return err
		}
		
		logger.Debug("retryable error occurred",
			zap.Error(err),
			zap.Int("attempt", attempt+1),
			zap.Int("max_attempts", config.MaxAttempts))
	}
	
	logger.Warn("max retries exceeded",
		zap.Error(lastErr),
		zap.Int("attempts", config.MaxAttempts))
	
	return errors.Join(ErrMaxRetriesExceeded, lastErr)
}

func calculateDelay(attempt int, config Config) time.Duration {
	delay := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt-1))
	
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}
	
	if config.Jitter {
		// Add jitter (±25% randomization)
		jitter := rand.Float64()*0.5 - 0.25
		delay = delay * (1 + jitter)
	}
	
	return time.Duration(delay)
}

func isRetryable(err error, retryableErrors []error) bool {
	if err == nil {
		return false
	}
	
	// Check against specific retryable errors
	for _, retryableErr := range retryableErrors {
		if errors.Is(err, retryableErr) {
			return true
		}
	}
	
	// Default retryable errors
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	
	// If specific retryable errors are defined, only retry those
	if len(retryableErrors) > 0 {
		return false
	}
	
	return true // Default to retrying unknown errors if no specific list provided
}

var ErrMaxRetriesExceeded = errors.New("maximum retry attempts exceeded")