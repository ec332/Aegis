package grpc

import (
    "context"
    "errors"
    "fmt"
    "time"

	"github.com/aegis/shared/circuitbreaker"
	"github.com/aegis/shared/kafka"
	"github.com/aegis/shared/retry"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	DefaultTimeout = 1 * time.Second
	DefaultRetries = 3
)

type ClientConfig struct {
	ServiceName      string
	Target           string
	Timeout          time.Duration
	MaxRetries       int
	CircuitBreaker   circuitbreaker.Config
	KafkaFallback    bool
	KafkaConfig      kafka.Config
}

func DefaultClientConfig(serviceName, target string) ClientConfig {
	return ClientConfig{
		ServiceName: serviceName,
		Target:      target,
		Timeout:     DefaultTimeout,
		MaxRetries:  DefaultRetries,
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

type ResilientClient struct {
    config         ClientConfig
    conn           *grpc.ClientConn
    circuitBreaker *circuitbreaker.CircuitBreaker
    kafkaProducer  *kafka.Producer
    logger         *zap.Logger
}

func NewResilientClient(config ClientConfig, logger *zap.Logger) (*ResilientClient, error) {
	// Create gRPC connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	conn, err := grpc.DialContext(ctx, config.Target, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC service %s: %w", config.ServiceName, err)
	}

	// Create circuit breaker
	cb := circuitbreaker.NewCircuitBreaker(config.ServiceName, config.CircuitBreaker, logger)

	// Create Kafka producer if fallback is enabled
	var kafkaProducer *kafka.Producer
	if config.KafkaFallback {
		kafkaProducer = kafka.NewProducer(config.KafkaConfig, logger)
	}

    return &ResilientClient{
        config:         config,
        conn:           conn,
        circuitBreaker: cb,
        kafkaProducer:  kafkaProducer,
        logger:         logger,
    }, nil
}

func (c *ResilientClient) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	// Create a retryable function
	retryable := func(ctx context.Context) error {
		return c.invokeWithCircuitBreaker(ctx, method, args, reply, opts...)
	}

	// Configure retry
	retryConfig := retry.DefaultConfig()
	retryConfig.MaxAttempts = c.config.MaxRetries
	retryConfig.RetryableErrors = []error{
		circuitbreaker.ErrCircuitOpen,
		context.DeadlineExceeded,
	}

	// Execute with retry
	return retry.Execute(ctx, retryConfig, c.logger, retryable)
}

func (c *ResilientClient) invokeWithCircuitBreaker(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	return c.circuitBreaker.Execute(ctx, func() error {
		return c.invokeWithTimeout(ctx, method, args, reply, opts...)
	})
}

func (c *ResilientClient) invokeWithTimeout(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	// Add timeout to context
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

    err := c.conn.Invoke(ctx, method, args, reply, opts...)

	if err != nil {
		// Check if this is a timeout or circuit breaker should trigger
		if c.isRetryableError(err) {
			c.logger.Warn("gRPC call failed, will attempt Kafka fallback",
				zap.String("service", c.config.ServiceName),
				zap.String("method", method),
				zap.Error(err))
			
			// Attempt Kafka fallback if enabled
			if c.config.KafkaFallback && c.kafkaProducer != nil {
				return c.fallbackToKafka(method, args)
			}
		}
	}

	return err
}

func (c *ResilientClient) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for timeout errors
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Check for gRPC status codes that indicate temporary failures
	st, ok := status.FromError(err)
	if !ok {
		return true // Default to retryable for unknown errors
	}

	switch st.Code() {
	case codes.DeadlineExceeded, codes.Unavailable, codes.ResourceExhausted:
		return true
	default:
		return false
	}
}

func (c *ResilientClient) fallbackToKafka(method string, args interface{}) error {
	topic := c.getTopicForMethod(method)
	key := fmt.Sprintf("%s_%s", c.config.ServiceName, method)
	
	message := KafkaMessage{
		Service:   c.config.ServiceName,
		Method:    method,
		Payload:   args,
		Timestamp: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := c.kafkaProducer.Publish(ctx, topic, key, message)
	if err != nil {
		c.logger.Error("failed to publish to Kafka fallback",
			zap.String("service", c.config.ServiceName),
			zap.String("method", method),
			zap.String("topic", topic),
			zap.Error(err))
		return fmt.Errorf("gRPC call failed and Kafka fallback also failed: %w", err)
	}

	c.logger.Info("successfully published to Kafka fallback",
		zap.String("service", c.config.ServiceName),
		zap.String("method", method),
		zap.String("topic", topic))

	
	// Return a specific error indicating fallback was used
	return ErrKafkaFallback
}

func (c *ResilientClient) getTopicForMethod(method string) string {
	// Map gRPC methods to Kafka topics
	switch c.config.ServiceName {
	case "market":
		return kafka.TopicMarketUpdated
	case "wallet":
		return kafka.TopicTransactionCreated
	case "settlement":
		return kafka.TopicSettlementCreated
	default:
		return kafka.TopicServiceHealth
	}
}

func (c *ResilientClient) Close() error {
	var errs []error
	
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close gRPC connection: %w", err))
		}
	}
	
	if c.kafkaProducer != nil {
		if err := c.kafkaProducer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Kafka producer: %w", err))
		}
	}
	
	if len(errs) > 0 {
		return fmt.Errorf("errors closing resilient client: %v", errs)
	}
	
	return nil
}

func (c *ResilientClient) GetConnection() *grpc.ClientConn {
	return c.conn
}

func (c *ResilientClient) GetCircuitBreaker() *circuitbreaker.CircuitBreaker {
	return c.circuitBreaker
}

// Metrics removed

type KafkaMessage struct {
	Service   string      `json:"service"`
	Method    string      `json:"method"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}

var ErrKafkaFallback = fmt.Errorf("request was handled via Kafka fallback due to gRPC failure")
