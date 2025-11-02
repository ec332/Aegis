import { Market, Option, Transaction } from "@/types";

// Fake data
const markets: Market[] = [
  {
    id: "1",
    title: "Will Bitcoin reach $100k?",
    description: "Predict if BTC will hit $100k by end of 2024",
    status: "Active",
  },
  {
    id: "2",
    title: "Will Ethereum outperform Bitcoin?",
    description: "Predict ETH performance vs BTC in Q4",
    status: "Active",
  },
  {
    id: "3",
    title: "Will AI stocks rally?",
    description: "Predict the movement of AI-focused stocks",
    status: "Active",
  },
  {
    id: "5",
    title: "Tech IPO Q4 2024",
    description: "Will there be a major tech IPO?",
    status: "Active",
  },
  {
    id: "6",
    title: "Gold price forecast",
    description: "Will gold break $2500/oz?",
    status: "Active",
  },
];

const options: Option[] = [
  { id: "opt1", market_id: "1", title: "Yes" },
  { id: "opt2", market_id: "1", title: "No" },
  { id: "opt3", market_id: "2", title: "Yes" },
  { id: "opt4", market_id: "2", title: "No" },
  { id: "opt5", market_id: "3", title: "Rally" },
  { id: "opt6", market_id: "3", title: "Decline" },
  { id: "opt9", market_id: "5", title: "Yes" },
  { id: "opt10", market_id: "5", title: "No" },
  { id: "opt11", market_id: "6", title: "Yes" },
  { id: "opt12", market_id: "6", title: "No" },
];

let transactions: Transaction[] = [
  {
    id: "tx1",
    user_id: "user1",
    market_id: "1",
    option_id: "opt1",
    transaction_type: "buy",
    price: 45.50,
    created_at: "2024-11-01T14:30:00Z",
  },
  {
    id: "tx2",
    user_id: "user1",
    market_id: "2",
    option_id: "opt3",
    transaction_type: "sell",
    price: 32.75,
    created_at: "2024-11-01T12:15:00Z",
  },
  {
    id: "tx3",
    user_id: "user1",
    market_id: "3",
    option_id: "opt5",
    transaction_type: "buy",
    price: 67.25,
    created_at: "2024-10-31T09:45:00Z",
  },
];

// Markets API
export async function fetchMarkets(): Promise<Market[]> {
  return markets;
}

export async function fetchMarketById(id: string): Promise<Market | null> {
    return markets.find((m) => m.id === id) || null;
}

// Options API
export async function fetchOptionsByMarketId(marketId: string): Promise<Option[]> {
  return options.filter((opt) => opt.market_id === marketId);
}

export async function fetchOptionById(id: string): Promise<Option | null> {
  return options.find((opt) => opt.id === id) || null;
}

// Transactions API
export async function fetchTransactions(): Promise<Transaction[]> {
  return transactions;
}

export async function createTransaction(
  transaction: Omit<Transaction, "id">
): Promise<Transaction> {
  const newTransaction: Transaction = {
    ...transaction,
    id: `tx${Date.now()}`,
  };
  transactions.push(newTransaction);
  return newTransaction;
}

export async function updateTransaction(
  id: string,
  updates: Partial<Transaction>
): Promise<Transaction | null> {
  const index = transactions.findIndex((t) => t.id === id);
  if (index === -1) return null;
  transactions[index] = { ...transactions[index], ...updates };
  return transactions[index];
}

export async function deleteTransaction(id: string): Promise<boolean> {
  const index = transactions.findIndex((t) => t.id === id);
  if (index === -1) return false;
  transactions.splice(index, 1);
  return true;
}
