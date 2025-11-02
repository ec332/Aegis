# Market Service

## Features

- **Market Management**: Create, update, and list prediction markets
- **Options & Liquidity Tracking**: Each market has multiple options with liquidity pools
- **Real-time Updates**: Server-Sent Events (SSE) streaming via Redis pub/sub
- **Status Management**: Track market lifecycle (draft → active → resolving → resolved)
- **PostgreSQL Storage**: Persistent storage for markets, options, and liquidity pools
- **Redis Pub/Sub**: Real-time event distribution for liquidity updates
- **RESTful API**: Clean HTTP endpoints for all operations

## Architecture

Standard Go project layout with clean separation of concerns:

```
market/
├── cmd/
│   └── main.go              # Application entry point
├── internal/
│   ├── api/
│   │   └── handler.go       # HTTP handlers & routing
│   ├── service/
│   │   └── service.go        # Business logic
│   ├── repository/
│   │   └── repository.go    # Database operations
│   └── middleware/
│       └── middleware.go    # HTTP middleware
├── pkg/
│   ├── models/
│   │   └── models.go        # Data models
│   └── config/
│       └── config.go        # Configuration
├── scripts/                  # Database setup scripts
├── go.mod
└── Dockerfile
```

## Quick Start

### 1. Setup Database
```bash
cd scripts
./setup_db.sh
./seed.sh
```
### 2. Configure Environment
```bash
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/aegis?sslmode=disable"
export REDIS_URL="redis://localhost:6379"
export PORT=8080
```

### 3. Start Redis (if not running)
```bash
# Check if Redis is running
redis-cli ping

# Start Redis (macOS with Homebrew)
brew services start redis

# Or run in foreground
redis-server
```

### 4. Run the Service
```bash
cd market
go run cmd/main.go
```

Or build and run:
```bash
go build -o ./bin/market cmd/main.go
./bin/market
```

Or use Docker:
```bash
docker build -t market-service .
docker run -p 8080:8080 \
  -e DATABASE_URL="postgres://..." \
  market-service
```

## API Endpoints

**Available Endpoints:**
- `GET /health` - Health check
- `POST /markets` - Create market
- `GET /markets` - List markets (with optional status filter)
- `GET /markets/{marketId}` - Get specific market
- `PUT /markets/{marketId}` - Update market
- `GET /markets/{marketId}/stream` - SSE stream for real-time liquidity updatesmarket
- `PUT /markets/{marketId}` - Update market

## Data Models

### Market
- Contains title, description, status, and resolution details
- Links to multiple options and liquidity pools

### Option
- Represents a tradeable outcome (e.g., "Yes", "No")
- Each market has 2+ options

### LiquidityPool
- Tracks liquidity value for each option

## Environment Variables
- `PORT`: HTTP server port (default: 8080)
- `DATABASE_URL`: PostgreSQL connection string
- `REDIS_URL`: Redis connection string (default: redis://localhost:6379)

## Redis Integration

The service uses Redis pub/sub for real-time liquidity updates:

- **Channel format**: `market:{marketId}:liquidity`
- **When updates are published**:
  - Market creation
  - Market updates
  - Liquidity pool changes

## Testing

```bash
# Run tests
go test ./...

# Check service health
curl http://localhost:8080/health

# Test SSE streaming
curl -N http://localhost:8080/markets/{marketId}/stream

# Monitor Redis pub/sub
redis-cli SUBSCRIBE market:{marketId}:liquidity
```
