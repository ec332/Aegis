// Market Type
export interface Market {
  id: string; // UUID
  title: string;
  description: string;
  status: string;
}

// Option Type
export interface Option {
  id: string; // UUID
  market_id: string; // UUID FK
  title: string;
}

// Market with Options
export interface MarketWithOptions extends Market {
  options: Option[];
}

// Transaction Type
export interface Transaction {
  id: string; // UUID
  user_id: string; // UUID FK
  market_id: string; // UUID FK
  option_id: string; // UUID FK
  transaction_type: string; // "buy" | "sell"
  created_at: string; // ISO timestamp
}
