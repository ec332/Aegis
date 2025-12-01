// Market Type - matches backend proto definition
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

// Option Type - matches backend proto definition
export interface Option {
  id: string;
  market_id: string;
  option_text: string;
  current_price: number;
  volume: number;
  created_at?: string;
  updated_at?: string;
}

// Market with Options - frontend convenience type
export interface MarketWithOptions extends Market {
  options: Option[];
}

// Wallet Account Type - matches backend proto definition
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

// Wallet Transaction Type - matches backend proto definition
export interface WalletTransaction {
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

// Settlement Type - matches backend proto definition
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

// Legacy Transaction Type - kept for backward compatibility
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

// API Error Type
export interface APIError {
  status: number;
  message: string;
  details?: any;
}

// API Response Types
export interface APIResponse<T> {
  data?: T;
  error?: APIError;
  status: 'loading' | 'success' | 'error';
}

// User Type (if needed for future user management)
export interface User {
  id: string;
  wallet_address: string;
  balance: number;
  nonce: string;
  role: string;
  created_at?: string;
  updated_at?: string;
  last_login?: string | null;
}
