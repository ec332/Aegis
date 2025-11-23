package kafka

import (
	"context"
	"sync"
	"go.uber.org/zap"
)

// MockProducer is a mock implementation of Kafka producer for testing
type MockProducer struct {
	logger           *zap.Logger
	publishedMessages []MockMessage
	mu               sync.RWMutex
}

type MockMessage struct {
	Topic string
	Key   string
	Value interface{}
}

func NewMockProducer(logger *zap.Logger) *MockProducer {
	return &MockProducer{
		logger:            logger,
		publishedMessages: make([]MockMessage, 0),
	}
}

func (m *MockProducer) Publish(ctx context.Context, topic, key string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.publishedMessages = append(m.publishedMessages, MockMessage{
		Topic: topic,
		Key:   key,
		Value: value,
	})
	
	return nil
}

func (m *MockProducer) Close() error {
	return nil
}

func (m *MockProducer) WasMessagePublished() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.publishedMessages) > 0
}

func (m *MockProducer) GetPublishedMessages() []MockMessage {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]MockMessage{}, m.publishedMessages...)
}

func (m *MockProducer) ClearMessages() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.publishedMessages = make([]MockMessage, 0)
}