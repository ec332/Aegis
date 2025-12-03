const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || '';

// API Response Types based on backend proto definitions
export interface Market {
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

export interface Option {
  id: string;
  market_id: string;
  option_text: string;
  current_price: number;
  volume: number;
  created_at?: string;
  updated_at?: string;
}

export type TransactionType = "BUY" | "SELL" | (string & {});

export interface Transaction {
  id: string;
  user_id: string;
  market_id: string;
  option_id: string;
  transaction_type: TransactionType;
  price: number;
  created_at: string;
  number_of_shares?: number;
  price_per_share?: number;
}

export interface CreateTransactionInput {
  user_id: string;
  market_id: string;
  option_id: string;
  transaction_type: TransactionType;
  number_of_shares: number;
  price_per_share: number;
}

export interface WalletAccount {
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

export interface WalletTransaction {
  id: string;
  wallet_id: string;
  market_id?: string;
  type: string;
  amount: number;
  status: string;
  reference_id?: string;
  metadata?: string;
  created_at?: string;
  updated_at?: string;
}

export interface Settlement {
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

export interface User {
  id: string;
  wallet_address: string;
  balance: number;
  role: string;
  nonce?: string;
  created_at?: string;
  updated_at?: string;
  last_login?: string | null;
}

// Error handling and retry logic
export class APIError extends Error {
  constructor(public status: number, message: string, public details?: any) {
    super(message);
    this.name = 'APIError';
  }
}

import { getStoredToken } from "@/services/auth";

async function buildApiError(response: Response): Promise<APIError> {
  const defaultMessage = `HTTP ${response.status}: ${response.statusText}`;
  let message = defaultMessage;
  let details: any;

  try {
    const contentType = response.headers.get('content-type') || '';
    if (contentType.includes('application/json')) {
      const body = await response.json();
      details = body;
      if (body && typeof body === 'object') {
        const bodyMessage =
          (typeof body.message === 'string' && body.message) ||
          (typeof body.error === 'string' && body.error) ||
          undefined;
        if (bodyMessage) {
          message = bodyMessage;
        }
      }
    } else {
      const text = await response.text();
      if (text) {
        details = text;
        message = text;
      }
    }
  } catch (parseError) {
    console.warn('Failed to parse error response:', parseError);
  }

  return new APIError(response.status, message, details);
}

async function fetchWithRetry(
  url: string,
  options: RequestInit = {},
  maxRetries = 3,
  retryDelay = 1000
): Promise<Response> {
  for (let attempt = 0; attempt < maxRetries; attempt++) {
    try {
      const token = typeof window !== 'undefined' ? getStoredToken() : null;
      const response = await fetch(url, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
          ...options.headers,
        },
      });

      // If we get a 202 Accepted, it means the request was queued for async processing
      if (response.status === 202) {
        console.log('Request queued for async processing');
        return response;
      }

      if (!response.ok) {
        // Do not retry on most 4xx client errors (except 429 Too Many Requests)
        const err = await buildApiError(response);
        if (response.status >= 400 && response.status < 500 && response.status !== 429) {
          throw err;
        }
        throw err;
      }

      return response;
    } catch (error) {
      // If client error (4xx except 429), do not retry
      if (error instanceof APIError && error.status >= 400 && error.status < 500 && error.status !== 429) {
        throw error;
      }
      if (attempt === maxRetries - 1) {
        throw error;
      }
      
      // Exponential backoff
      const delay = retryDelay * Math.pow(2, attempt);
      console.log(`Retry attempt ${attempt + 1} after ${delay}ms delay`);
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }
  
  throw new Error('Max retries exceeded');
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (response.status === 202) {
    // Return a default response for async processing
    return {} as T;
  }
  
  const data = await response.json();
  return data as T;
}

// Market APIs
export async function fetchMarkets(): Promise<Market[]> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/markets`);
    const data = await handleResponse<{ markets: Market[] }>(response);
    return data.markets || [];
  } catch (error) {
    console.error('Error fetching markets:', error);
    throw error;
  }
}

export async function fetchMarketById(id: string): Promise<Market | null> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/markets/${id}`);
    const data = await handleResponse<{ market: Market }>(response);
    return data.market || null;
  } catch (error) {
    console.error(`Error fetching market ${id}:`, error);
    throw error;
  }
}

