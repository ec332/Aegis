package grpc

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aegis/shared/circuitbreaker"
	"github.com/aegis/shared/kafka"
	"github.com/aegis/shared/retry"
	"go.uber.org/zap"
)

// Test the core resilient client functionality without proto dependencies
func TestCircuitBreakerIntegration(t *testing.T) {
	logger := zap.NewNop()
	
	config := DefaultClientConfig("test-service", "localhost:50051")
	config.Timeout = 50 * time.Millisecond
	config.CircuitBreaker.FailureThreshold = 2
	
	// Create circuit breaker
	cb := circuitbreaker.NewCircuitBreaker("test-service", config.CircuitBreaker, logger)
	
	// Test successful execution
	ctx := context.Background()
	err := cb.Execute(ctx, func() error {
		return nil // Success
	})
	
	if err != nil {
		t.Errorf("Expected successful execution, got error: %v", err)
	}
	
	// Test failure execution
	err = cb.Execute(ctx, func() error {
		return errors.New("test error")
	})
	
	if err == nil {
		t.Error("Expected execution to fail")
	}
	
	// Test circuit opening after multiple failures
	for i := 0; i < config.CircuitBreaker.FailureThreshold; i++ {
		cb.Execute(ctx, func() error {
			return errors.New("test error")
		})
	}
	
	// Circuit should now be open
	err = cb.Execute(ctx, func() error {
		return nil // This should not be called
	})
	
	if !errors.Is(err, circuitbreaker.ErrCircuitOpen) {
		t.Errorf("Expected circuit open error, got: %v", err)
	}
}

func TestKafkaFallbackMessage(t *testing.T) {
	logger := zap.NewNop()
	
	// Create mock Kafka producer
	kafkaProducer := kafka.NewMockProducer(logger)
	
	// Test message publishing
	ctx := context.Background()
	message := KafkaMessage{
		Service:   "test-service",
		Method:    "TestMethod",
		Payload:   map[string]string{"key": "value"},
		Timestamp: time.Now(),
	}
	
	err := kafkaProducer.Publish(ctx, kafka.TopicMarketUpdated, "test-key", message)
	
	if err != nil {
		t.Errorf("Expected successful publish, got error: %v", err)
	}
	
	if !kafkaProducer.WasMessagePublished() {
		t.Error("Expected message to be published")
	}
	
	messages := kafkaProducer.GetPublishedMessages()
	if len(messages) != 1 {
		t.Errorf("Expected 1 published message, got: %d", len(messages))
	}
	
	if messages[0].Topic != kafka.TopicMarketUpdated {
		t.Errorf("Expected topic %s, got: %s", kafka.TopicMarketUpdated, messages[0].Topic)
	}
}

func TestRetryMechanism(t *testing.T) {
	logger := zap.NewNop()
	
	config := retry.DefaultConfig()
	config.MaxAttempts = 3
	config.InitialDelay = 10 * time.Millisecond
	
	attemptCount := 0
	
	retryable := func(ctx context.Context) error {
		attemptCount++
		if attemptCount < 3 {
			return errors.New("temporary error")
		}
		return nil // Success on 3rd attempt
	}
	
	ctx := context.Background()
	err := retry.Execute(ctx, config, logger, retryable)
	
	if err != nil {
		t.Errorf("Expected successful execution after retries, got error: %v", err)
	}
	
	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts, got: %d", attemptCount)
	}
}

func TestTopicMapping(t *testing.T) {
	tests := []struct {
		serviceName string
		method      string
		expectedTopic string
	}{
		{"market", "GetMarket", kafka.TopicMarketUpdated},
		{"wallet", "Deposit", kafka.TopicTransactionCreated},
		{"settlement", "CreateSettlement", kafka.TopicSettlementCreated},
		{"unknown", "SomeMethod", kafka.TopicServiceHealth},
	}
	
	for _, test := range tests {
		config := DefaultClientConfig(test.serviceName, "localhost:50051")
		client := &ResilientClient{
			config: config,
		}
		
		topic := client.getTopicForMethod(test.method)
		if topic != test.expectedTopic {
			t.Errorf("For service %s, expected topic %s, got %s", test.serviceName, test.expectedTopic, topic)
		}
	}
}