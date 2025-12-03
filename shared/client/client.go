package client

import (
    "context"
    "fmt"
    "time"

	"github.com/aegis/shared/circuitbreaker"
	"github.com/aegis/shared/kafka"
	"github.com/aegis/shared/retry"
	"github.com/aegis/shared/utils"
	"go.uber.org/zap"
)

// ServiceClient provides a resilient client for inter-service communication
type ServiceClient struct {
    name           string
    baseURL        string
    circuitBreaker *circuitbreaker.CircuitBreaker
    kafkaProducer  *kafka.Producer
    logger         *zap.Logger
    httpClient     *HTTPClient // We'll implement this separately
}

// ClientConfig holds configuration for the service client
type ClientConfig struct {
	ServiceName      string
	BaseURL          string
	Timeout          time.Duration
	MaxRetries       int
	CircuitBreaker   circuitbreaker.Config
	KafkaFallback    bool
	KafkaConfig      kafka.Config
}

// DefaultClientConfig returns a default configuration
func DefaultClientConfig(serviceName, baseURL string) ClientConfig {
	return ClientConfig{
		ServiceName: serviceName,
		BaseURL:     baseURL,
		Timeout:     1 * time.Second,
		MaxRetries:  3,
		CircuitBreaker: circuitbreaker.Config{
			FailureThreshold:   5,
			SuccessThreshold:   2,
			Timeout:            60 * time.Second,
			MaxConcurrentCalls: 100,
		},
		KafkaFallback: true,
		KafkaConfig:   kafka.DefaultConfig(),
	}
}

// NewServiceClient creates a new resilient service client
func NewServiceClient(config ClientConfig, logger *zap.Logger) (*ServiceClient, error) {
	// Create HTTP client
	httpClient, err := NewHTTPClient(config.Timeout, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Create circuit breaker
	cb := circuitbreaker.NewCircuitBreaker(config.ServiceName, config.CircuitBreaker, logger)

	// Create Kafka producer if fallback is enabled
	var kafkaProducer *kafka.Producer
	if config.KafkaFallback {
		kafkaProducer = kafka.NewProducer(config.KafkaConfig, logger)
	}

    return &ServiceClient{
        name:           config.ServiceName,
        baseURL:        config.BaseURL,
        circuitBreaker: cb,
        kafkaProducer:  kafkaProducer,
        logger:         logger,
        httpClient:     httpClient,
    }, nil
}

// Call makes a resilient HTTP call to another service
func (c *ServiceClient) Call(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	// Create a retryable function
	retryable := func(ctx context.Context) error {
		return c.callWithCircuitBreaker(ctx, method, path, body, result)
	}

	// Configure retry
	retryConfig := retry.DefaultConfig()
	retryConfig.MaxAttempts = 3
	retryConfig.RetryableErrors = []error{
		circuitbreaker.ErrCircuitOpen,
		utils.ErrServiceUnavailable,
	}

	// Execute with retry
	return retry.Execute(ctx, retryConfig, c.logger, retryable)
}

func (c *ServiceClient) callWithCircuitBreaker(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	return c.circuitBreaker.Execute(ctx, func() error {
		return c.callWithTimeout(ctx, method, path, body, result)
	})
}

func (c *ServiceClient) callWithTimeout(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

    // Make HTTP call
    err := c.httpClient.Call(ctx, method, c.baseURL+path, body, result)

	if err != nil {
		// Check if this is a timeout or service unavailable error
		if c.isRetryableError(err) {
			c.logger.Warn("HTTP call failed, will attempt Kafka fallback",
				zap.String("service", c.name),
				zap.String("method", method),
				zap.String("path", path),
				zap.Error(err))
			
			// Attempt Kafka fallback if enabled
			if c.kafkaProducer != nil {
				return c.fallbackToKafka(method, path, body, err)
			}
		}
	}

	return err
}

func (c *ServiceClient) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for timeout errors
	if utils.IsTimeoutError(err) {
		return true
	}

	// Check for service unavailable errors
	if utils.IsServiceUnavailableError(err) {
		return true
	}

	return false
}

func (c *ServiceClient) fallbackToKafka(method, path string, body interface{}, originalErr error) error {
	topic := c.getTopicForRequest(method, path)
	key := fmt.Sprintf("%s_%s_%s", c.name, method, path)
	
	message := KafkaFallbackMessage{
		Service:    c.name,
		Method:     method,
		Path:       path,
		Body:       body,
		Timestamp:  time.Now(),
		Error:      originalErr.Error(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.kafkaProducer.Publish(ctx, topic, key, message)
	if err != nil {
		c.logger.Error("failed to publish to Kafka fallback",
			zap.String("service", c.name),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("topic", topic),
			zap.Error(err))
		return fmt.Errorf("HTTP call failed and Kafka fallback also failed: %w", err)
	}

	c.logger.Info("successfully published to Kafka fallback",
		zap.String("service", c.name),
		zap.String("method", method),
		zap.String("path", path),
		zap.String("topic", topic))

    // metrics removed
	
	// Return a specific error indicating fallback was used
	return utils.NewKafkaFallbackError(originalErr)
}

func (c *ServiceClient) getTopicForRequest(method, path string) string {
	// Map HTTP methods and paths to Kafka topics
	switch c.name {
	case "market":
		if method == "POST" && path == "/markets" {
			return kafka.TopicMarketCreated
		}
		return kafka.TopicMarketUpdated
	case "wallet":
		if method == "POST" && path == "/transactions" {
			return kafka.TopicTransactionCreated
		}
		return kafka.TopicWalletUpdated
	case "settlement":
		if method == "POST" && path == "/settlements" {
			return kafka.TopicSettlementCreated
		}
		return kafka.TopicSettlementCompleted
	default:
		return kafka.TopicServiceHealth
	}
}

// Close closes the client and its resources
func (c *ServiceClient) Close() error {
	var errs []error
	
	if c.httpClient != nil {
		if err := c.httpClient.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close HTTP client: %w", err))
		}
	}
	
	if c.kafkaProducer != nil {
		if err := c.kafkaProducer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Kafka producer: %w", err))
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("errors closing service client: %v", errs)
	}
	
	return nil
}

// GetCircuitBreaker returns the circuit breaker for monitoring
func (c *ServiceClient) GetCircuitBreaker() *circuitbreaker.CircuitBreaker {
	return c.circuitBreaker
}

// GetMetrics returns the metrics for monitoring
func (c *ServiceClient) GetMetrics() *metrics.ServiceMetrics {
	return c.metrics
}

// KafkaFallbackMessage represents a message sent to Kafka as a fallback
type KafkaFallbackMessage struct {
	Service   string      `json:"service"`
	Method    string      `json:"method"`
	Path      string      `json:"path"`
	Body      interface{} `json:"body"`
	Timestamp time.Time   `json:"timestamp"`
	Error     string      `json:"error"`
}
