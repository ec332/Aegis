#!/bin/bash

# Aegis Test Runner Script
# This script provides an easy way to run different types of tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
TEST_TYPE="all"
VERBOSE=false
COVERAGE=false
PARALLEL=false
TIMEOUT="10m"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show help
show_help() {
    cat << EOF
Aegis Test Runner

Usage: $0 [OPTIONS]

OPTIONS:
    -t, --type TYPE     Test type: all, unit, integration, e2e, smoke (default: all)
    -v, --verbose       Run tests with verbose output
    -c, --coverage      Generate coverage report
    -p, --parallel      Run tests in parallel
    -T, --timeout TIME  Test timeout (default: 10m)
    -h, --help          Show this help message

EXAMPLES:
    $0                  # Run all tests
    $0 -t unit          # Run unit tests only
    $0 -t e2e -v        # Run E2E tests with verbose output
    $0 -t integration -c # Run integration tests with coverage
    $0 -t smoke -p      # Run smoke tests in parallel

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -t|--type)
            TEST_TYPE="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -c|--coverage)
            COVERAGE=true
            shift
            ;;
        -p|--parallel)
            PARALLEL=true
            shift
            ;;
        -T|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Validate test type
case $TEST_TYPE in
    all|unit|integration|e2e|smoke)
        ;;
    *)
        print_error "Invalid test type: $TEST_TYPE"
        print_error "Valid types: all, unit, integration, e2e, smoke"
        exit 1
        ;;
esac

# Function to check if services are running
check_services() {
    print_status "Checking if services are running..."
    
    local services=(
        "http://localhost:8080/health:API Gateway"
        "http://localhost:8081/health:Transaction Service"
    )
    
    local all_healthy=true
    
    for service in "${services[@]}"; do
        IFS=':' read -r url name <<< "$service"
        if curl -f -s "$url" > /dev/null 2>&1; then
            print_success "$name is healthy"
        else
            print_error "$name is not responding at $url"
            all_healthy=false
        fi
    done
    
    if [ "$all_healthy" = false ]; then
        print_warning "Some services are not running. Starting services with docker-compose..."
        if command -v docker-compose &> /dev/null; then
            docker-compose up -d
            sleep 10
        else
            print_error "docker-compose not found. Please start services manually."
            exit 1
        fi
    fi
}

# Function to run tests
run_tests() {
    local test_cmd="go test"
    local test_path=""
    
    # Build test command based on type
    case $TEST_TYPE in
        all)
            test_path="./..."
            ;;
        unit)
            test_path="./api-gateway/... ./shared/... ./market/... ./wallet/... ./settlement/... ./transaction-service/..."
            ;;
        integration)
            cd tests/integration
            test_path="."
            ;;
        e2e)
            cd tests/e2e
            test_path="."
            ;;
        smoke)
            test_cmd="go test -run TestHealth"
            test_path="./..."
            ;;
    esac
    
    # Add options
    if [ "$VERBOSE" = true ]; then
        test_cmd="$test_cmd -v"
    fi
    
    if [ "$COVERAGE" = true ]; then
        if [ "$TEST_TYPE" = "all" ] || [ "$TEST_TYPE" = "unit" ]; then
            test_cmd="$test_cmd -coverprofile=coverage.out -covermode=atomic"
        fi
    fi
    
    if [ "$PARALLEL" = true ]; then
        test_cmd="$test_cmd -parallel 4"
    fi
    
    # Add timeout
    test_cmd="$test_cmd -timeout $TIMEOUT"
    
    # Add test path
    test_cmd="$test_cmd $test_path"
    
    print_status "Running $TEST_TYPE tests..."
    print_status "Command: $test_cmd"
    
    # Execute test command
    if eval "$test_cmd"; then
        print_success "$TEST_TYPE tests completed successfully"
        
        # Generate coverage report if requested
        if [ "$COVERAGE" = true ] && ([ "$TEST_TYPE" = "all" ] || [ "$TEST_TYPE" = "unit" ]); then
            if [ -f "coverage.out" ]; then
                print_status "Generating coverage report..."
                go tool cover -html=coverage.out -o coverage.html
                print_success "Coverage report generated: coverage.html"
                
                # Show coverage summary
                local coverage_percent=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
                print_status "Total coverage: $coverage_percent"
            fi
        fi
    else
        print_error "$TEST_TYPE tests failed"
        exit 1
    fi
}

# Function to run benchmarks
run_benchmarks() {
    print_status "Running benchmark tests..."
    
    if go test -bench=. -benchmem ./...; then
        print_success "Benchmark tests completed successfully"
    else
        print_error "Benchmark tests failed"
        exit 1
    fi
}

# Main execution
main() {
    print_status "Starting Aegis test runner..."
    print_status "Test type: $TEST_TYPE"
    print_status "Verbose: $VERBOSE"
    print_status "Coverage: $COVERAGE"
    print_status "Parallel: $PARALLEL"
    print_status "Timeout: $TIMEOUT"
    echo
    
    # Check services for integration and e2e tests
    if [ "$TEST_TYPE" = "integration" ] || [ "$TEST_TYPE" = "e2e" ] || [ "$TEST_TYPE" = "all" ]; then
        check_services
        echo
    fi
    
    # Run tests
    run_tests
    
    # Run benchmarks if requested
    if [ "$TEST_TYPE" = "benchmark" ]; then
        run_benchmarks
    fi
    
    print_success "Test runner completed successfully!"
}

# Run main function
main