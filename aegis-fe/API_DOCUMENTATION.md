# Aegis Frontend API Integration Documentation

## Overview

This document describes the frontend integration with the Aegis backend API Gateway. The frontend communicates with the backend via HTTP/REST endpoints, while the backend uses gRPC for interservice communication with circuit breaker and Kafka fallback mechanisms.

## Architecture

```
Frontend (Next.js) → API Gateway (HTTP/REST) → Backend Services (gRPC)
```

- **Frontend**: Next.js with TypeScript, Zustand for state management
- **API Gateway**: HTTP endpoints that translate to gRPC calls
- **Backend Services**: Market, Wallet, and Settlement services with gRPC
- **Resilience**: Circuit breaker with 1-second timeout, automatic Kafka fallback

## API Endpoints

### Market Endpoints

#### `GET /api/markets`
- **Description**: List all markets
- **Response**: `{ markets: Market[] }`
- **Error Handling**: Automatic retry with exponential backoff
- **Example**:
```typescript
const markets = await fetchMarkets();
```

#### `GET /api/markets/{id}`
- **Description**: Get market by ID
- **Response**: `{ market: Market }`
- **Example**:
```typescript
const market = await fetchMarketById('market-id');
```

#### `POST /api/markets`
- **Description**: Create a new market
- **Request Body**: 
```json
{
  "question": "Will Bitcoin reach $100k?",
  "description": "Predict if BTC will hit $100k by end of 2024",
  "category": "Crypto",
  "end_time": "2024-12-31T23:59:59Z"
}
```
- **Response**: `{ market: Market }`
- **Example**:
```typescript
const newMarket = await createMarket({
  question: "Will Bitcoin reach $100k?",
  description: "Predict if BTC will hit $100k by end of 2024",
  category: "Crypto",
  end_time: "2024-12-31T23:59:59Z"
});
```

#### `PUT /api/markets/{id}`
- **Description**: Update market
- **Request Body**: Partial market data
- **Response**: `{ market: Market }`

#### `GET /api/markets/{id}/options`
- **Description**: Get options for a market
- **Response**: `{ options: Option[] }`

### Wallet Endpoints

#### `POST /api/wallets`
- **Description**: Create a new wallet account
- **Request Body**: 
```json
{
  "user_id": "user-123",
  "currency": "USD"
}
```
- **Response**: `{ account: WalletAccount }`
- **Example**:
```typescript
const wallet = await createWallet('user-123', 'USD');
```

#### `GET /api/wallets/{id}`
- **Description**: Get wallet account by ID
- **Response**: `{ account: WalletAccount }`

#### `POST /api/wallets/{id}/deposit`
- **Description**: Deposit funds to wallet
- **Request Body**: 
```json
{
  "account_id": "wallet-123",
  "amount": 100.50,
  "reference_id": "deposit-001"
}
```
- **Response**: `{ transaction: WalletTransaction }`
- **Example**:
```typescript
const transaction = await deposit(walletId, 100.50, 'deposit-001');
```

#### `POST /api/wallets/{id}/withdraw`
- **Description**: Withdraw funds from wallet
- **Request Body**: 
```json
{
  "account_id": "wallet-123",
  "amount": 50.25,
  "reference_id": "withdraw-001"
}
```
- **Response**: `{ transaction: WalletTransaction }`

### Settlement Endpoints

#### `POST /api/settlements`
- **Description**: Create a settlement for a market
- **Request Body**: 
```json
{
  "market_id": "market-123",
  "winning_option_id": "option-456"
}
```
- **Response**: `{ settlement: Settlement }`

#### `GET /api/settlements/{id}`
- **Description**: Get settlement by ID
- **Response**: `{ settlement: Settlement }`

#### `PUT /api/settlements/{id}/complete`
- **Description**: Complete a settlement
- **Response**: `{ settlement: Settlement }`

### Health Check

#### `GET /health`
- **Description**: Check API Gateway health
- **Response**: `{ status: "healthy" }`

## Data Types

### Market
```typescript
interface Market {
  id: string;
  question: string;
  description: string;
  category: string;
  end_time?: string;
  resolution_time?: string;
  status: string;
  outcome?: string;
  created_at?: string;
  updated_at?: string;
}
```

### Option
```typescript
interface Option {
  id: string;
  market_id: string;
  option_text: string;
  current_price: number;
  volume: number;
  created_at?: string;
  updated_at?: string;
}
```

### WalletAccount
```typescript
interface WalletAccount {
  id: string;
  user_id: string;
  address: string;
  currency: string;
  total_balance: number;
  available_balance: number;
  status: string;
  created_at?: string;
  updated_at?: string;
}
```

