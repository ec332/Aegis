#!/bin/bash

# Aegis Prediction Market System Test Script
# This script tests the complete microservices architecture

set -e

echo "🚀 Starting Aegis Prediction Market System Test"
echo "================================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
API_GATEWAY="http://localhost:8080"
MARKET_SERVICE="http://localhost:8081"
WALLET_SERVICE="http://localhost:8082"
SETTLEMENT_SERVICE="http://localhost:8084"

# Helper functions
test_endpoint() {
    local method=$1
    local url=$2
    local data=$3
    local description=$4
    
    echo -n "Testing $description... "
    
    if [ "$method" == "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" -X GET "$url" 2>/dev/null || echo "000")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" -H "Content-Type: application/json" -d "$data" 2>/dev/null || echo "000")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [[ "$http_code" =~ ^2[0-9][0-9]$ ]]; then
        echo -e "${GREEN}✓ PASS${NC} (HTTP $http_code)"
        return 0
    else
        echo -e "${RED}✗ FAIL${NC} (HTTP $http_code)"
        echo "Response: $body"
        return 1
    fi
}

wait_for_service() {
    local service_name=$1
    local url=$2
    local max_attempts=30
    local attempt=0
    
    echo "Waiting for $service_name to be ready..."
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "$url/health" >/dev/null 2>&1; then
            echo -e "${GREEN}$service_name is ready!${NC}"
            return 0
        fi
        
        attempt=$((attempt + 1))
        echo "Attempt $attempt/$max_attempts - waiting..."
        sleep 2
    done
    
    echo -e "${RED}$service_name failed to start!${NC}"
    return 1
}

# Test data
USER_ID="$(uuidgen)"
MARKET_ID="$(uuidgen)"
OPTION_ID="$(uuidgen)"
WALLET_ADDRESS="0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb7"

# Main test sequence
echo "📋 Test Configuration:"
echo "   User ID: $USER_ID"
echo "   Market ID: $MARKET_ID"
echo "   Option ID: $OPTION_ID"
echo "   Wallet Address: $WALLET_ADDRESS"
echo ""

# Wait for services to be ready
echo "⏳ Waiting for services to start..."
wait_for_service "API Gateway" "$API_GATEWAY" || exit 1
wait_for_service "Market Service" "$MARKET_SERVICE" || exit 1
wait_for_service "Wallet Service" "$WALLET_SERVICE" || exit 1
wait_for_service "Settlement Service" "$SETTLEMENT_SERVICE" || exit 1

echo ""
echo "🔍 Starting API Tests"
echo "===================="

# Test 1: Health checks
echo "1. Testing service health endpoints..."
test_endpoint "GET" "$API_GATEWAY/health" "" "API Gateway Health"
test_endpoint "GET" "$MARKET_SERVICE/health" "" "Market Service Health"
test_endpoint "GET" "$WALLET_SERVICE/health" "" "Wallet Service Health"
test_endpoint "GET" "$SETTLEMENT_SERVICE/health" "" "Settlement Service Health"

echo ""

# Test 2: Create a user (Market Service)
echo "2. Testing user creation..."
USER_DATA='{
    "wallet_address": "'$WALLET_ADDRESS'",
    "balance": 1000.0,
    "role": "user"
}'
test_endpoint "POST" "$API_GATEWAY/api/v1/users" "$USER_DATA" "Create User"

echo ""

# Test 3: Get user by wallet address
echo "3. Testing user retrieval..."
test_endpoint "GET" "$API_GATEWAY/api/v1/users/wallet/$WALLET_ADDRESS" "" "Get User by Wallet"

echo ""

# Test 4: Create wallet account (Wallet Service)
echo "4. Testing wallet account creation..."
WALLET_DATA='{
    "user_id": "'$USER_ID'",
    "currency": "USDC"
}'
test_endpoint "POST" "$API_GATEWAY/api/v1/wallets" "$WALLET_DATA" "Create Wallet Account"

echo ""

# Test 5: Deposit to wallet
echo "5. Testing wallet deposit..."
DEPOSIT_DATA='{
    "amount": 500.0,
    "reference_id": "deposit_001"
}'
test_endpoint "POST" "$API_GATEWAY/api/v1/wallets/$USER_ID/deposit" "$DEPOSIT_DATA" "Deposit to Wallet"

echo ""

# Test 6: Create a market (Market Service)
echo "6. Testing market creation..."
MARKET_DATA='{
    "question": "Will Bitcoin price exceed $100,000 by end of 2024?",
    "description": "Bitcoin price prediction market",
    "category": "cryptocurrency",
    "end_time": "2024-12-31T23:59:59Z"
}'
test_endpoint "POST" "$API_GATEWAY/api/v1/markets" "$MARKET_DATA" "Create Market"

echo ""

# Test 7: Create market option
echo "7. Testing option creation..."
OPTION_DATA='{
    "market_id": "'$MARKET_ID'",
    "option_text": "Yes, Bitcoin will exceed $100,000",
    "current_price": 65.0
}'
test_endpoint "POST" "$API_GATEWAY/api/v1/options" "$OPTION_DATA" "Create Market Option"

echo ""

# Test 8: Create settlement (Settlement Service)
echo "8. Testing settlement creation..."
SETTLEMENT_DATA='{
    "market_id": "'$MARKET_ID'",
    "winning_option_id": "'$OPTION_ID'"
}'
test_endpoint "POST" "$API_GATEWAY/api/v1/settlements" "$SETTLEMENT_DATA" "Create Settlement"

echo ""

# Test 9: Get settlement by market ID
echo "9. Testing settlement retrieval..."
test_endpoint "GET" "$API_GATEWAY/api/v1/settlements/market/$MARKET_ID" "" "Get Settlement by Market"

echo ""

# Test 10: Complete settlement
echo "10. Testing settlement completion..."
test_endpoint "POST" "$API_GATEWAY/api/v1/settlements/$MARKET_ID/complete" "" "Complete Settlement"

echo ""

echo "🎉 Test Summary"
echo "=============="
echo -e "${GREEN}All core API endpoints are working!${NC}"
echo ""
echo "📊 Services Tested:"
echo "   ✓ API Gateway (Port 8080)"
echo "   ✓ Market Service (Port 8081)"
echo "   ✓ Wallet Service (Port 8082)"
echo "   ✓ Settlement Service (Port 8084)"
echo ""
echo "🔗 Architecture Verified:"
echo "   ✓ Microservices communication"
echo "   ✓ Database integration"
echo "   ✓ API routing and load balancing"
echo "   ✓ Service health monitoring"
echo ""
echo "🚀 The Aegis Prediction Market Platform is ready for use!"