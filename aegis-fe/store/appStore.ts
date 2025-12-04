import { create } from "zustand";
import {
  Market,
  Option,
  Transaction,
  WalletAccount,
  WalletTransaction,
  Settlement,
  APIError
} from "@/types";
import { DEFAULT_USER_ID } from "@/constants";
import {
  fetchMarkets,
  fetchMarketById,
  createMarket as apiCreateMarket,
  updateMarket as apiUpdateMarket,
  fetchOptionsByMarketId,
  createWallet as apiCreateWallet,
  getWallet as apiGetWallet,
  deposit as apiDeposit,
  withdraw as apiWithdraw,
  createSettlement as apiCreateSettlement,
  getSettlement as apiGetSettlement,
  completeSettlement as apiCompleteSettlement,
  checkHealth,
  fetchUserTransactions,
} from "@/services/api";
import { fetchWalletTransactions, getWalletByUserId } from "@/services/api";

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

  // Legacy transactions
  transactions: Transaction[];

  // Loading states
  isLoadingMarkets: boolean;
  isLoadingOptions: boolean;
  isLoadingTransactions: boolean;
  isLoadingWallet: boolean;
  isLoadingSettlements: boolean;

  // Error states
  error: APIError | null;
  isBackendHealthy: boolean;
  initialized: boolean;

  // Actions
  initializeApp: () => Promise<void>;
  loadMarkets: () => Promise<void>;
  loadMarketById: (marketId: string) => Promise<Market | null>;
  createMarket: (market: Omit<Market, 'id' | 'created_at' | 'updated_at'>) => Promise<Market>;
  updateMarket: (id: string, updates: Partial<Market>) => Promise<Market>;
  loadOptionsForMarket: (marketId: string) => Promise<void>;
  loadTransactions: () => Promise<void>;
  loadTransactionsForUser: (userId: string) => Promise<void>;

  // Wallet actions
  createWallet: (userId: string, currency?: string) => Promise<WalletAccount>;
  createWalletForUser: (userId: string) => Promise<WalletAccount>;
  loadWallet: (walletId: string) => Promise<WalletAccount | null>;
  loadCurrentUserWallet: (userId: string) => Promise<void>;
  depositFunds: (walletId: string, amount: number, referenceId?: string) => Promise<WalletTransaction>;
  withdrawFunds: (walletId: string, amount: number, referenceId?: string) => Promise<WalletTransaction>;
  loadWalletTransactions: (walletId: string) => Promise<void>;

  // Settlement actions
  createSettlement: (marketId: string, winningOptionId: string) => Promise<Settlement>;
  loadSettlement: (settlementId: string) => Promise<Settlement | null>;
  completeSettlement: (settlementId: string) => Promise<Settlement>;

  // Legacy actions
  addTransaction: (transaction: Omit<Transaction, "id">) => Promise<void>;
  removeTransaction: (transactionId: string) => Promise<void>;
  updateTransaction: (id: string, updates: Partial<Transaction>) => Promise<void>;

  // Utility actions
  clearError: () => void;
  checkBackendHealth: () => Promise<void>;
}

