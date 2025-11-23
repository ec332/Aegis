package utils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger creates a configured zap logger
func NewLogger(serviceName string) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.InitialFields = map[string]interface{}{
		"service": serviceName,
	}
	
	// Use development config if in development mode
	if os.Getenv("ENVIRONMENT") == "development" {
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}
	
	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	
	return logger, nil
}

// ContextWithTimeout creates a context with timeout and proper error handling
func ContextWithTimeout(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, timeout)
}

// RetryableError wraps an error to indicate it's retryable
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error: %v", e.Err)
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

func NewRetryableError(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err}
}

func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	
	var retryable *RetryableError
	return errors.As(err, &retryable)
}

// ServiceError represents a service-level error with additional context
type ServiceError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *ServiceError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewServiceError(code, message string) *ServiceError {
	return &ServiceError{
		Code:    code,
		Message: message,
	}
}

func NewServiceErrorWithDetails(code, message, details string) *ServiceError {
	return &ServiceError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// Common error codes
const (
	ErrCodeInternal        = "INTERNAL"
	ErrCodeInvalidRequest  = "INVALID_REQUEST"
	ErrCodeNotFound        = "NOT_FOUND"
	ErrCodeAlreadyExists   = "ALREADY_EXISTS"
	ErrCodeUnauthorized    = "UNAUTHORIZED"
	ErrCodeTimeout         = "TIMEOUT"
	ErrCodeServiceUnavail  = "SERVICE_UNAVAILABLE"
	ErrCodeCircuitOpen     = "CIRCUIT_BREAKER_OPEN"
)

// TimeoutError creates a timeout error with context
func TimeoutError(operation string, timeout time.Duration) error {
	return NewServiceErrorWithDetails(
		ErrCodeTimeout,
		fmt.Sprintf("operation %s timed out after %v", operation, timeout),
		operation,
	)
}

// GetEnvOrDefault gets an environment variable or returns a default value
func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvOrDefaultDuration gets an environment variable as duration or returns default
func GetEnvOrDefaultDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}