export async function createMarket(market: Omit<Market, 'id' | 'created_at' | 'updated_at'>): Promise<Market> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/markets`, {
      method: 'POST',
      body: JSON.stringify(market),
    });
    const data = await handleResponse<{ market: Market }>(response);
    return data.market;
  } catch (error) {
    console.error('Error creating market:', error);
    throw error;
  }
}

export async function updateMarket(id: string, updates: Partial<Market>): Promise<Market> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/markets/${id}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
    const data = await handleResponse<{ market: Market }>(response);
    return data.market;
  } catch (error) {
    console.error(`Error updating market ${id}:`, error);
    throw error;
  }
}

// Options APIs
export async function fetchOptionsByMarketId(marketId: string): Promise<Option[]> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/markets/${marketId}/options`);
    const data = await handleResponse<{ options: Option[] }>(response);
    return data.options || [];
  } catch (error) {
    console.error(`Error fetching options for market ${marketId}:`, error);
    throw error;
  }
}

// Wallet APIs
export async function createWallet(userId: string): Promise<WalletAccount> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/wallets`, {
      method: 'POST',
      body: JSON.stringify({ user_id: userId }),
    });
    const data = await handleResponse<{ account: WalletAccount }>(response);
    return data.account;
  } catch (error) {
    console.error('Error creating wallet:', error);
    throw error;
  }
}

export async function getWallet(walletId: string): Promise<WalletAccount | null> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/wallets/${walletId}`);
    const data = await handleResponse<{ account: WalletAccount }>(response);
    return data.account || null;
  } catch (error) {
    console.error(`Error fetching wallet ${walletId}:`, error);
    throw error;
  }
}

export async function getWalletByUserId(userId: string): Promise<WalletAccount | null> {
  try {
    const token = typeof window !== 'undefined' ? getStoredToken() : null;
    const res = await fetch(`${API_BASE_URL}/api/wallets/user/${userId}`, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
    });
    if (res.status === 404) {
      return null;
    }
    if (!res.ok) {
      throw new APIError(res.status, `HTTP ${res.status}: ${res.statusText}`);
    }
    const data = await res.json() as { account: WalletAccount | null };
    return data.account || null;
  } catch (error) {
    console.error(`Error fetching wallet by user ${userId}:`, error);
    return null;
  }
}

export async function getUserByWallet(walletAddress: string): Promise<User | null> {
  try {
    const token = typeof window !== 'undefined' ? getStoredToken() : null;
    const res = await fetch(`${API_BASE_URL}/api/users/wallet/${encodeURIComponent(walletAddress)}`, {
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
    });
    if (res.status === 404) {
      return null;
    }
    if (!res.ok) {
      throw new APIError(res.status, `HTTP ${res.status}: ${res.statusText}`);
    }
    const data = await res.json() as { user: User | null };
    return data.user || null;
  } catch (error) {
    console.error(`Error fetching user by wallet ${walletAddress}:`, error);
    return null;
  }
}

export async function createUser(walletAddress: string, balance = 0, role = 'user'): Promise<User> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/users`, {
      method: 'POST',
      body: JSON.stringify({ wallet_address: walletAddress, balance, role }),
    });
    const data = await handleResponse<{ user: User }>(response);
    return data.user;
  } catch (error) {
    console.error('Error creating user:', error);
    throw error;
  }
}

export async function deposit(walletId: string, amount: number, referenceId?: string): Promise<WalletTransaction> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/wallets/${walletId}/deposit`, {
      method: 'POST',
      body: JSON.stringify({ account_id: walletId, amount, reference_id: referenceId }),
    });
    const data = await handleResponse<{ transaction: WalletTransaction }>(response);
    return data.transaction;
  } catch (error) {
    console.error(`Error depositing to wallet ${walletId}:`, error);
    throw error;
  }
}

