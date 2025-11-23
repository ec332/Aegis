# Aegis Prediction Market Platform

A decentralized prediction market platform built with Go microservices architecture, enabling users to create, trade, and settle prediction markets on various topics.

## 🏗️ Architecture Overview

The Aegis platform follows a microservices architecture with the following services:

### Core Services

1. **API Gateway** (Port 8080)
   - Central entry point for all client requests
   - Routes requests to appropriate microservices
   - Handles load balancing and service discovery
   - Provides unified API interface

2. **Market Service** (Port 8081)
   - Manages prediction markets and options
   - Handles market creation, updates, and queries
   - Integrates with Redis for real-time updates
   - Manages user accounts and authentication

3. **Wallet Service** (Port 8082)
   - Handles all financial transactions
   - Manages user wallet accounts and balances
   - Processes deposits, withdrawals, and transfers
   - Supports USDC currency with decimal precision

4. **Settlement Service** (Port 8084)
   - Manages market settlement and payout distribution
   - Calculates winning pools and user payouts
   - Handles settlement completion and distribution
   - Integrates with wallet service for payouts

5. **Transaction Service** (Port 5555)
   - Legacy service for basic transaction processing
   - Handles transaction creation and management

### Infrastructure

- **PostgreSQL**: Primary database for all services
- **Redis**: Caching and real-time updates
- **Docker Compose**: Service orchestration and deployment

## 🚀 Quick Start

### Prerequisites

- Go 1.22+
- Docker and Docker Compose
- PostgreSQL client tools (optional)

### Installation

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd Aegis
   ```

2. **Start all services with Docker Compose:**
   ```bash
   docker-compose up -d
   ```

3. **Wait for services to be ready:**
   ```bash
   ./test-system.sh
   ```

4. **Verify services are running:**
   ```bash
   curl http://localhost:8080/health
   ```

## 📋 API Documentation

### Base URL
All API requests should be made to the API Gateway:
```
http://localhost:8080/api/v1/
```

### Authentication
Currently using wallet-based authentication with nonce generation.

### Endpoints

#### Users
- `POST /users` - Create a new user
- `GET /users/{id}` - Get user by ID
- `GET /users/wallet/{address}` - Get user by wallet address
- `PUT /users/{id}` - Update user information

#### Markets
- `POST /markets` - Create a new prediction market
- `GET /markets/{id}` - Get market details
- `GET /markets` - List all markets
- `PUT /markets/{id}` - Update market information

#### Options
- `POST /options` - Create market option
- `GET /options/{id}` - Get option details
- `GET /options/market/{marketId}` - Get options for a market

#### Wallets
- `POST /wallets` - Create wallet account
- `GET /wallets/{accountId}` - Get wallet account
- `GET /wallets/user/{userId}` - Get wallet by user ID
- `PUT /wallets/{accountId}` - Update wallet account
- `POST /wallets/{accountId}/deposit` - Deposit funds
- `POST /wallets/{accountId}/withdrawal` - Withdraw funds
- `POST /wallets/{accountId}/debit` - Debit wallet
- `POST /wallets/{accountId}/credit` - Credit wallet
- `GET /wallets/{accountId}/transactions` - Get wallet transactions

#### Settlements
- `POST /settlements` - Create settlement
- `GET /settlements/{id}` - Get settlement
- `GET /settlements/market/{marketId}` - Get settlement by market
- `POST /settlements/{id}/complete` - Complete settlement

#### Transactions
- `GET /transactions/{id}` - Get transaction
- `POST /transactions/{id}/settle` - Settle transaction

### Example Usage

#### Create a User
```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "wallet_address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb7",
    "balance": 1000.0,
    "role": "user"
  }'
```

#### Create a Market
```bash
curl -X POST http://localhost:8080/api/v1/markets \
  -H "Content-Type: application/json" \
  -d '{
    "question": "Will Bitcoin price exceed $100,000 by end of 2024?",
    "description": "Bitcoin price prediction market",
    "category": "cryptocurrency",
    "end_time": "2024-12-31T23:59:59Z"
  }'
```

#### Create Wallet Account
```bash
curl -X POST http://localhost:8080/api/v1/wallets \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user-uuid",
    "currency": "USDC"
  }'
```

#### Deposit Funds
```bash
curl -X POST http://localhost:8080/api/v1/wallets/{accountId}/deposit \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 500.0,
    "reference_id": "deposit_001"
  }'
