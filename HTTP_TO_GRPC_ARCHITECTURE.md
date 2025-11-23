# HTTP-to-gRPC Architecture for Aegis Prediction Market

## Overview

This document describes the HTTP-to-gRPC architecture implemented for the Aegis prediction market platform. The architecture enables frontend applications to communicate with the API Gateway using standard HTTP/REST requests, while the API Gateway communicates with backend services using gRPC with comprehensive resilience patterns.

## Architecture Components

### 1. Frontend → API Gateway (HTTP/REST)
- **Protocol**: HTTP/JSON
- **Port**: 8080
- **Format**: Standard REST API with JSON request/response bodies
- **Endpoints**:
  - `/api/markets` - Market management
  - `/api/wallets` - Wallet operations
  - `/api/settlements` - Settlement processing

### 2. API Gateway → Backend Services (gRPC)
- **Protocol**: gRPC with Protocol Buffers
- **Service Ports**:
  - Market Service: 50051
  - Wallet Service: 50052
  - Settlement Service: 50053
- **Resilience**: Circuit breaker, retry mechanisms, timeout handling

### 3. Fallback Messaging (Kafka)
- **Protocol**: Apache Kafka for asynchronous messaging
- **Port**: 9092
- **Topics**: Service-specific fallback topics (e.g., `market.ListMarkets.fallback`)

## Key Features

### Circuit Breaker Pattern
- **Timeout**: 1 second per gRPC call
- **Failure Threshold**: 5 consecutive failures
- **Recovery Time**: 30 seconds in Half-Open state
- **Behavior**: Fast-fail when circuit is open

### Retry Mechanism
- **Max Attempts**: 3
- **Initial Delay**: 100ms
- **Backoff Factor**: 2.0
- **Max Delay**: 1 second
- **Jitter**: ±10% to prevent thundering herd

### Kafka Fallback
- **Trigger**: Circuit breaker open or timeout
- **Topics**: `{service}.{method}.fallback`
- **Message Format**: JSON with service, method, timestamp, and error details
- **Processing**: Asynchronous by Kafka consumers

## Service Definitions

### Market Service (gRPC)
```protobuf
service MarketService {
  rpc GetMarket(GetMarketRequest) returns (GetMarketResponse);
  rpc CreateMarket(CreateMarketRequest) returns (CreateMarketResponse);
  rpc UpdateMarket(UpdateMarketRequest) returns (UpdateMarketResponse);
  rpc ListMarkets(ListMarketsRequest) returns (ListMarketsResponse);
  rpc GetMarketOptions(GetMarketOptionsRequest) returns (GetMarketOptionsResponse);
}
```

### Wallet Service (gRPC)
```protobuf
service WalletService {
  rpc CreateWalletAccount(CreateWalletAccountRequest) returns (CreateWalletAccountResponse);
  rpc GetWalletAccount(GetWalletAccountRequest) returns (GetWalletAccountResponse);
  rpc Deposit(DepositRequest) returns (DepositResponse);
  rpc Withdrawal(WithdrawalRequest) returns (WithdrawalResponse);
  rpc DebitWallet(DebitWalletRequest) returns (DebitWalletResponse);
  rpc CreditWallet(CreditWalletRequest) returns (CreditWalletResponse);
}
```

### Settlement Service (gRPC)
```protobuf
service SettlementService {
  rpc CreateSettlement(CreateSettlementRequest) returns (CreateSettlementResponse);
  rpc GetSettlement(GetSettlementRequest) returns (GetSettlementResponse);
  rpc CompleteSettlement(CompleteSettlementRequest) returns (CompleteSettlementResponse);
  rpc ProcessPayout(ProcessPayoutRequest) returns (ProcessPayoutResponse);
  rpc GetSettlementDistributions(GetSettlementDistributionsRequest) returns (GetSettlementDistributionsResponse);
}
```

## Request Flow

### Successful Request Flow
1. **Frontend** sends HTTP request to API Gateway
2. **API Gateway** validates and parses the request
3. **API Gateway** creates gRPC client with circuit breaker and retry
4. **gRPC Client** makes call to backend service
5. **Backend Service** processes request and returns response
6. **API Gateway** converts gRPC response to HTTP/JSON
7. **Frontend** receives HTTP response

### Circuit Breaker Open Flow
1. **Frontend** sends HTTP request to API Gateway
2. **API Gateway** attempts gRPC call
3. **Circuit Breaker** detects open state (too many failures)
4. **API Gateway** immediately fails fast (no timeout)
5. **API Gateway** sends message to Kafka fallback topic
6. **API Gateway** returns HTTP 202 (Accepted) to frontend
7. **Kafka Consumer** processes message asynchronously