export async function withdraw(walletId: string, amount: number, referenceId?: string): Promise<WalletTransaction> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/wallets/${walletId}/withdraw`, {
      method: 'POST',
      body: JSON.stringify({ account_id: walletId, amount, reference_id: referenceId }),
    });
    const data = await handleResponse<{ transaction: WalletTransaction }>(response);
    return data.transaction;
  } catch (error) {
    console.error(`Error withdrawing from wallet ${walletId}:`, error);
    throw error;
  }
}

export async function fetchWalletTransactions(walletId: string, limit = 50, offset = 0): Promise<{ transactions: WalletTransaction[]; total: number }> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/wallets/${walletId}/transactions?limit=${limit}&offset=${offset}`);
    const data = await handleResponse<{ transactions: WalletTransaction[]; total: number }>(response);
    return { transactions: data.transactions || [], total: data.total || 0 };
  } catch (error) {
    console.error(`Error fetching wallet transactions ${walletId}:`, error);
    throw error;
  }
}

// Settlement APIs
export async function createSettlement(marketId: string, winningOptionId: string): Promise<Settlement> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/settlements`, {
      method: 'POST',
      body: JSON.stringify({ market_id: marketId, winning_option_id: winningOptionId }),
    });
    const data = await handleResponse<{ settlement: Settlement }>(response);
    return data.settlement;
  } catch (error) {
    console.error('Error creating settlement:', error);
    throw error;
  }
}

export async function getSettlement(settlementId: string): Promise<Settlement | null> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/settlements/${settlementId}`);
    const data = await handleResponse<{ settlement: Settlement }>(response);
    return data.settlement || null;
  } catch (error) {
    console.error(`Error fetching settlement ${settlementId}:`, error);
    throw error;
  }
}

export async function completeSettlement(settlementId: string): Promise<Settlement> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/settlements/${settlementId}/complete`, {
      method: 'PUT',
      body: JSON.stringify({}),
    });
    const data = await handleResponse<{ settlement: Settlement }>(response);
    return data.settlement;
  } catch (error) {
    console.error(`Error completing settlement ${settlementId}:`, error);
    throw error;
  }
}

// Health check
export async function checkHealth(): Promise<boolean> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/health`);
    return response.ok;
  } catch (error) {
    console.error('Health check failed:', error);
    return false;
  }
}

// Legacy transaction APIs (for backward compatibility)
export async function fetchUserTransactions(userId: string): Promise<Transaction[]> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/transactions?user_id=${encodeURIComponent(userId)}`);
    const data = await handleResponse<{ transactions: Transaction[] }>(response);
    return data.transactions || [];
  } catch (error) {
    console.error('Error fetching user transactions:', error);
    throw error;
  }
}

export async function fetchUserTransactionsForMarket(userId: string, marketId: string, page = 1, pageSize = 200): Promise<Transaction[]> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/transactions?user_id=${encodeURIComponent(userId)}&market_id=${encodeURIComponent(marketId)}&page=${page}&page_size=${pageSize}`);
    const data = await handleResponse<{ transactions: Transaction[]; total?: number }>(response);
    return data.transactions || [];
  } catch (error) {
    console.error(`Error fetching transactions for user ${userId} and market ${marketId}:`, error);
    throw error;
  }
}

export async function createTransaction(
  transaction: CreateTransactionInput
): Promise<Transaction> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/transactions`, {
      method: 'POST',
      body: JSON.stringify(transaction),
    });
    const data = await handleResponse<{ transaction?: Transaction } & Partial<Transaction>>(response);
    if (data && 'transaction' in data && data.transaction) {
      return data.transaction;
    }
    return data as Transaction;
  } catch (error) {
    console.error('Error creating transaction:', error);
    throw error;
  }
}

export async function updateTransaction(
  id: string,
  updates: Partial<Transaction>
): Promise<Transaction | null> {
  console.warn('updateTransaction is deprecated.');
  return null;
}

export async function deleteTransaction(id: string): Promise<boolean> {
  console.warn('deleteTransaction is deprecated.');
  return false;
}
