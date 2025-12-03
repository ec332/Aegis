# K6 Load Testing for Aegis Platform

This directory contains load tests for the Aegis prediction market platform using [k6](https://k6.io/).

## Prerequisites

Install k6:
```bash
# macOS
brew install k6

# Linux (Debian/Ubuntu)
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update && sudo apt-get install k6

# Docker
docker pull grafana/k6
```

## Test Scenarios

The test suite includes three scenarios:

| Scenario | Description | Duration | Max VUs |
|----------|-------------|----------|---------|
| **Smoke** | Quick validation of all endpoints | 30s | 1 |
| **Load** | Normal traffic simulation | 5m | 20 |
| **Stress** | High traffic with concurrent ops | 3m | 100 |

## Running Tests

### Full Test Suite
```bash
# From project root
k6 run tests/load/k6-load-test.js

# With custom base URL
k6 run -e BASE_URL=http://localhost:8080 tests/load/k6-load-test.js

# With custom wallet for auth
k6 run -e DEV_WALLET=0xMyTestWallet tests/load/k6-load-test.js
```

### Run Specific Scenario
```bash
# Smoke test only
k6 run --scenario smoke tests/load/k6-load-test.js

# Load test only
k6 run --scenario load tests/load/k6-load-test.js

# Stress test only
k6 run --scenario stress tests/load/k6-load-test.js
```

### Quick Smoke Test
```bash
k6 run --vus 1 --duration 30s tests/load/k6-load-test.js
```

### Using Docker
```bash
docker run --rm -i --network=host grafana/k6 run - <tests/load/k6-load-test.js
```

## Test Coverage

### Endpoints Tested

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/auth/dev-login` | POST | Dev authentication |
| `/api/markets` | GET | List markets |
| `/api/markets` | POST | Create market |
| `/api/markets/{id}` | GET | Get market details |
| `/api/markets/{id}/options` | GET | Get market options |
| `/api/wallets` | POST | Create wallet |
| `/api/wallets/user/{id}` | GET | Get wallet by user |
| `/api/wallets/{id}/deposit` | POST | Deposit funds |
| `/api/transactions` | POST | Create transaction (BUY/SELL) |
| `/api/transactions` | GET | List transactions |
| `/health` | GET | Health check |

### Flow Tested

1. **Authentication**: Dev login to get JWT token
2. **Market Creation**: Create prediction markets with options
3. **Market Browsing**: List and view market details
4. **Wallet Setup**: Create wallet and deposit funds
5. **Trading**: Buy and sell shares in markets
6. **Transaction History**: View transaction records

## Custom Metrics

| Metric | Description |
|--------|-------------|
| `auth_duration` | Authentication request duration |
| `create_market_duration` | Market creation duration |
| `list_markets_duration` | Market listing duration |
| `get_market_duration` | Single market fetch duration |
| `buy_transaction_duration` | Buy transaction duration |
| `sell_transaction_duration` | Sell transaction duration |
| `get_transactions_duration` | Transaction listing duration |
| `create_wallet_duration` | Wallet creation duration |
| `deposit_duration` | Deposit operation duration |
| `errors` | Error rate |
| `successful_requests` | Count of successful requests |
| `failed_requests` | Count of failed requests |

## Thresholds

Default thresholds (adjust in `options.thresholds`):

| Metric | Threshold |
|--------|-----------|
| `http_req_duration` | p(95) < 1000ms, p(99) < 2000ms |
| `errors` | rate < 10% |
| `http_req_failed` | rate < 5% |
| `create_market_duration` | p(95) < 800ms |
| `buy_transaction_duration` | p(95) < 500ms |
| `sell_transaction_duration` | p(95) < 500ms |
| `list_markets_duration` | p(95) < 300ms |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BASE_URL` | `http://localhost:8080` | API gateway base URL |
| `DEV_WALLET` | `0xTESTUSER` | Mock wallet for dev auth |

## Output & Reporting

### Console Output (Default)
```bash
k6 run tests/load/k6-load-test.js
```

### JSON Output
```bash
k6 run --out json=results.json tests/load/k6-load-test.js
```

### CSV Output
```bash
k6 run --out csv=results.csv tests/load/k6-load-test.js
```

### InfluxDB + Grafana (Advanced)
```bash
# Start InfluxDB
docker run -d -p 8086:8086 influxdb:1.8

# Run with InfluxDB output
k6 run --out influxdb=http://localhost:8086/k6 tests/load/k6-load-test.js
```

## Real-Time Dashboard (Recommended)

### Option 1: k6 Web Dashboard
```bash
# Run with web dashboard enabled
K6_WEB_DASHBOARD=true k6 run tests/load/k6-load-test.js

# Or with explicit port
K6_WEB_DASHBOARD=true K6_WEB_DASHBOARD_PORT=5665 k6 run tests/load/k6-load-test.js
```
Then open **http://localhost:5665** in your browser.

### Option 2: Grafana + InfluxDB Stack (Full Dashboard)

1. **Start the monitoring stack:**
```bash
# Create a docker-compose-monitoring.yml in tests/load/
docker compose -f tests/load/docker-compose-monitoring.yml up -d
```

2. **Run k6 with InfluxDB output:**
```bash
k6 run --out influxdb=http://localhost:8086/k6 tests/load/k6-load-test.js
```

3. **Open Grafana:** http://localhost:3001 (admin/admin)

4. **Import k6 dashboard:** Dashboard ID `2587` from Grafana.com

### Option 3: k6 Cloud (SaaS)
```bash
# Login to k6 cloud
k6 login cloud

# Run with cloud output
k6 run --out cloud tests/load/k6-load-test.js
```
View results at https://app.k6.io

## Troubleshooting

### Common Issues

1. **Auth failures**: Ensure `/auth/dev-login` endpoint is enabled and `AUTH_DEV_LOGIN_ENABLED=true`
2. **Connection refused**: Ensure services are running (`docker compose up`)
3. **High error rate**: Check service logs (`docker compose logs`)
4. **Insufficient balance**: Tests auto-deposit funds, but check wallet service logs

### Debug Mode
```bash
# Verbose output
k6 run --http-debug tests/load/k6-load-test.js

# With request/response logging
k6 run --http-debug=full tests/load/k6-load-test.js
```

## Tips for CI/CD

```yaml
# GitHub Actions example
- name: Run k6 Load Tests
  uses: grafana/k6-action@v0.3.0
  with:
    filename: tests/load/k6-load-test.js
    flags: --out json=results.json
  env:
    BASE_URL: ${{ secrets.API_URL }}
```
