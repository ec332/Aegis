package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

type Config struct {
	Brokers      []string
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func DefaultConfig() Config {
	return Config{
		Brokers:      []string{"localhost:9092"},
		DialTimeout:  10 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
}

type Producer struct {
	writer *kafka.Writer
	logger *zap.Logger
}

func NewProducer(config Config, logger *zap.Logger) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		BatchTimeout: 100 * time.Millisecond,
		WriteTimeout: config.WriteTimeout,
		Async:        false,
		Logger:       kafka.LoggerFunc(logger.Sugar().Debugf),
		ErrorLogger:  kafka.LoggerFunc(logger.Sugar().Errorf),
	}

	return &Producer{
		writer: writer,
		logger: logger,
	}
}

func (p *Producer) Publish(ctx context.Context, topic string, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	message := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
		Time:  time.Now(),
	}

	p.logger.Debug("publishing message to kafka",
		zap.String("topic", topic),
		zap.String("key", key),
		zap.Int("size", len(data)))

	err = p.writer.WriteMessages(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	p.logger.Info("message published to kafka",
		zap.String("topic", topic),
		zap.String("key", key))

	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

type Consumer struct {
	reader *kafka.Reader
	logger *zap.Logger
}

func NewConsumer(config Config, topic string, groupID string, logger *zap.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        config.Brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6, // 10MB
		CommitInterval: 1 * time.Second,
		StartOffset:    kafka.FirstOffset,
		Logger:         kafka.LoggerFunc(logger.Sugar().Debugf),
		ErrorLogger:    kafka.LoggerFunc(logger.Sugar().Errorf),
	})

	return &Consumer{
		reader: reader,
		logger: logger,
	}
}

func (c *Consumer) Consume(ctx context.Context, handler func(Message) error) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					return err
				}
				c.logger.Error("failed to read message from kafka",
					zap.Error(err))
				continue
			}

			c.logger.Debug("consumed message from kafka",
				zap.String("topic", msg.Topic),
				zap.String("key", string(msg.Key)),
				zap.Int("size", len(msg.Value)))

			message := Message{
				Topic:     msg.Topic,
				Key:       string(msg.Key),
				Value:     msg.Value,
				Offset:    msg.Offset,
				Partition: msg.Partition,
				Time:      msg.Time,
			}

			if err := handler(message); err != nil {
				c.logger.Error("failed to handle message",
					zap.Error(err),
					zap.String("topic", msg.Topic),
					zap.String("key", string(msg.Key)))
				continue
			}
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

type Message struct {
	Topic     string
	Key       string
	Value     []byte
	Offset    int64
	Partition int
	Time      time.Time
}

func (m *Message) Unmarshal(v interface{}) error {
	return json.Unmarshal(m.Value, v)
}

// Topic names for different services and message types
const (
	// Market Service Topics
	TopicMarketCreated     = "market.created"
	TopicMarketUpdated     = "market.updated"
	TopicMarketResolved    = "market.resolved"
	TopicUserCreated       = "user.created"
	TopicUserUpdated       = "user.updated"
	
	// Wallet Service Topics
	TopicWalletCreated     = "wallet.created"
	TopicWalletUpdated     = "wallet.updated"
	TopicTransactionCreated = "transaction.created"
	TopicDepositCompleted  = "deposit.completed"
	TopicWithdrawalCompleted = "withdrawal.completed"
	
	// Settlement Service Topics
	TopicSettlementCreated = "settlement.created"
	TopicSettlementCompleted = "settlement.completed"
	TopicPayoutProcessed   = "payout.processed"
	
	// Cross-service Topics
	TopicServiceHealth     = "service.health"
	TopicCircuitBreaker    = "circuit.breaker"
)