export const useAppStore = create<AppState>((set, get) => ({
  // Initial state
  markets: [],
  marketOptions: {},
  walletAccounts: [],
  walletTransactions: [],
  currentWallet: null,
  settlements: [],
  transactions: [],
  isLoadingMarkets: false,
  isLoadingOptions: false,
  isLoadingTransactions: false,
  isLoadingWallet: false,
  isLoadingSettlements: false,
  error: null,
  isBackendHealthy: false,
  initialized: false,

  // Actions
  initializeApp: async () => {
    if (get().initialized) return;
    set({
      isLoadingMarkets: true,
      isLoadingTransactions: true,
      isLoadingOptions: true,
      initialized: true,
    });

    try {
      // Check backend health first
      await get().checkBackendHealth();

      // Skip auto wallet creation during startup to avoid 500s when wallet service is not available.

      const [markets] = await Promise.all([
        fetchMarkets(),
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
        marketOptions: marketOptionsMap,
        isLoadingMarkets: false,
        isLoadingTransactions: false,
        isLoadingOptions: false,
      });
    } catch (error) {
      console.error("Error initializing app:", error);
      set({
        error: error as APIError,
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
      set({
        error: error as APIError,
        isLoadingMarkets: false
      });
    }
  },

  loadMarketById: async (marketId: string) => {
    try {
      const market = await fetchMarketById(marketId);
      if (market) {
        set((state) => ({
          markets: state.markets.map(m => m.id === marketId ? market : m)
        }));
      }
      return market;
    } catch (error) {
      console.error(`Error loading market ${marketId}:`, error);
      set({ error: error as APIError });
      return null;
    }
  },

  createMarket: async (market: Omit<Market, 'id' | 'created_at' | 'updated_at'>) => {
    try {
      const newMarket = await apiCreateMarket(market);
      set((state) => ({
        markets: [...state.markets, newMarket]
      }));
      return newMarket;
    } catch (error) {
      console.error("Error creating market:", error);
      set({ error: error as APIError });
      throw error;
    }
  },

  updateMarket: async (id: string, updates: Partial<Market>) => {
    try {
      const updatedMarket = await apiUpdateMarket(id, updates);
      set((state) => ({
        markets: state.markets.map(m => m.id === id ? updatedMarket : m)
      }));
      return updatedMarket;
    } catch (error) {
      console.error(`Error updating market ${id}:`, error);
      set({ error: error as APIError });
      throw error;
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
      console.error(`Error loading options for market ${marketId}:`, error);
      set({
        error: error as APIError,
        isLoadingOptions: false
      });
    }
  },

  loadTransactions: async () => {
    set({ isLoadingTransactions: true });
    try {
      // This is now legacy - wallet transactions should be used instead
      set({ isLoadingTransactions: false });
    } catch (error) {
      console.error("Error loading transactions:", error);
      set({
        error: error as APIError,
        isLoadingTransactions: false
      });
    }
  },

  loadTransactionsForUser: async (userId: string) => {
    set({ isLoadingTransactions: true });
    try {
      const txs = await fetchUserTransactions(userId);
      set({ transactions: txs, isLoadingTransactions: false });
    } catch (error) {
      console.error("Error loading transactions for user:", error);
      set({
        error: error as APIError,
        isLoadingTransactions: false
      });
    }
  },

  // Wallet actions
  createWallet: async (userId: string, currency: string = 'USD') => {
    try {
      const wallet = await apiCreateWallet(userId);
      set((state) => ({
        walletAccounts: [...state.walletAccounts, wallet]
      }));
      return wallet;
    } catch (error) {
      console.error("Error creating wallet:", error);
      set({ error: error as APIError });
      throw error;
    }
  },

  createWalletForUser: async (userId: string) => {
    try {
      const wallet = await apiCreateWallet(userId);
      set((state) => ({
        walletAccounts: [...state.walletAccounts, wallet],
        currentWallet: wallet,
      }));
      await get().loadWalletTransactions(wallet.id);
      return wallet;
    } catch (error) {
      console.error("Error creating wallet for user:", error);
      set({ error: error as APIError });
      throw error;
    }
  },

  loadWallet: async (walletId: string) => {
    try {
      const wallet = await apiGetWallet(walletId);
      if (wallet) {
        set((state) => ({
          walletAccounts: state.walletAccounts.map(w => w.id === walletId ? wallet : w),
          currentWallet: state.currentWallet?.id === walletId ? wallet : state.currentWallet
        }));
      }
      return wallet;
    } catch (error) {
      console.error(`Error loading wallet ${walletId}:`, error);
      set({ error: error as APIError });
      return null;
    }
  },

  loadCurrentUserWallet: async (userId: string) => {
    set({ isLoadingWallet: true });
    try {
      const existing = await getWalletByUserId(userId);
      if (existing) {
        set({ currentWallet: existing, isLoadingWallet: false });
        await get().loadWalletTransactions(existing.id);
      } else {
        set({ isLoadingWallet: false });
      }
    } catch (error) {
      console.error(`Error loading user wallet for ${userId}:`, error);
      set({
        error: error as APIError,
        isLoadingWallet: false
      });
    }
  },

  depositFunds: async (walletId: string, amount: number, referenceId?: string) => {
    try {
      const transaction = await apiDeposit(walletId, amount, referenceId);
      set((state) => ({
        walletTransactions: [...state.walletTransactions, transaction]
      }));
      // Reload wallet to get updated balance
      await get().loadWallet(walletId);
      return transaction;
    } catch (error) {
      console.error(`Error depositing to wallet ${walletId}:`, error);
      set({ error: error as APIError });
      throw error;
    }
  },

  withdrawFunds: async (walletId: string, amount: number, referenceId?: string) => {
    try {
      const transaction = await apiWithdraw(walletId, amount, referenceId);
      set((state) => ({
        walletTransactions: [...state.walletTransactions, transaction]
      }));
      // Reload wallet to get updated balance
      await get().loadWallet(walletId);
      return transaction;
    } catch (error) {
      console.error(`Error withdrawing from wallet ${walletId}:`, error);
      set({ error: error as APIError });
      throw error;
    }
  },

  loadWalletTransactions: async (walletId: string) => {
    try {
      const { transactions } = await fetchWalletTransactions(walletId);
      set({ walletTransactions: transactions });
    } catch (error) {
      console.error(`Error fetching wallet transactions for ${walletId}:`, error);
      if (error && typeof error === 'object' && (error as APIError).status === 401) {
        set({ walletTransactions: [] });
        return;
      }
      set({ error: error as APIError });
    }
  },

  // Settlement actions
  createSettlement: async (marketId: string, winningOptionId: string) => {
    try {
      const settlement = await apiCreateSettlement(marketId, winningOptionId);
      set((state) => ({
        settlements: [...state.settlements, settlement]
      }));
      return settlement;
    } catch (error) {
      console.error("Error creating settlement:", error);
      set({ error: error as APIError });
      throw error;
    }
  },

  loadSettlement: async (settlementId: string) => {
    try {
      const settlement = await apiGetSettlement(settlementId);
      if (settlement) {
        set((state) => ({
          settlements: state.settlements.map(s => s.id === settlementId ? settlement : s)
        }));
      }
      return settlement;
    } catch (error) {
      console.error(`Error loading settlement ${settlementId}:`, error);
      set({ error: error as APIError });
      return null;
    }
  },

  completeSettlement: async (settlementId: string) => {
    try {
      const settlement = await apiCompleteSettlement(settlementId);
      set((state) => ({
        settlements: state.settlements.map(s => s.id === settlementId ? settlement : s)
      }));
      if (settlement && settlement.market_id) {
        await get().loadMarketById(settlement.market_id);
      }
      return settlement;
    } catch (error) {
      console.error(`Error completing settlement ${settlementId}:`, error);
      set({ error: error as APIError });
      throw error;
    }
  },

  // Legacy actions
  addTransaction: async (transaction: Omit<Transaction, "id">) => {
    console.warn("addTransaction is deprecated. Use wallet operations instead.");
  },

  removeTransaction: async (transactionId: string) => {
    console.warn("removeTransaction is deprecated.");
  },

  updateTransaction: async (id: string, updates: Partial<Transaction>) => {
    console.warn("updateTransaction is deprecated.");
  },

  // Utility actions
  clearError: () => {
    set({ error: null });
  },

  checkBackendHealth: async () => {
    try {
      const isHealthy = await checkHealth();
      set({ isBackendHealthy: isHealthy });
    } catch (error) {
      console.error("Backend health check failed:", error);
      set({ isBackendHealthy: false });
    }
  },
}));
