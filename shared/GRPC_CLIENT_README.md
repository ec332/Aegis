# Resilient gRPC Client with Circuit Breaker and Kafka Fallback

This implementation provides a robust gRPC client with automatic circuit breaker, retry mechanisms, and Kafka fallback for the Aegis prediction market platform.

## Features

- **Circuit Breaker Pattern**: Automatically opens when failure threshold is reached, preventing cascading failures
- **1-Second Timeout**: All gRPC calls timeout after 1 second as specified
- **Kafka Fallback**: When gRPC calls timeout or fail, requests are automatically queued to Kafka for asynchronous processing
- **Retry Mechanism**: Exponential backoff retry for transient failures
- **Comprehensive Metrics**: Request counts, failure rates, response times, and circuit breaker state
- **Service-Specific Topic Routing**: Automatic routing to appropriate Kafka topics based on service and method

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Application   │───▶│ Resilient Client │───▶│   gRPC Server   │
│                 │    │                  │    │                 │
└─────────────────┘    │  • Circuit Breaker│    └─────────────────┘
                       │  • Retry Logic    │              │
                       │  • Timeout (1s)   │              ▼
                       │  • Kafka Fallback │    ┌─────────────────┐
                       └──────────────────┘    │     Kafka       │
                                │              │    Topics         │
                                ▼              └─────────────────┘
                        ┌──────────────────┐
                        │  Service-Specific │
                        │  Topic Mapping    │
                        └──────────────────┘
```

## Usage

### Basic Usage

```go
import (
    "context"
    "github.com/aegis/shared/grpc"
    "github.com/aegis/shared/kafka"
    "github.com/aegis/shared/metrics"
    "go.uber.org/zap"
)

// Initialize logger and metrics
logger, _ := zap.NewProduction()
metricsRegistry := metrics.NewRegistry(logger)

// Configure client
config := grpc.DefaultClientConfig("market", "localhost:50051")
config.Timeout = 1 * time.Second
config.KafkaFallback = true

// Create resilient client
client, err := grpc.NewResilientClient(config, logger, metricsRegistry)
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Make gRPC call with automatic resilience
request := &market.GetMarketRequest{MarketId: "market-123"}
response := &market.GetMarketResponse{}

err = client.Invoke(ctx, "/market.MarketService/GetMarket", request, response)

if err == grpc.ErrKafkaFallback {
    // Request was queued to Kafka for async processing
    log.Println("Request queued to Kafka fallback")
} else if err != nil {
    log.Printf("Request failed: %v", err)
} else {
    log.Printf("Response: %+v", response)
}
```

### Configuration Options

```go
config := grpc.ClientConfig{
    ServiceName: "market",                    // Service identifier
    Target:      "localhost:50051",           // gRPC server address
    Timeout:     1 * time.Second,             // 1-second timeout
    MaxRetries:  3,                            // Maximum retry attempts
    
    CircuitBreaker: circuitbreaker.Config{
        FailureThreshold:   5,                // Failures before opening
        SuccessThreshold:   2,                // Successes before closing
        Timeout:            60 * time.Second, // Time before attempting reset
        MaxConcurrentCalls: 100,              // Max concurrent calls
    },
    
    KafkaFallback: true,                        // Enable Kafka fallback
    KafkaConfig:   kafka.DefaultConfig(),     // Kafka configuration
}
```

## Circuit Breaker States

### Closed State (Normal Operation)
- All requests pass through to gRPC service
- Failures are counted
- After `FailureThreshold` failures, circuit opens

### Open State (Service Degraded)
- All requests fail immediately with `ErrCircuitOpen`
- No calls made to gRPC service
- After `Timeout` duration, circuit transitions to half-open

### Half-Open State (Testing Recovery)
- Limited number of requests allowed through
- If successful, circuit closes
- If failed, circuit opens again

## Kafka Fallback

When gRPC calls timeout or fail, requests are automatically published to Kafka topics:

- **Market Service**: `market.updated`
- **Wallet Service**: `transaction.created`
- **Settlement Service**: `settlement.created`
- **Health/Other**: `service.health`

### Message Format

```json
{
  "service": "market",
  "method": "GetMarket",
  "payload": { /* original request */ },
  "timestamp": "2023-11-23T10:30:00Z"
}
```

## Retry Mechanism

- **Exponential Backoff**: Delay increases exponentially between retries
- **Jitter**: Random variation prevents thundering herd
- **Configurable**: Max attempts and initial delay can be customized
- **Smart Retries**: Only retries on specific error types (timeouts, circuit open, unavailable)

## Metrics and Observability

### Request Metrics
- Total requests count
- Failed requests count
- Average response time
- Request rate (requests/second)

### Circuit Breaker Metrics
- Circuit breaker state (open/closed/half-open)
- Number of state transitions
- Failure count and rate
- Success count and rate

### Kafka Fallback Metrics
- Number of messages published to Kafka
- Kafka publish success/failure rates
- Message queue depth (if applicable)

## Error Handling

### gRPC Errors
- **DeadlineExceeded**: Retried with exponential backoff, falls back to Kafka
- **Unavailable**: Retried, falls back to Kafka after max retries
- **ResourceExhausted**: Retried with backoff
- **Other errors**: Not retried, no Kafka fallback

### Circuit Breaker Errors
- **ErrCircuitOpen**: Request rejected immediately, no retry
- **ErrCircuitHalfOpen**: Limited requests allowed for testing

### Kafka Errors
- **Publish failures**: Logged and returned as combined error
- **Connection issues**: Retried with exponential backoff

## Testing

Run the comprehensive test suite:

```bash
cd /Users/bytedance/Desktop/school/Aegis/shared
go test ./grpc -v
```

### Test Coverage

- **Circuit breaker state transitions**: Closed → Open → Half-Open → Closed
- **Timeout scenarios**: gRPC calls timeout and fall back to Kafka
- **Retry mechanisms**: Exponential backoff and jitter behavior
- **Kafka fallback**: Message publishing and topic routing
- **Error handling**: Various gRPC error codes and responses
- **Concurrent operations**: Thread safety and race condition testing

## Integration with Existing Services

### API Gateway Integration

Replace HTTP proxy calls with resilient gRPC client calls:

```go
// Before (HTTP proxy)
func (g *Gateway) handleGetMarket(w http.ResponseWriter, r *http.Request) {
    // Proxy to market service
    g.marketProxy.ServeHTTP(w, r)
}

