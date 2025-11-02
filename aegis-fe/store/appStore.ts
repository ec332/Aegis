import { create } from "zustand";
import { Market, Option, Transaction } from "@/types";
import {
  fetchMarkets,
  fetchOptionsByMarketId,
  fetchTransactions,
  createTransaction as apiCreateTransaction,
  updateTransaction as apiUpdateTransaction,
  deleteTransaction as apiDeleteTransaction,
} from "@/services/api";

interface AppState {
  // Markets
  markets: Market[];
  marketOptions: { [key: string]: Option[] };
  
  // Transactions
  transactions: Transaction[];
  
  // Loading states
  isLoadingMarkets: boolean;
  isLoadingOptions: boolean;
  isLoadingTransactions: boolean;
  
  // Actions
  initializeApp: () => Promise<void>;
  loadMarkets: () => Promise<void>;
  loadOptionsForMarket: (marketId: string) => Promise<void>;
  loadTransactions: () => Promise<void>;
  addTransaction: (transaction: Omit<Transaction, "id">) => Promise<void>;
  removeTransaction: (transactionId: string) => Promise<void>;
  updateTransaction: (id: string, updates: Partial<Transaction>) => Promise<void>;
}

export const useAppStore = create<AppState>((set) => ({
  // Initial state
  markets: [],
  marketOptions: {},
  transactions: [],
  isLoadingMarkets: false,
  isLoadingOptions: false,
  isLoadingTransactions: false,

  // Actions
  initializeApp: async () => {
    set({
      isLoadingMarkets: true,
      isLoadingTransactions: true,
      isLoadingOptions: true,
    });
    
    try {
      const [markets, transactions] = await Promise.all([
        fetchMarkets(),
        fetchTransactions(),
      ]);
      
      // Load options for all markets
      const marketOptionsMap: { [key: string]: Option[] } = {};
      const optionsPromises = markets.map(async (market) => {
        const options = await fetchOptionsByMarketId(market.id);
        marketOptionsMap[market.id] = options;
      });
      
      await Promise.all(optionsPromises);
      
      set({
        markets,
        transactions,
        marketOptions: marketOptionsMap,
        isLoadingMarkets: false,
        isLoadingTransactions: false,
        isLoadingOptions: false,
      });
    } catch (error) {
      console.error("Error initializing app:", error);
      set({
        isLoadingMarkets: false,
        isLoadingTransactions: false,
        isLoadingOptions: false,
      });
    }
  },

  loadMarkets: async () => {
    set({ isLoadingMarkets: true });
    try {
      const markets = await fetchMarkets();
      set({ markets, isLoadingMarkets: false });
    } catch (error) {
      console.error("Error loading markets:", error);
      set({ isLoadingMarkets: false });
    }
  },

  loadOptionsForMarket: async (marketId: string) => {
    set({ isLoadingOptions: true });
    try {
      const options = await fetchOptionsByMarketId(marketId);
      set((state) => ({
        marketOptions: {
          ...state.marketOptions,
          [marketId]: options,
        },
        isLoadingOptions: false,
      }));
    } catch (error) {
      console.error("Error loading options:", error);
      set({ isLoadingOptions: false });
    }
  },

  loadTransactions: async () => {
    set({ isLoadingTransactions: true });
    try {
      const transactions = await fetchTransactions();
      set({ transactions, isLoadingTransactions: false });
    } catch (error) {
      console.error("Error loading transactions:", error);
      set({ isLoadingTransactions: false });
    }
  },

  addTransaction: async (transaction: Omit<Transaction, "id">) => {
    try {
      const newTransaction = await apiCreateTransaction(transaction);
      set((state) => ({
        transactions: [...state.transactions, newTransaction],
      }));
    } catch (error) {
      console.error("Error adding transaction:", error);
    }
  },

  removeTransaction: async (transactionId: string) => {
    try {
      const success = await apiDeleteTransaction(transactionId);
      if (success) {
        set((state) => ({
          transactions: state.transactions.filter((t) => t.id !== transactionId),
        }));
      }
    } catch (error) {
      console.error("Error removing transaction:", error);
    }
  },

  updateTransaction: async (id: string, updates: Partial<Transaction>) => {
    try {
      const updated = await apiUpdateTransaction(id, updates);
      if (updated) {
        set((state) => ({
          transactions: state.transactions.map((t) =>
            t.id === id ? updated : t
          ),
        }));
      }
    } catch (error) {
      console.error("Error updating transaction:", error);
    }
  },
}));
