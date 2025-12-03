import http from 'k6/http';
import { check, sleep, group, fail } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// ============================================================================
// CUSTOM METRICS
// ============================================================================
const errorRate = new Rate('errors');
const authDuration = new Trend('auth_duration', true);
const createMarketDuration = new Trend('create_market_duration', true);
const listMarketsDuration = new Trend('list_markets_duration', true);
const getMarketDuration = new Trend('get_market_duration', true);
const buyTransactionDuration = new Trend('buy_transaction_duration', true);
const sellTransactionDuration = new Trend('sell_transaction_duration', true);
const getTransactionsDuration = new Trend('get_transactions_duration', true);
const createWalletDuration = new Trend('create_wallet_duration', true);
const depositDuration = new Trend('deposit_duration', true);
const successfulRequests = new Counter('successful_requests');
const failedRequests = new Counter('failed_requests');

// ============================================================================
// TEST CONFIGURATION
// ============================================================================
export const options = {
  scenarios: {
    // Smoke test - quick validation
    smoke: {
      executor: 'constant-vus',
      vus: 1,
      duration: '30s',
      startTime: '0s',
      tags: { scenario: 'smoke' },
      exec: 'smokeTest',
    },
    // Load test - normal traffic
    load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '1m', target: 10 },   // Ramp up
        { duration: '3m', target: 20 },   // Steady state
        { duration: '1m', target: 0 },    // Ramp down
      ],
      startTime: '35s',
      tags: { scenario: 'load' },
      exec: 'loadTest',
    },
    // Stress test - high traffic
    stress: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 30 },
        { duration: '1m', target: 50 },
        { duration: '30s', target: 80 },
        { duration: '1m', target: 100 },
        { duration: '30s', target: 0 },
      ],
      startTime: '6m',
      tags: { scenario: 'stress' },
      exec: 'stressTest',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<2000', 'p(99)<5000'],
    errors: ['rate<0.3'],
    http_req_failed: ['rate<0.2'],
    create_market_duration: ['p(95)<2000'],
    buy_transaction_duration: ['p(95)<1500'],
    sell_transaction_duration: ['p(95)<1500'],
    list_markets_duration: ['p(95)<1000'],
  },
};

// ============================================================================
// ENVIRONMENT CONFIGURATION
// ============================================================================
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const DEV_WALLET = __ENV.DEV_WALLET || '0xTESTUSER';

// Sample market questions for variety
const marketQuestions = [
  'Will Bitcoin reach $100k by end of 2025?',
  'Will ETH 2.0 launch successfully?',
  'Will the Fed cut interest rates?',
  'Will AI replace 50% of jobs by 2030?',
  'Will Tesla stock double in 2025?',
  'Will SpaceX land on Mars by 2027?',
  'Will inflation drop below 2%?',
  'Will crypto regulation pass in US?',
];

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

function getAuthToken() {
  const payload = JSON.stringify({ wallet: `${DEV_WALLET}-${__VU}` });
  const res = http.post(`${BASE_URL}/auth/dev-login`, payload, {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'auth_dev_login' },
  });

  authDuration.add(res.timings.duration);

  if (res.status === 200) {
    try {
      const body = JSON.parse(res.body);
      return body.token;
    } catch (e) {
      console.error(`Failed to parse auth response: ${e}`);
    }
  }
  console.error(`Auth failed: ${res.status} - ${res.body}`);
  return null;
}

function getHeaders(token) {
  const headers = { 'Content-Type': 'application/json' };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  return headers;
}

function randomChoice(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

function randomInt(min, max) {
  return Math.floor(Math.random() * (max - min + 1)) + min;
}

// ============================================================================
// API FUNCTIONS
// ============================================================================

function createMarket(token) {
  const question = `${randomChoice(marketQuestions)} (VU:${__VU}, Iter:${__ITER})`;
  const payload = JSON.stringify({
    question: question,
    description: `Load test market created at ${new Date().toISOString()}`,
    options: ['Yes', 'No'],
    end_time: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(), // 7 days from now
  });

  const res = http.post(`${BASE_URL}/api/markets`, payload, {
    headers: getHeaders(token),
    tags: { name: 'create_market' },
  });

  createMarketDuration.add(res.timings.duration);

  const success = check(res, {
    'create market status 201': (r) => r.status === 201,
    'create market has market data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.market && body.market.id;
      } catch (e) {
        return false;
      }
    },
  });

  if (!success) {
    errorRate.add(1);
    failedRequests.add(1);
    console.error(`Create market failed: ${res.status} - ${res.body}`);
    return null;
  }

  successfulRequests.add(1);
  errorRate.add(0);

  try {
    return JSON.parse(res.body).market;
  } catch (e) {
    return null;
  }
}

