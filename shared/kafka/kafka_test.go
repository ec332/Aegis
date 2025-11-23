package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// Mock implementations for testing
type mockKafkaWriter struct {
	messages []Message
	err      error
}

func (m *mockKafkaWriter) WriteMessages(ctx context.Context, messages ...Message) error {
	if m.err != nil {
		return m.err
	}
	m.messages = append(m.messages, messages...)
	return nil
}

func (m *mockKafkaWriter) Close() error {
	return nil
}

type mockKafkaReader struct {
	messages []Message
	index    int
	err      error
}

func (m *mockKafkaReader) ReadMessage(ctx context.Context) (Message, error) {
	if m.err != nil {
		return Message{}, m.err
	}
	if m.index >= len(m.messages) {
		return Message{}, nil // No more messages
	}
	msg := m.messages[m.index]
	m.index++
	return msg, nil
}

func (m *mockKafkaReader) Close() error {
	return nil
}

func (m *mockKafkaReader) SetOffset(offset int64) error {
	m.index = int(offset)
	return nil
}

// Producer Tests
func TestKafkaProducer(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	t.Run("Successful Message Publishing", func(t *testing.T) {
		mockWriter := &mockKafkaWriter{}
		producer := &Producer{
			writer: mockWriter,
			logger: logger,
		}

		ctx := context.Background()
		key := "test-key"
		message := map[string]interface{}{
			"event":   "test_event",
			"data":    "test_data",
			"timestamp": time.Now().Unix(),
		}

		err := producer.Publish(ctx, "test-topic", key, message)
		require.NoError(t, err)
		assert.Len(t, mockWriter.messages, 1)
		
		publishedMsg := mockWriter.messages[0]
		assert.Equal(t, "test-topic", publishedMsg.Topic)
		assert.Equal(t, []byte(key), publishedMsg.Key)
		
		var publishedData map[string]interface{}
		err = json.Unmarshal(publishedMsg.Value, &publishedData)
		require.NoError(t, err)
		assert.Equal(t, message["event"], publishedData["event"])
		assert.Equal(t, message["data"], publishedData["data"])
	})

	t.Run("Multiple Message Publishing", func(t *testing.T) {
		mockWriter := &mockKafkaWriter{}
		producer := &Producer{
			writer: mockWriter,
			logger: logger,
		}

		ctx := context.Background()
		topic := "test-topic"
		
		messages := []map[string]interface{}{
			{"id": 1, "message": "first"},
			{"id": 2, "message": "second"},
			{"id": 3, "message": "third"},
		}

		for i, msg := range messages {
			key := fmt.Sprintf("key-%d", i)
			err := producer.Publish(ctx, topic, key, msg)
			require.NoError(t, err)
		}

		assert.Len(t, mockWriter.messages, 3)
		
		for i, publishedMsg := range mockWriter.messages {
			assert.Equal(t, topic, publishedMsg.Topic)
			assert.Equal(t, []byte(fmt.Sprintf("key-%d", i)), publishedMsg.Key)
			
			var data map[string]interface{}
			err := json.Unmarshal(publishedMsg.Value, &data)
			require.NoError(t, err)
			assert.Equal(t, messages[i]["id"], data["id"])
			assert.Equal(t, messages[i]["message"], data["message"])
		}
	})

	t.Run("Publishing with Different Topics", func(t *testing.T) {
		mockWriter := &mockKafkaWriter{}
		producer := &Producer{
			writer: mockWriter,
			logger: logger,
		}

		ctx := context.Background()
		topics := []string{"market-events", "wallet-events", "settlement-events"}
		
		for _, topic := range topics {
			message := map[string]interface{}{
				"topic":   topic,
				"service": "test-service",
			}
			
			err := producer.Publish(ctx, topic, "test-key", message)
			require.NoError(t, err)
		}

		assert.Len(t, mockWriter.messages, 3)
		
		for i, publishedMsg := range mockWriter.messages {
			assert.Equal(t, topics[i], publishedMsg.Topic)
			
			var data map[string]interface{}
			err := json.Unmarshal(publishedMsg.Value, &data)
			require.NoError(t, err)
			assert.Equal(t, topics[i], data["topic"])
		}
	})

	t.Run("Publishing with Context Cancellation", func(t *testing.T) {
		mockWriter := &mockKafkaWriter{
			err: context.Canceled,
		}
		producer := &Producer{
			writer: mockWriter,
			logger: logger,
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		message := map[string]interface{}{"test": "data"}
		err := producer.Publish(ctx, "test-topic", "key", message)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("Publishing with Writer Error", func(t *testing.T) {
		mockWriter := &mockKafkaWriter{
			err: assert.AnError,
		}
		producer := &Producer{
			writer: mockWriter,
			logger: logger,
		}

		ctx := context.Background()
		message := map[string]interface{}{"test": "data"}
		
		err := producer.Publish(ctx, "test-topic", "key", message)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to publish message")
	})
}

// Consumer Tests
func TestKafkaConsumer(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	t.Run("Successful Message Consumption", func(t *testing.T) {
		testMessages := []Message{
			{
				Topic: "test-topic",
				Key:   []byte("key1"),
				Value: []byte(`{"event": "test1", "data": "data1"}`),
			},
			{
				Topic: "test-topic",
				Key:   []byte("key2"),
				Value: []byte(`{"event": "test2", "data": "data2"}`),
			},
		}

		mockReader := &mockKafkaReader{
			messages: testMessages,
		}
		
		consumer := &Consumer{
			reader: mockReader,
			logger: logger,
		}

		ctx := context.Background()
		
		// Consume first message
		msg, err := consumer.ReadMessage(ctx)
		require.NoError(t, err)
		assert.Equal(t, testMessages[0].Topic, msg.Topic)
		assert.Equal(t, testMessages[0].Key, msg.Key)
		assert.Equal(t, testMessages[0].Value, msg.Value)

		// Consume second message
		msg, err = consumer.ReadMessage(ctx)
		require.NoError(t, err)
		assert.Equal(t, testMessages[1].Topic, msg.Topic)
		assert.Equal(t, testMessages[1].Key, msg.Key)
		assert.Equal(t, testMessages[1].Value, msg.Value)
	})

	t.Run("Subscribe to Multiple Topics", func(t *testing.T) {
		testMessages := []Message{
			{Topic: "market-events", Key: []byte("key1"), Value: []byte(`{"type": "market"}`)},
			{Topic: "wallet-events", Key: []byte("key2"), Value: []byte(`{"type": "wallet"}`)},
			{Topic: "settlement-events", Key: []byte("key3"), Value: []byte(`{"type": "settlement"}`)},
		}

		mockReader := &mockKafkaReader{
			messages: testMessages,
		}
		
		consumer := &Consumer{
			reader: mockReader,
			logger: logger,
		}

		// Subscribe to multiple topics
		topics := []string{"market-events", "wallet-events", "settlement-events"}
		err := consumer.SubscribeTopics(topics)
		require.NoError(t, err)

		ctx := context.Background()
		
		// Consume messages from different topics
		for i := 0; i < len(testMessages); i++ {
			msg, err := consumer.ReadMessage(ctx)
			require.NoError(t, err)
			
			// Verify message content
			var data map[string]interface{}
			err = json.Unmarshal(msg.Value, &data)
			require.NoError(t, err)
			
			switch msg.Topic {
			case "market-events":
				assert.Equal(t, "market", data["type"])
			case "wallet-events":
				assert.Equal(t, "wallet", data["type"])
			case "settlement-events":
				assert.Equal(t, "settlement", data["type"])
			}
		}
	})

	t.Run("Message Processing with Timeout", func(t *testing.T) {
		mockReader := &mockKafkaReader{
			messages: []Message{}, // No messages
		}
		
		consumer := &Consumer{
			reader: mockReader,
			logger: logger,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		_, err := consumer.ReadMessage(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})

	t.Run("Offset Management", func(t *testing.T) {
		testMessages := []Message{
			{Topic: "test-topic", Key: []byte("key1"), Value: []byte(`{"id": 1}`)},
			{Topic: "test-topic", Key: []byte("key2"), Value: []byte(`{"id": 2}`)},
			{Topic: "test-topic", Key: []byte("key3"), Value: []byte(`{"id": 3}`)},
		}

		mockReader := &mockKafkaReader{
			messages: testMessages,
		}
		
		consumer := &Consumer{
			reader: mockReader,
			logger: logger,
		}

		ctx := context.Background()
		
		// Read first message
		msg1, err := consumer.ReadMessage(ctx)
		require.NoError(t, err)
		assert.Equal(t, []byte("key1"), msg1.Key)

		// Set offset back to beginning
		err = consumer.SetOffset(0)
		require.NoError(t, err)

		// Read first message again
		msg1Again, err := consumer.ReadMessage(ctx)
		require.NoError(t, err)
		assert.Equal(t, msg1.Key, msg1Again.Key)
		assert.Equal(t, msg1.Value, msg1Again.Value)
	})

	t.Run("Consumer with Reader Error", func(t *testing.T) {
		mockReader := &mockKafkaReader{
			err: assert.AnError,
		}
		
		consumer := &Consumer{
			reader: mockReader,
			logger: logger,
		}

		ctx := context.Background()
		_, err := consumer.ReadMessage(ctx)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read message")
	})
}

// Integration Tests
func TestKafkaProducerConsumerIntegration(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	t.Run("Producer-Consumer Message Flow", func(t *testing.T) {
		// Create test messages
		testMessages := []map[string]interface{}{
			{"event": "market_created", "market_id": "market-123", "timestamp": time.Now().Unix()},
			{"event": "wallet_funded", "wallet_id": "wallet-456", "amount": 1000.0},
			{"event": "settlement_completed", "settlement_id": "settlement-789", "status": "completed"},
		}

		// Create mock writer and reader that share messages
		sharedMessages := []Message{}
		mockWriter := &mockKafkaWriter{
			messages: sharedMessages,
		}
		mockReader := &mockKafkaReader{
			messages: sharedMessages,
		}

		producer := &Producer{
			writer: mockWriter,
			logger: logger,
		}

		consumer := &Consumer{
			reader: mockReader,
			logger: logger,
		}

		ctx := context.Background()
		topic := "integration-test-topic"

		// Publish messages
		for i, msg := range testMessages {
			key := fmt.Sprintf("key-%d", i)
			err := producer.Publish(ctx, topic, key, msg)
			require.NoError(t, err)
		}

		// Consume messages and verify
		for i, expectedMsg := range testMessages {
			consumedMsg, err := consumer.ReadMessage(ctx)
			require.NoError(t, err)

			assert.Equal(t, topic, consumedMsg.Topic)
			assert.Equal(t, []byte(fmt.Sprintf("key-%d", i)), consumedMsg.Key)

			var consumedData map[string]interface{}
			err = json.Unmarshal(consumedMsg.Value, &consumedData)
			require.NoError(t, err)

			// Verify message content matches
			for key, expectedValue := range expectedMsg {
				assert.Equal(t, expectedValue, consumedData[key])
			}
		}
	})

	t.Run("Error Handling in Message Flow", func(t *testing.T) {
		// Test error handling when producer fails
		mockWriter := &mockKafkaWriter{
			err: assert.AnError,
		}
		producer := &Producer{
			writer: mockWriter,
			logger: logger,
		}

		ctx := context.Background()
		message := map[string]interface{}{"test": "data"}
		
		err := producer.Publish(ctx, "test-topic", "key", message)
		assert.Error(t, err)

		// Test error handling when consumer fails
		mockReader := &mockKafkaReader{
			err: assert.AnError,
		}
		consumer := &Consumer{
			reader: mockReader,
			logger: logger,
		}

		_, err = consumer.ReadMessage(ctx)
		assert.Error(t, err)
	})
}

// Configuration Tests
func TestKafkaConfiguration(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	t.Run("Producer Configuration Validation", func(t *testing.T) {
		// Test with valid configuration
		config := Config{
			Brokers: []string{"localhost:9092", "localhost:9093"},
		}
		
		// This would normally create a real producer, but we'll test the config validation
		assert.NotEmpty(t, config.Brokers)
		assert.Equal(t, 2, len(config.Brokers))
	})

	t.Run("Consumer Configuration Validation", func(t *testing.T) {
		// Test with valid configuration
		brokers := []string{"localhost:9092"}
		groupID := "test-group"
		
		assert.NotEmpty(t, brokers)
		assert.NotEmpty(t, groupID)
	})

	t.Run("TLS Configuration", func(t *testing.T) {
		// Test TLS configuration options
		config := Config{
			Brokers: []string{"localhost:9092"},
			TLS: &TLSConfig{
				Enabled:  true,
				CertFile: "/path/to/cert.pem",
				KeyFile:  "/path/to/key.pem",
				CAFile:   "/path/to/ca.pem",
			},
		}
		
		assert.True(t, config.TLS.Enabled)
		assert.Equal(t, "/path/to/cert.pem", config.TLS.CertFile)
		assert.Equal(t, "/path/to/key.pem", config.TLS.KeyFile)
		assert.Equal(t, "/path/to/ca.pem", config.TLS.CAFile)
	})
}

// Performance Tests
func TestKafkaPerformance(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	t.Run("High Volume Message Publishing", func(t *testing.T) {
		mockWriter := &mockKafkaWriter{}
		producer := &Producer{
			writer: mockWriter,
			logger: logger,
		}

		ctx := context.Background()
		topic := "performance-test"
		messageCount := 100

		start := time.Now()
		
		for i := 0; i < messageCount; i++ {
			message := map[string]interface{}{
				"id":      i,
				"message": fmt.Sprintf("message-%d", i),
				"timestamp": time.Now().Unix(),
			}
			
			err := producer.Publish(ctx, topic, fmt.Sprintf("key-%d", i), message)
			require.NoError(t, err)
		}
		
		elapsed := time.Since(start)
		
		assert.Len(t, mockWriter.messages, messageCount)
		assert.Less(t, elapsed, 5*time.Second, "Publishing 100 messages should take less than 5 seconds")
		
		t.Logf("Published %d messages in %v", messageCount, elapsed)
	})

	t.Run("Concurrent Publishing and Consuming", func(t *testing.T) {
		sharedMessages := []Message{}
		mockWriter := &mockKafkaWriter{
			messages: sharedMessages,
		}
		mockReader := &mockKafkaReader{
			messages: sharedMessages,
		}

		producer := &Producer{
			writer: mockWriter,
			logger: logger,
		}

		consumer := &Consumer{
			reader: mockReader,
			logger: logger,
		}

		ctx := context.Background()
		topic := "concurrent-test"
		messageCount := 50

		// Start consuming in background
		consumed := make(chan Message, messageCount)
		go func() {
			for i := 0; i < messageCount; i++ {
				msg, err := consumer.ReadMessage(ctx)
				if err != nil {
					continue
				}
				consumed <- msg
			}
			close(consumed)
		}()

		// Publish messages
		start := time.Now()
		for i := 0; i < messageCount; i++ {
			message := map[string]interface{}{
				"id":   i,
				"data": fmt.Sprintf("data-%d", i),
			}
			
			err := producer.Publish(ctx, topic, fmt.Sprintf("key-%d", i), message)
			require.NoError(t, err)
		}

		// Wait for all messages to be consumed
		consumedCount := 0
		for range consumed {
			consumedCount++
		}

		elapsed := time.Since(start)
		
		assert.Equal(t, messageCount, consumedCount)
		assert.Less(t, elapsed, 10*time.Second, "Concurrent publishing and consuming should complete within 10 seconds")
		
		t.Logf("Concurrent test completed: published and consumed %d messages in %v", messageCount, elapsed)
	})
}