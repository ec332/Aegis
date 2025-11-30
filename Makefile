protos:
	docker run --rm -v $(PWD):/work -w /work golang:1.22-alpine sh -c 'apk add --no-cache git protobuf && go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0 && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0 && PATH=/go/bin:$$PATH protoc -I proto --go_out=paths=source_relative:proto/gen --go-grpc_out=paths=source_relative:proto/gen proto/market/market.proto proto/settlement/settlement.proto proto/wallet/wallet.proto'

protos-local:
	protoc -I proto --go_out=paths=source_relative:proto/gen --go-grpc_out=paths=source_relative:proto/gen proto/market/market.proto proto/settlement/settlement.proto proto/wallet/wallet.proto

# Test targets
.PHONY: test test-unit test-integration test-e2e test-all test-coverage test-parallel test-local test-ci test-smoke test-verbose coverage-report

# Run all tests
test-all:
	@echo "Running all tests..."
	@go test -v ./...

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	@go test -v ./api-gateway/... ./shared/... ./market/... ./wallet/... ./settlement/... ./transaction-service/...

# Run integration tests only
test-integration:
	@echo "Running integration tests..."
	@cd tests/integration && go test -v

# Run end-to-end tests only
test-e2e:
	@echo "Running end-to-end tests..."
	@cd tests/e2e && go test -v -timeout 10m

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests in parallel
test-parallel:
	@echo "Running tests in parallel..."
	@go test -parallel 4 -v ./...

# Run tests for local development
test-local:
	@echo "Running local tests..."
	@CORS_ORIGINS="http://localhost:3000,http://127.0.0.1:3000" \
	KAFKA_BROKERS="localhost:9092" \
	go test -v ./...

# Run tests for CI/CD
test-ci:
	@echo "Running CI tests..."
	@go test -race -coverprofile=coverage.out -covermode=atomic -v ./...
	@go tool cover -func=coverage.out

# Run smoke tests only
test-smoke:
	@echo "Running smoke tests..."
	@go test -run TestHealth -v ./...
	@cd tests/integration && go test -run TestHealthCheckIntegration -v

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	@go test -v -count=1 ./...

# Generate coverage report
coverage-report:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out -covermode=atomic ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

# Wait for services to be ready
wait-for-services:
	@echo "Waiting for services to be ready..."
	@timeout 60 bash -c 'until curl -f http://localhost:8080/health; do sleep 2; done' || (echo "API Gateway not ready" && exit 1)
	@timeout 60 bash -c 'until curl -f http://localhost:8081/health; do sleep 2; done' || (echo "Transaction Service not ready" && exit 1)
	@echo "All services are ready"

# Run tests with service dependencies
test-with-services: wait-for-services
	@echo "Services are ready, running tests..."
	@make test-all

# Clean test artifacts
test-clean:
	@echo "Cleaning test artifacts..."
	@rm -f coverage.out coverage.html
	@find . -name "*.test" -delete
	@find . -name "test.log" -delete

# Benchmark tests
.PHONY: benchmark
benchmark:
	@echo "Running benchmark tests..."
	@go test -bench=. -benchmem ./...

# Lint and format before testing
.PHONY: lint format
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

format:
	@echo "Formatting code..."
	@go fmt ./...

# Pre-test checks
pre-test: lint format
	@echo "Pre-test checks completed"

# Complete test pipeline
test-pipeline: pre-test test-coverage
	@echo "Test pipeline completed successfully"