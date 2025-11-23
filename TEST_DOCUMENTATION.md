# Aegis Test Suite Documentation

## Overview

The Aegis test suite provides comprehensive testing coverage for the microservices architecture, including unit tests, integration tests, end-to-end tests, and performance tests. The test suite ensures reliability, performance, and correctness of all services and their interactions.

## Test Structure

```
tests/
├── integration/          # Integration tests
│   ├── http_grpc_test.go          # HTTP to gRPC integration tests
│   └── services_integration_test.go # Service-to-service integration tests
├── e2e/                  # End-to-end tests
│   └── complete_flows_test.go      # Complete business flow tests
└── unit/                 # Unit tests (distributed across services)

api-gateway/cmd/main_test.go     # API Gateway unit tests
shared/kafka/kafka_test.go       # Kafka unit tests
*/internal/*_test.go             # Service-specific unit tests
```

## Test Categories

### 1. Unit Tests

Unit tests focus on individual components and functions, ensuring they work correctly in isolation.

#### API Gateway Unit Tests (`api-gateway/cmd/main_test.go`)
- **CORS Configuration**: Tests various CORS scenarios including allowed origins, methods, headers, and preflight requests
- **Health Endpoints**: Verifies health check functionality and response formats
- **Market Endpoints**: Tests market creation, retrieval, listing, and updates
- **Wallet Endpoints**: Tests wallet creation, deposits, and withdrawals
- **Settlement Endpoints**: Tests settlement creation and completion
- **Error Handling**: Tests gRPC error conversion and Kafka fallback scenarios
- **Utility Functions**: Tests helper functions like ID extraction and JSON response writing

#### Kafka Unit Tests (`shared/kafka/kafka_test.go`)
- **Producer Tests**: Message publishing, multiple topics, error handling
- **Consumer Tests**: Message consumption, topic subscription, offset management
- **Integration Tests**: Producer-consumer message flow, error scenarios
- **Configuration Tests**: Broker configuration, TLS settings
- **Performance Tests**: High-volume publishing, concurrent operations

### 2. Integration Tests

Integration tests verify that different services work together correctly.

#### Service-to-Service Integration (`tests/integration/services_integration_test.go`)
- **Market Service Integration**: Complete market lifecycle testing
- **Wallet Service Integration**: Wallet operations and balance management
- **Settlement Service Integration**: Settlement creation and completion
- **Cross-Service Integration**: End-to-end flows across all services
- **Error Handling Integration**: Invalid requests and service unavailability
- **Concurrent Requests**: Multiple simultaneous operations
- **Kafka Integration**: Fallback message publishing and consumption
- **Health Check Integration**: Service dependency verification

#### HTTP to gRPC Integration (`tests/integration/http_grpc_test.go`)
- **HTTP to gRPC Flow**: Tests API Gateway routing to backend services
- **Circuit Breaker**: Tests circuit breaker opening and Kafka fallback
- **Resilience Patterns**: Retry mechanisms and metrics collection
- **End-to-End Flow**: Complete prediction market workflow

### 3. End-to-End Tests

E2E tests simulate real user scenarios and complete business flows.

#### Complete Business Flows (`tests/e2e/complete_flows_test.go`)
- **Complete Prediction Market Flow**: Full lifecycle from market creation to settlement completion
- **Concurrent Operations**: Multiple markets and simultaneous wallet operations
- **Error Recovery**: System resilience and recovery from various error conditions
- **System Health**: Comprehensive health check and monitoring verification

## Running Tests

### Prerequisites

1. **Services Running**: All services must be running (API Gateway, Market Service, Wallet Service, Settlement Service, Transaction Service)
2. **Infrastructure**: Kafka and Redis must be available
3. **Environment Variables**: Required environment variables must be set

### Quick Start

```bash
# Run all tests
make test-all

# Run specific test categories
make test-unit
make test-integration
make test-e2e

# Run tests with coverage
make test-coverage

# Run tests in parallel
make test-parallel
```

### Detailed Test Commands

#### Unit Tests
```bash
# Run API Gateway unit tests
cd api-gateway && go test ./cmd -v

# Run Kafka unit tests
cd shared && go test ./kafka -v

# Run all service unit tests
find . -name "*_test.go" -path "*/internal/*" -exec dirname {} \; | sort -u | xargs -I {} go test {} -v
```

#### Integration Tests
```bash
# Run all integration tests
cd tests/integration && go test -v

# Run specific integration test
cd tests/integration && go test -run TestHTTPToGRPCIntegration -v

# Run with race detection
cd tests/integration && go test -race -v
```

#### End-to-End Tests
```bash
# Run all E2E tests
cd tests/e2e && go test -v

# Run specific E2E test
cd tests/e2e && go test -run TestCompletePredictionMarketFlow -v

# Run with timeout
cd tests/e2e && go test -timeout 5m -v
```

### Environment-Specific Testing