function listMarkets(token, page = 1, pageSize = 10) {
  const res = http.get(`${BASE_URL}/api/markets?page=${page}&page_size=${pageSize}`, {
    headers: getHeaders(token),
    tags: { name: 'list_markets' },
  });

  listMarketsDuration.add(res.timings.duration);

  const success = check(res, {
    'list markets status 200': (r) => r.status === 200,
    'list markets has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.markets !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  if (!success) {
    errorRate.add(1);
    failedRequests.add(1);
    return null;
  }

  successfulRequests.add(1);
  errorRate.add(0);

  try {
    return JSON.parse(res.body);
  } catch (e) {
    return null;
  }
}

function getMarket(token, marketId) {
  const res = http.get(`${BASE_URL}/api/markets/${marketId}`, {
    headers: getHeaders(token),
    tags: { name: 'get_market' },
  });

  getMarketDuration.add(res.timings.duration);

  const success = check(res, {
    'get market status 200': (r) => r.status === 200,
    'get market has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.market && body.market.id;
      } catch (e) {
        return false;
      }
    },
  });

  if (!success) {
    errorRate.add(1);
    failedRequests.add(1);
    return null;
  }

  successfulRequests.add(1);
  errorRate.add(0);

  try {
    return JSON.parse(res.body).market;
  } catch (e) {
    return null;
  }
}

function getMarketOptions(token, marketId) {
  const res = http.get(`${BASE_URL}/api/markets/${marketId}/options`, {
    headers: getHeaders(token),
    tags: { name: 'get_market_options' },
  });

  const success = check(res, {
    'get market options status 200': (r) => r.status === 200,
  });

  if (!success) {
    errorRate.add(1);
    failedRequests.add(1);
    return null;
  }

  successfulRequests.add(1);
  errorRate.add(0);

  try {
    return JSON.parse(res.body).options;
  } catch (e) {
    return null;
  }
}

function createWallet(token, userId) {
  const payload = JSON.stringify({
    user_id: userId,
    currency: 'USD',
  });

  const res = http.post(`${BASE_URL}/api/wallets`, payload, {
    headers: getHeaders(token),
    tags: { name: 'create_wallet' },
  });

  createWalletDuration.add(res.timings.duration);

  const success = check(res, {
    'create wallet status 200/201': (r) => r.status === 200 || r.status === 201,
  });

  if (!success) {
    errorRate.add(1);
    failedRequests.add(1);
    return null;
  }

  successfulRequests.add(1);
  errorRate.add(0);

  try {
    const body = JSON.parse(res.body);
    return body.account || body;
  } catch (e) {
    return null;
  }
}

function getWalletByUserId(token, userId) {
  const res = http.get(`${BASE_URL}/api/wallets/user/${userId}`, {
    headers: getHeaders(token),
    tags: { name: 'get_wallet_by_user' },
  });

  const success = check(res, {
    'get wallet by user status 200': (r) => r.status === 200,
  });

  if (!success) {
    errorRate.add(1);
    failedRequests.add(1);
    return null;
  }

  successfulRequests.add(1);
  errorRate.add(0);

  try {
    return JSON.parse(res.body).account;
  } catch (e) {
    return null;
  }
}

function deposit(token, walletId, amount) {
  const payload = JSON.stringify({
    amount: amount,
    reference_id: `deposit-${__VU}-${__ITER}-${Date.now()}`,
  });

  const res = http.post(`${BASE_URL}/api/wallets/${walletId}/deposit`, payload, {
    headers: getHeaders(token),
    tags: { name: 'deposit' },
  });

  depositDuration.add(res.timings.duration);

  const success = check(res, {
    'deposit status 200': (r) => r.status === 200,
  });

  if (!success) {
    errorRate.add(1);
    failedRequests.add(1);
    return false;
  }

  successfulRequests.add(1);
  errorRate.add(0);
  return true;
}