### Timeout Fallback Flow
1. **Frontend** sends HTTP request to API Gateway
2. **API Gateway** attempts gRPC call with 1-second timeout
3. **gRPC Call** times out after 1 second
4. **API Gateway** sends message to Kafka fallback topic
5. **API Gateway** returns HTTP 202 (Accepted) to frontend
6. **Kafka Consumer** processes message asynchronously

## Configuration

### Environment Variables
```bash
# API Gateway
PORT=8080
MARKET_SERVICE_GRPC_ADDR=market-service:50051
WALLET_SERVICE_GRPC_ADDR=wallet-service:50052
SETTLEMENT_SERVICE_GRPC_ADDR=settlement-service:50053
KAFKA_BROKERS=kafka:9092

# Backend Services
GRPC_PORT=50051  # Varies by service
DB_HOST=postgres
DB_PORT=5432
```

### Circuit Breaker Configuration
```go
const (
    CircuitBreakerTimeout       = 1 * time.Second
    CircuitBreakerMaxConcurrent = 100
    CircuitBreakerMinRequests  = 5
    CircuitBreakerFailureRatio  = 0.6
    CircuitBreakerCooldown     = 30 * time.Second
)
```

### Retry Configuration
```go
const (
    RetryMaxAttempts   = 3
    RetryInitialDelay  = 100 * time.Millisecond
    RetryMaxDelay      = 1 * time.Second
    RetryBackoffFactor = 2.0
    RetryJitter        = 0.1
)
```

## Deployment

### Docker Compose
```bash
# Start all services with gRPC architecture
docker-compose -f docker-compose-grpc.yml up -d

# Services will be available at:
# - API Gateway: http://localhost:8080
# - gRPC Services: localhost:50051, 50052, 50053
# - Kafka: localhost:9092
```

### Health Checks
- **API Gateway**: `GET /health`
- **gRPC Services**: Built-in gRPC health check protocol
- **Kafka**: Built-in Kafka health check

## Testing

### Integration Tests
```bash
# Run HTTP-to-gRPC integration tests
go test ./tests/integration -v -run TestHTTPToGRPCIntegration

# Test circuit breaker and Kafka fallback
go test ./tests/integration -v -run TestCircuitBreakerAndKafkaFallback

# Test complete end-to-end flow
go test ./tests/integration -v -run TestEndToEndFlow
```

### Manual Testing
```bash
# Create a market via HTTP
curl -X POST http://localhost:8080/api/markets \
  -H "Content-Type: application/json" \
  -d '{"title":"Test Market","description":"Test","options":[{"title":"A","odds":2.0}],"end_time":1234567890}'

# Get markets via HTTP
curl http://localhost:8080/api/markets

# Create wallet via HTTP
curl -X POST http://localhost:8080/api/wallets \
  -H "Content-Type: application/json" \
  -d '{"user_id":"user123","currency":"USD","initial_balance":1000}'
```

## Monitoring and Observability

### Metrics Collection
- **gRPC Call Metrics**: Success/failure rates, latency, circuit breaker state
- **HTTP Metrics**: Request rates, response times, error rates
- **Kafka Metrics**: Message production/consumption rates

### Logging
- **Structured Logging**: JSON format with correlation IDs
- **Log Levels**: ERROR, WARN, INFO, DEBUG
- **Key Fields**: service, method, duration, error, circuit_breaker_state

### Distributed Tracing
- **Request Tracing**: Correlation IDs across HTTP and gRPC calls
- **Performance Monitoring**: Request latency breakdown by service

## Benefits

1. **Frontend Simplicity**: Standard HTTP/REST interface
2. **Service Resilience**: Circuit breaker and retry patterns
3. **High Availability**: Kafka fallback for critical operations
4. **Performance**: gRPC for efficient inter-service communication
5. **Observability**: Comprehensive metrics and logging
6. **Scalability**: Asynchronous processing via Kafka

## Migration Path

The architecture supports both HTTP and gRPC backends:
- **Phase 1**: Deploy gRPC services alongside existing HTTP services
- **Phase 2**: Update API Gateway to use gRPC clients
- **Phase 3**: Gradually migrate all inter-service communication to gRPC
- **Phase 4**: Retire HTTP-only service endpoints

## Security Considerations

1. **Network Isolation**: gRPC services communicate over internal network
2. **Authentication**: JWT tokens passed from HTTP to gRPC context
3. **Authorization**: Service-level authorization checks
4. **Encryption**: TLS for gRPC communication (recommended for production)
5. **Rate Limiting**: API Gateway implements rate limiting per client

## Performance Characteristics

- **Latency**: ~1-5ms for gRPC calls within same datacenter
- **Throughput**: 10,000+ requests/second per service instance
- **Circuit Breaker**: <100ms fast-fail when open
- **Retry Overhead**: <10ms average for successful calls
- **Kafka Latency**: ~10-100ms for message production

This architecture provides a robust, scalable foundation for the Aegis prediction market platform with comprehensive resilience patterns and observability.