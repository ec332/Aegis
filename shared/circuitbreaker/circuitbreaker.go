package circuitbreaker

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

type Config struct {
	FailureThreshold   int           // Number of failures before opening circuit
	SuccessThreshold   int           // Number of successes before closing from half-open
	Timeout            time.Duration // Time before attempting to close from open
	MaxConcurrentCalls int           // Maximum concurrent calls allowed
}

type CircuitBreaker struct {
	name       string
	config     Config
	state      State
	failures   int
	successes  int
	lastFailTime time.Time
	calls      int
	mu         sync.RWMutex
	logger     *zap.Logger
}

func NewCircuitBreaker(name string, config Config, logger *zap.Logger) *CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 2
	}
	if config.Timeout <= 0 {
		config.Timeout = 60 * time.Second
	}
	if config.MaxConcurrentCalls <= 0 {
		config.MaxConcurrentCalls = 100
	}

	return &CircuitBreaker{
		name:   name,
		config: config,
		state:  StateClosed,
		logger: logger,
	}
}

func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if !cb.allowRequest() {
		return ErrCircuitOpen
	}

	defer cb.onCallFinished()

	err := fn()
	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return cb.calls < cb.config.MaxConcurrentCalls
	case StateOpen:
		if time.Since(cb.lastFailTime) > cb.config.Timeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

func (cb *CircuitBreaker) onCallFinished() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.calls--
}

func (cb *CircuitBreaker) onSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.successes++
	cb.failures = 0

	switch cb.state {
	case StateHalfOpen:
		if cb.successes >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.logger.Info("circuit breaker closed",
				zap.String("circuit", cb.name),
				zap.String("state", cb.state.String()))
		}
	case StateClosed:
		// Reset failures on success in closed state
		cb.failures = 0
	}
}

func (cb *CircuitBreaker) onFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.successes = 0
	cb.lastFailTime = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.config.FailureThreshold {
			cb.state = StateOpen
			cb.logger.Warn("circuit breaker opened",
				zap.String("circuit", cb.name),
				zap.Int("failures", cb.failures),
				zap.String("state", cb.state.String()))
		}
	case StateHalfOpen:
		cb.state = StateOpen
		cb.logger.Warn("circuit breaker reopened from half-open",
			zap.String("circuit", cb.name),
			zap.String("state", cb.state.String()))
	}
}

func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

func (cb *CircuitBreaker) GetStats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return Stats{
		State:     cb.state.String(),
		Failures:  cb.failures,
		Successes: cb.successes,
		Calls:     cb.calls,
	}
}

type Stats struct {
	State     string `json:"state"`
	Failures  int    `json:"failures"`
	Successes int    `json:"successes"`
	Calls     int    `json:"calls"`
}

var ErrCircuitOpen = errors.New("circuit breaker is open")