```

## 🧪 Testing

### Run System Tests
Execute the comprehensive test script:
```bash
./test-system.sh
```

This script will:
- Verify all services are running
- Test all API endpoints
- Validate service communication
- Check database connectivity

### Individual Service Testing
Each service has its own health check endpoint:
```bash
# API Gateway
curl http://localhost:8080/health

# Market Service
curl http://localhost:8081/health

# Wallet Service
curl http://localhost:8082/health

# Settlement Service
curl http://localhost:8084/health
```

## 🗄️ Database Schema

### Users Table
```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    wallet_address VARCHAR(255) NOT NULL UNIQUE,
    balance DECIMAL(20, 8) NOT NULL DEFAULT 0,
    nonce TEXT NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### Markets Table
```sql
CREATE TABLE markets (
    id UUID PRIMARY KEY,
    question TEXT NOT NULL,
    description TEXT,
    category VARCHAR(100),
    end_time TIMESTAMP NOT NULL,
    resolution_time TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    outcome VARCHAR(50),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### Wallet Accounts Table
```sql
CREATE TABLE wallet_accounts (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    address VARCHAR(255) NOT NULL UNIQUE,
    currency VARCHAR(10) NOT NULL DEFAULT 'USDC',
    total_balance DECIMAL(20, 8) NOT NULL DEFAULT 0,
    available_balance DECIMAL(20, 8) NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### Settlements Table
```sql
CREATE TABLE settlements (
    id UUID PRIMARY KEY,
    market_id UUID NOT NULL,
    winning_option_id UUID NOT NULL,
    total_pool DECIMAL(20, 8) NOT NULL,
    winning_pool DECIMAL(20, 8) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    settled_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

## 🔧 Configuration

### Environment Variables

Each service supports the following environment variables:

#### Database Configuration
- `DB_HOST`: PostgreSQL host (default: localhost)
- `DB_PORT`: PostgreSQL port (default: 5432)
- `DB_NAME`: Database name
- `DB_USER`: Database user (default: postgres)
- `DB_PASSWORD`: Database password (default: postgres)
- `DB_SSLMODE`: SSL mode (default: disable)

#### Service Configuration
- `PORT`: Service port
- `REDIS_HOST`: Redis host (for Market Service)
- `REDIS_PORT`: Redis port (for Market Service)

#### API Gateway Configuration
- `MARKET_SERVICE_URL`: Market service URL
- `WALLET_SERVICE_URL`: Wallet service URL
- `SETTLEMENT_SERVICE_URL`: Settlement service URL
- `TRANSACTION_SERVICE_URL`: Transaction service URL

## 🚢 Deployment

### Docker Compose Deployment
```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down

# Rebuild and restart
docker-compose down && docker-compose up --build -d
```

### Individual Service Deployment
Each service can be deployed independently:
```bash
# Build service
cd market && go build ./cmd/main.go

# Run service
cd market && ./main
```

## 🔮 Future Enhancements

### Planned Features
- **Resolution Service**: External oracle integration for market resolution
- **Real-time Updates**: WebSocket support for live market data
- **Advanced Analytics**: Market analytics and user insights
- **Multi-chain Support**: Integration with multiple blockchain networks
- **Mobile App**: Native mobile applications
- **Advanced Trading**: Limit orders, stop-loss, and other trading features

### External Integrations
- **Fireblocks**: Institutional custody and wallet management
- **Chainlink**: Decentralized oracle network for market resolution
- **Sema**: Alternative oracle provider
- **Polygon**: Layer 2 scaling solution

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 📞 Support

For support and questions:
- Create an issue in the GitHub repository
- Check the documentation and examples
- Review the test cases for usage patterns

---

**Aegis Prediction Market Platform** - Empowering decentralized prediction markets with robust microservices architecture.
## Proto Generation

- Pinned plugins: `protoc-gen-go v1.33.0`, `protoc-gen-go-grpc v1.3.0`
- Dockerized one-liner:
  - `make protos`
- Local (requires `protoc` and plugins installed):
  - `make protos-local`

Generated files are written to `proto/gen` with source-relative paths.

### Local setup (if not using Docker)

- Install prerequisites:
  - `brew install go protobuf`
  - `go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.33.0`
  - `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0`
  - Ensure `~/go/bin` is on your `PATH`
- Generate:
  - `make protos-local`

### Notes

- Keep the generator versions pinned to avoid runtime incompatibilities
- Commit changes under `proto/gen` when proto definitions change

## Docker Compose

- The Compose `version:` field is obsolete and has been removed to silence warnings
- Use `docker compose up -d` and `docker compose build` as usual