// After (Resilient gRPC)
func (g *Gateway) handleGetMarket(w http.ResponseWriter, r *http.Request) {
    marketID := r.URL.Query().Get("market_id")
    
    request := &market.GetMarketRequest{MarketId: marketID}
    response := &market.GetMarketResponse{}
    
    err := g.marketClient.GetMarket(r.Context(), request, response)
    
    if err != nil {
        if err == grpc.ErrKafkaFallback {
            // Return 202 Accepted - request queued for async processing
            w.WriteHeader(http.StatusAccepted)
            json.NewEncoder(w).Encode(map[string]string{
                "status": "queued",
                "message": "Request queued for asynchronous processing",
            })
        } else {
            // Return appropriate error
            http.Error(w, err.Error(), http.StatusServiceUnavailable)
        }
        return
    }
    
    // Return successful response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

### Service Implementation

Services can consume from Kafka topics for async processing:

```go
func (s *MarketService) processKafkaMessages() {
    consumer := kafka.NewConsumer(kafka.Config{
        Brokers: []string{"localhost:9092"},
        Topic:   kafka.TopicMarketUpdated,
        GroupID: "market-service",
    }, s.logger)
    
    for message := range consumer.Messages() {
        var grpcMessage grpc.KafkaMessage
        if err := json.Unmarshal(message.Value, &grpcMessage); err != nil {
            s.logger.Error("Failed to unmarshal Kafka message", zap.Error(err))
            continue
        }
        
        // Process the request asynchronously
        switch grpcMessage.Method {
        case "GetMarket":
            s.processGetMarket(grpcMessage.Payload)
        case "CreateMarket":
            s.processCreateMarket(grpcMessage.Payload)
        // ... other methods
        }
    }
}
```

## Performance Considerations

- **Connection Pooling**: gRPC connections are reused across requests
- **Circuit Breaker**: Prevents resource exhaustion during service degradation
- **Kafka Buffering**: Asynchronous processing prevents blocking on service failures
- **Metrics Overhead**: Minimal performance impact with efficient counters and histograms
- **Timeout Configuration**: 1-second timeout balances responsiveness and reliability

## Monitoring and Alerting

### Key Metrics to Monitor

1. **Circuit Breaker State**: Alert when circuit opens
2. **Kafka Fallback Rate**: High fallback rate indicates service issues
3. **Response Time**: Monitor p50, p95, p99 response times
4. **Error Rate**: Track request failure rates by service
5. **Retry Rate**: High retry rates may indicate network issues

### Alerting Thresholds

- **Circuit Open**: Immediate alert
- **Fallback Rate > 10%**: Warning alert
- **Error Rate > 5%**: Warning alert
- **Response Time > 2s**: Performance alert
- **Retry Rate > 20%**: Investigation alert

## Future Enhancements

- **Distributed Tracing**: Integration with OpenTelemetry/Jaeger
- **Load Balancing**: Client-side load balancing for multiple service instances
- **Health Checks**: Automatic health checking and service discovery
- **Rate Limiting**: Request rate limiting per client/service
- **Bulk Operations**: Batch processing for high-volume scenarios