### WalletTransaction
```typescript
interface WalletTransaction {
  id: string;
  wallet_id: string;
  market_id?: string;
  type: string; // "deposit", "withdrawal", "debit", "credit"
  amount: number;
  status: string;
  reference_id?: string;
  metadata?: string;
  created_at?: string;
  updated_at?: string;
}
```

### Settlement
```typescript
interface Settlement {
  id: string;
  market_id: string;
  winning_option_id: string;
  total_pool: number;
  winning_pool: number;
  status: string;
  settled_at?: string;
  created_at?: string;
  updated_at?: string;
}
```

## Error Handling

### API Error Responses
All endpoints return consistent error responses:

```typescript
interface APIError {
  status: number;
  message: string;
  details?: any;
}
```

### Error Types
- **400 Bad Request**: Invalid request data
- **404 Not Found**: Resource not found
- **500 Internal Server Error**: Server error
- **202 Accepted**: Request queued for async processing (circuit breaker fallback)

### Client-Side Error Handling
The API service includes:
- **Automatic retry** with exponential backoff (3 attempts)
- **Circuit breaker awareness** (handles 202 responses)
- **Network error handling** with user-friendly messages
- **Timeout handling** with graceful degradation

## State Management

### Zustand Store Structure
```typescript
interface AppState {
  // Markets
  markets: Market[];
  marketOptions: { [key: string]: Option[] };
  
  // Wallet
  walletAccounts: WalletAccount[];
  walletTransactions: WalletTransaction[];
  currentWallet: WalletAccount | null;
  
  // Settlements
  settlements: Settlement[];
  
  // Loading states
  isLoadingMarkets: boolean;
  isLoadingOptions: boolean;
  isLoadingWallet: boolean;
  isLoadingSettlements: boolean;
  
  // Error states
  error: APIError | null;
  isBackendHealthy: boolean;
}
```

### Key Actions
- `initializeApp()`: Load initial data and check backend health
- `loadMarkets()`: Fetch markets from API
- `createMarket()`: Create new market
- `createWallet()`: Create wallet account
- `depositFunds()`: Deposit to wallet
- `withdrawFunds()`: Withdraw from wallet

## Configuration

### Environment Variables
```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### Default Configuration
- **API Base URL**: `http://localhost:8080`
- **Retry Attempts**: 3
- **Retry Delay**: 1000ms (exponential backoff)
- **Health Check Interval**: 30 seconds

## Usage Examples

### Initialize Application
```typescript
import { useAppStore } from '@/store/appStore';

function App() {
  const { initializeApp, isBackendHealthy } = useAppStore();
  
  useEffect(() => {
    initializeApp();
  }, []);
  
  if (!isBackendHealthy) {
    return <div>Backend connection issues detected...</div>;
  }
  
  return <div>App loaded successfully</div>;
}
```

### Create and Fund Wallet
```typescript
const { createWallet, depositFunds } = useAppStore();

// Create wallet
const wallet = await createWallet('user-123', 'USD');

// Deposit funds
const transaction = await depositFunds(wallet.id, 100.00, 'initial-deposit');
```

### Create Market
```typescript
const { createMarket } = useAppStore();

const market = await createMarket({
  question: "Will it rain tomorrow?",
  description: "Weather prediction for New York City",
  category: "Weather",
  end_time: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString()
});
```

## Testing

### Running Tests
```bash
# Start backend services
docker compose up

# Run frontend development server
cd aegis-fe
npm run dev

# Test API endpoints
curl http://localhost:8080/health
curl http://localhost:8080/api/markets
```

### Test Scenarios
1. **Backend Health**: Verify `/health` endpoint responds
2. **Market Operations**: Create, read, update markets
3. **Wallet Operations**: Create wallet, deposit, withdraw
4. **Error Handling**: Test network failures, timeouts
5. **Circuit Breaker**: Verify Kafka fallback on timeouts

## Troubleshooting

### Common Issues

1. **CORS Errors**
   - Ensure API Gateway is configured for CORS
   - Check `Access-Control-Allow-Origin` headers

2. **Connection Timeouts**
   - Verify backend services are running
   - Check Docker Compose network configuration

3. **Circuit Breaker Fallback**
   - Look for 202 Accepted responses
   - Check Kafka logs for fallback messages

4. **Data Not Loading**
   - Verify API Gateway is accessible
   - Check browser console for errors
   - Test endpoints directly with curl

### Debug Information
- Enable browser developer tools
- Check Network tab for API calls
- Monitor console logs for retry attempts
- Use health check endpoint to verify connectivity

## Future Enhancements

### Planned Features
- WebSocket support for real-time updates
- Advanced filtering and search
- User authentication and authorization
- Transaction history pagination
- Market analytics dashboard

### Performance Optimizations
- Implement React Query for caching
- Add request debouncing
- Optimize bundle size
- Implement service worker for offline support