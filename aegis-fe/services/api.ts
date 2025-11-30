const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

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

export interface Transaction {
  id: string;
  user_id: string;
  market_id: string;
  option_id: string;
  transaction_type: string;
  price: number;
  created_at: string;
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

// Error handling and retry logic
class APIError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = 'APIError';
  }
}

import { getStoredToken } from "@/services/auth";

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
        const err = new APIError(response.status, `HTTP ${response.status}: ${response.statusText}`);
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
export async function createWallet(userId: string, currency: string = 'USD'): Promise<WalletAccount> {
  try {
    const response = await fetchWithRetry(`${API_BASE_URL}/api/wallets`, {
      method: 'POST',
      body: JSON.stringify({ user_id: userId, currency }),
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
export async function fetchTransactions(): Promise<Transaction[]> {
  console.warn('fetchTransactions is deprecated. Use wallet transactions instead.');
  return [];
}

export async function createTransaction(
  transaction: Omit<Transaction, "id">
): Promise<Transaction> {
  console.warn('createTransaction is deprecated. Use wallet operations instead.');
  return { ...transaction, id: `tx${Date.now()}` };
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