function createTransaction(token, userId, marketId, optionId, type, shares, pricePerShare) {
  const payload = JSON.stringify({
    user_id: userId,
    market_id: marketId,
    option_id: optionId,
    transaction_type: type, // "BUY" or "SELL"
    number_of_shares: shares,
    price_per_share: pricePerShare,
  });

  const res = http.post(`${BASE_URL}/api/transactions`, payload, {
    headers: getHeaders(token),
    tags: { name: `transaction_${type.toLowerCase()}` },
  });

  if (type === 'BUY') {
    buyTransactionDuration.add(res.timings.duration);
  } else {
    sellTransactionDuration.add(res.timings.duration);
  }

  const success = check(res, {
    [`${type} transaction status 201`]: (r) => r.status === 201,
    [`${type} transaction has data`]: (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.transaction !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  if (!success) {
    errorRate.add(1);
    failedRequests.add(1);
    console.error(`${type} transaction failed: ${res.status} - ${res.body}`);
    return null;
  }

  successfulRequests.add(1);
  errorRate.add(0);

  try {
    return JSON.parse(res.body);
  } catch (e) {
    return null;
  }
}

function getTransactions(token, userId = null, marketId = null, page = 1, pageSize = 20) {
  let url = `${BASE_URL}/api/transactions?page=${page}&page_size=${pageSize}`;
  if (userId) url += `&user_id=${userId}`;
  if (marketId) url += `&market_id=${marketId}`;

  const res = http.get(url, {
    headers: getHeaders(token),
    tags: { name: 'get_transactions' },
  });

  getTransactionsDuration.add(res.timings.duration);

  const success = check(res, {
    'get transactions status 200': (r) => r.status === 200,
    'get transactions has data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.transactions !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  if (!success) {
    errorRate.add(1);
    failedRequests.add(1);
    return null;
  }

  successfulRequests.add(1);
  errorRate.add(0);

  try {
    return JSON.parse(res.body);
  } catch (e) {
    return null;
  }
}

function healthCheck() {
  const res = http.get(`${BASE_URL}/health`, {
    tags: { name: 'health_check' },
  });

  return check(res, {
    'health check status 200': (r) => r.status === 200,
  });
}

// ============================================================================
// TEST SCENARIOS
// ============================================================================

// Setup function - runs once before tests
export function setup() {
  console.log(`Running load tests against: ${BASE_URL}`);

  // Health check
  if (!healthCheck()) {
    console.error('Health check failed! Ensure services are running.');
    return { ready: false };
  }

  // Get a token for setup
  const token = getAuthToken();
  if (!token) {
    console.error('Could not obtain auth token for setup');
    return { ready: false, token: null };
  }

  console.log('Setup complete. Starting tests...');
  return { ready: true, token: token };
}

// Smoke Test - Quick validation of all endpoints
export function smokeTest(data) {
  if (!data.ready) {
    fail('Setup failed');
    return;
  }

  const token = getAuthToken();
  if (!token) {
    fail('Could not get auth token');
    return;
  }

  const userId = `smoke-user-${__VU}`;

  group('Smoke Test - Full Flow', function () {
    // 1. List existing markets
    group('List Markets', function () {
      listMarkets(token);
    });
    sleep(0.5);

    // 2. Create a new market
    let market = null;
    group('Create Market', function () {
      market = createMarket(token);
    });
    sleep(0.5);

    if (market && market.id) {
      // 3. Get market details
      group('Get Market', function () {
        getMarket(token, market.id);
      });
      sleep(0.5);

      // 4. Get market options
      let options = null;
      group('Get Market Options', function () {
        options = getMarketOptions(token, market.id);
      });
      sleep(0.5);

      // 5. Create wallet and deposit funds
      let wallet = null;
      group('Wallet Operations', function () {
        wallet = createWallet(token, userId);
        if (wallet && wallet.id) {
          deposit(token, wallet.id, 1000);
        }
      });
      sleep(0.5);

      // 6. Buy shares
      if (options && options.length > 0 && wallet) {
        const optionId = options[0].id;
        group('Buy Transaction', function () {
          createTransaction(token, userId, market.id, optionId, 'BUY', 10, 0.5);
        });
        sleep(0.5);

        // 7. Sell shares
        group('Sell Transaction', function () {
          createTransaction(token, userId, market.id, optionId, 'SELL', 5, 0.6);
        });
        sleep(0.5);
      }

      // 8. Check transactions
      group('Get Transactions', function () {
        getTransactions(token, userId, market.id);
      });
    }
  });

  sleep(1);
}

// Load Test - Normal traffic simulation
export function loadTest(data) {
  if (!data.ready) {
    fail('Setup failed');
    return;
  }

  const token = getAuthToken();
  if (!token) {
    fail('Could not get auth token');
    return;
  }

  const userId = `load-user-${__VU}-${__ITER}`;
  const action = randomInt(1, 10);

  // Weighted random actions to simulate realistic traffic
  if (action <= 4) {
    // 40% - List and browse markets
    group('Browse Markets', function () {
      const result = listMarkets(token, randomInt(1, 3));
      if (result && result.markets && result.markets.length > 0) {
        const market = randomChoice(result.markets);
        sleep(0.3);
        getMarket(token, market.id);
        sleep(0.3);
        getMarketOptions(token, market.id);
      }
    });
  } else if (action <= 6) {
    // 20% - Create new market
    group('Create Market', function () {
      createMarket(token);
    });
  } else if (action <= 9) {
    // 30% - Trading (Buy/Sell)
    group('Trading', function () {
      const result = listMarkets(token);
      if (result && result.markets && result.markets.length > 0) {
        const market = randomChoice(result.markets);
        const options = getMarketOptions(token, market.id);

        if (options && options.length > 0) {
          // Ensure wallet exists with funds
          let wallet = getWalletByUserId(token, userId);
          if (!wallet || !wallet.id) {
            wallet = createWallet(token, userId);
            if (wallet && wallet.id) {
              deposit(token, wallet.id, 1000);
            }
          }

          if (wallet && wallet.id) {
            const option = randomChoice(options);
            const isBuy = Math.random() > 0.3; // 70% buy, 30% sell
            const shares = randomInt(1, 20);
            const price = Math.random() * 0.5 + 0.25; // 0.25 - 0.75

            if (isBuy) {
              createTransaction(token, userId, market.id, option.id, 'BUY', shares, price);
            } else {
              createTransaction(token, userId, market.id, option.id, 'SELL', shares, price);
            }
          }
        }
      }
    });
  } else {
    // 10% - View transactions
    group('View Transactions', function () {
      getTransactions(token, null, null, 1, 20);
    });
  }

  sleep(randomInt(1, 3));
}

// Stress Test - High traffic with concurrent operations
export function stressTest(data) {
  if (!data.ready) {
    fail('Setup failed');
    return;
  }

  const token = getAuthToken();
  if (!token) {
    fail('Could not get auth token');
    return;
  }

  const userId = `stress-user-${__VU}-${__ITER}`;

  // High-frequency operations
  group('Stress Operations', function () {
    // Rapid market listing
    listMarkets(token, 1, 20);
    sleep(0.1);

    // Create market under load
    const market = createMarket(token);
    sleep(0.1);

    if (market && market.id) {
      // Rapid options fetch
      const options = getMarketOptions(token, market.id);
      sleep(0.1);

      if (options && options.length > 0) {
        // Quick wallet setup
        let wallet = createWallet(token, userId);
        if (wallet && wallet.id) {
          deposit(token, wallet.id, 5000);
          sleep(0.1);

          // Rapid trading
          const option = options[0];
          for (let i = 0; i < 3; i++) {
            createTransaction(token, userId, market.id, option.id, 'BUY', randomInt(1, 10), 0.5);
            sleep(0.05);
          }
        }
      }
    }

    // Concurrent transaction queries
    getTransactions(token, userId);
  });

  sleep(0.5);
}

// Teardown function - runs once after all tests
export function teardown(data) {
  console.log('Load tests completed');
  console.log(`Total successful requests: ${successfulRequests.name}`);
  console.log(`Total failed requests: ${failedRequests.name}`);
}