#### Local Development
```bash
# Set local environment variables
export CORS_ORIGINS="http://localhost:3000,http://127.0.0.1:3000"
export KAFKA_BROKERS="localhost:9092"

# Run tests with local configuration
make test-local
```

#### CI/CD Pipeline
```bash
# Run tests in CI mode (with coverage and JUnit output)
make test-ci

# Run smoke tests only
make test-smoke
```

## Test Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `CORS_ORIGINS` | Allowed CORS origins | `http://localhost:3000,http://127.0.0.1:3000` |
| `CORS_METHODS` | Allowed HTTP methods | `GET,POST,PUT,DELETE,OPTIONS` |
| `CORS_HEADERS` | Allowed request headers | `Accept,Content-Type,Authorization` |
| `KAFKA_BROKERS` | Kafka broker addresses | `localhost:9092` |

### Test Data

Tests use the following test data patterns:
- **User IDs**: `test-user-{n}`, `e2e-user-{n}`, `concurrent-user-{n}`
- **Market IDs**: Auto-generated UUIDs
- **Wallet IDs**: Auto-generated UUIDs
- **Settlement IDs**: Auto-generated UUIDs
- **Currencies**: `USD`, `EUR`
- **Time formats**: Unix timestamps

## Test Coverage

### Coverage Areas

1. **Functional Coverage**: All API endpoints and service methods
2. **Error Scenarios**: Invalid inputs, service failures, network issues
3. **Performance**: Concurrent operations, load testing
4. **Security**: CORS handling, input validation
5. **Integration**: Service-to-service communication, Kafka messaging
6. **Resilience**: Circuit breakers, retries, fallbacks

### Coverage Metrics

Run coverage reports:
```bash
# Generate coverage report
make coverage-report

# View HTML coverage report
open coverage.html

# Coverage summary
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## Continuous Integration

### CI Pipeline Integration

The test suite integrates with CI/CD pipelines through:

1. **Automated Test Execution**: Runs on every commit and pull request
2. **Coverage Reporting**: Generates and uploads coverage reports
3. **Performance Benchmarking**: Tracks performance regressions
4. **Smoke Tests**: Quick validation of critical functionality
5. **Parallel Execution**: Optimized for fast feedback

### GitHub Actions Example

```yaml
name: Test Suite
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      
      - name: Start Services
        run: docker-compose up -d
      
      - name: Wait for Services
        run: ./scripts/wait-for-services.sh
      
      - name: Run Tests
        run: make test-ci
      
      - name: Upload Coverage
        uses: codecov/codecov-action@v3
        with:
          file: ./coverage.out
```

## Troubleshooting

### Common Issues

1. **Services Not Starting**: Check Docker logs and port availability
2. **Kafka Connection Issues**: Verify Kafka is running and accessible
3. **Test Timeouts**: Increase timeout values for slower environments
4. **Port Conflicts**: Ensure required ports are available
5. **Environment Variables**: Verify all required variables are set

### Debug Mode

```bash
# Run tests with verbose output
make test-verbose

# Run specific test with debug logging
export LOG_LEVEL=debug
go test -run TestSpecificTest -v

# Enable gRPC debugging
export GRPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info
```

### Service Logs

```bash
# View service logs during testing
docker-compose logs -f api-gateway
docker-compose logs -f market-service

# Monitor Kafka topics during testing
kafka-console-consumer --bootstrap-server localhost:9092 --topic market.ListMarkets.fallback --from-beginning
```

## Best Practices

### Test Writing Guidelines

1. **Isolation**: Tests should be independent and not rely on state from other tests
2. **Idempotency**: Tests should be repeatable without manual cleanup
3. **Timeout Handling**: Always set appropriate timeouts for operations
4. **Error Handling**: Test both success and failure scenarios
5. **Resource Cleanup**: Properly close connections and clean up resources
6. **Mock Usage**: Use mocks for external dependencies in unit tests
7. **Real Services**: Use real services for integration and E2E tests

### Performance Considerations

1. **Parallel Execution**: Design tests to run in parallel where possible
2. **Resource Usage**: Monitor memory and CPU usage during test execution
3. **Database Cleanup**: Ensure test data doesn't accumulate
4. **Network Efficiency**: Minimize unnecessary network calls

### Maintenance

1. **Regular Updates**: Keep tests updated with API changes
2. **Flaky Test Detection**: Monitor and fix flaky tests promptly
3. **Test Data Management**: Use consistent and realistic test data
4. **Documentation**: Keep test documentation up to date

## Contributing

When adding new tests:

1. Follow existing test patterns and naming conventions
2. Add appropriate test documentation
3. Ensure tests are included in the appropriate Makefile targets
4. Verify tests work in both local and CI environments
5. Update this documentation with new test categories or changes

## Support

For test-related issues:
1. Check the troubleshooting section
2. Review service logs and test output
3. Verify environment configuration
4. Consult the development team