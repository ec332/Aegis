import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { create } from 'zustand';
import { useAppStore } from '@/store/appStore';
import * as api from '@/services/api';

// Mock the API module
vi.mock('@/services/api', () => ({
  fetchMarkets: vi.fn(),
  fetchMarketById: vi.fn(),
  createMarket: vi.fn(),
  updateMarket: vi.fn(),
  fetchOptionsByMarketId: vi.fn(),
  createWallet: vi.fn(),
  getWallet: vi.fn(),
  deposit: vi.fn(),
  withdraw: vi.fn(),
  createSettlement: vi.fn(),
  getSettlement: vi.fn(),
  completeSettlement: vi.fn(),
  checkHealth: vi.fn(),
}));

describe('App Store Integration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    // Reset store state after each test
    useAppStore.setState({
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
    });
  });

  describe('Market Operations', () => {
    it('should load markets successfully', async () => {
      const mockMarkets = [
        {
          id: '1',
          question: 'Will Bitcoin reach $100k?',
          description: 'Predict if BTC will hit $100k by end of 2024',
          category: 'Crypto',
          status: 'Active',
        },
        {
          id: '2',
          question: 'Will Ethereum outperform Bitcoin?',
          description: 'Predict ETH performance vs BTC in Q4',
          category: 'Crypto',
          status: 'Active',
        },
      ];

      vi.mocked(api.fetchMarkets).mockResolvedValueOnce(mockMarkets);

      const { result } = renderHook(() => useAppStore());

      await act(async () => {
        await result.current.loadMarkets();
      });

      expect(result.current.markets).toEqual(mockMarkets);
      expect(result.current.isLoadingMarkets).toBe(false);
      expect(result.current.error).toBeNull();
    });

    it('should handle market loading errors', async () => {
      const error = new Error('Network error');
      vi.mocked(api.fetchMarkets).mockRejectedValueOnce(error);

      const { result } = renderHook(() => useAppStore());

      await act(async () => {
        await result.current.loadMarkets();
      });

      expect(result.current.markets).toEqual([]);
      expect(result.current.isLoadingMarkets).toBe(false);
      expect(result.current.error).toEqual(error);
    });

    it('should create a new market', async () => {
      const newMarketData = {
        question: 'Will it rain tomorrow?',
        description: 'Weather prediction for NYC',
        category: 'Weather',
        status: 'Active',
      };

      const createdMarket = {
        id: '3',
        ...newMarketData,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      };

      vi.mocked(api.createMarket).mockResolvedValueOnce(createdMarket);

      const { result } = renderHook(() => useAppStore());

      await act(async () => {
        const market = await result.current.createMarket(newMarketData);
        expect(market).toEqual(createdMarket);
      });

      expect(result.current.markets).toContainEqual(createdMarket);
      expect(result.current.error).toBeNull();
    });

    it('should load options for a market', async () => {
      const mockOptions = [
        {
          id: 'opt1',
          market_id: '1',
          option_text: 'Yes',
          current_price: 0.75,
          volume: 1000,
        },
        {
          id: 'opt2',
          market_id: '1',
          option_text: 'No',
          current_price: 0.25,
          volume: 500,
        },
      ];

      vi.mocked(api.fetchOptionsByMarketId).mockResolvedValueOnce(mockOptions);

      const { result } = renderHook(() => useAppStore());

      await act(async () => {
        await result.current.loadOptionsForMarket('1');
      });

      expect(result.current.marketOptions['1']).toEqual(mockOptions);
      expect(result.current.isLoadingOptions).toBe(false);
    });
  });

  describe('Wallet Operations', () => {
    it('should create wallet successfully', async () => {
      const mockWallet = {
        id: 'wallet-1',
        user_id: 'user-123',
        address: '0x1234567890abcdef',
        currency: 'USD',
        total_balance: 0,
        available_balance: 0,
        status: 'Active',
      };

      vi.mocked(api.createWallet).mockResolvedValueOnce(mockWallet);

      const { result } = renderHook(() => useAppStore());

      await act(async () => {
        const wallet = await result.current.createWallet('user-123', 'USD');
        expect(wallet).toEqual(mockWallet);
      });

      expect(result.current.walletAccounts).toContainEqual(mockWallet);
      expect(result.current.error).toBeNull();
    });

    it('should deposit funds to wallet', async () => {
      const mockTransaction = {
        id: 'tx-1',
        wallet_id: 'wallet-1',
        type: 'deposit',
        amount: 100,
        status: 'completed',
      };

      vi.mocked(api.deposit).mockResolvedValueOnce(mockTransaction);

      const { result } = renderHook(() => useAppStore());

      await act(async () => {
        const transaction = await result.current.depositFunds('wallet-1', 100, 'deposit-001');
        expect(transaction).toEqual(mockTransaction);
      });

      expect(result.current.walletTransactions).toContainEqual(mockTransaction);
    });

    it('should withdraw funds from wallet', async () => {
      const mockTransaction = {
        id: 'tx-2',
        wallet_id: 'wallet-1',
        type: 'withdrawal',
        amount: 50,
        status: 'completed',
      };

      vi.mocked(api.withdraw).mockResolvedValueOnce(mockTransaction);

      const { result } = renderHook(() => useAppStore());

      await act(async () => {
        const transaction = await result.current.withdrawFunds('wallet-1', 50, 'withdraw-001');
        expect(transaction).toEqual(mockTransaction);
      });

      expect(result.current.walletTransactions).toContainEqual(mockTransaction);
    });
  });

  describe('Settlement Operations', () => {
    it('should create settlement successfully', async () => {
      const mockSettlement = {
        id: 'settlement-1',
        market_id: 'market-1',
        winning_option_id: 'option-1',
        total_pool: 10000,
        winning_pool: 6000,
        status: 'Pending',
      };

      vi.mocked(api.createSettlement).mockResolvedValueOnce(mockSettlement);

      const { result } = renderHook(() => useAppStore());

      await act(async () => {
        const settlement = await result.current.createSettlement('market-1', 'option-1');
        expect(settlement).toEqual(mockSettlement);
      });

      expect(result.current.settlements).toContainEqual(mockSettlement);
    });

    it('should complete settlement successfully', async () => {
      const completedSettlement = {
        id: 'settlement-1',
        market_id: 'market-1',
        winning_option_id: 'option-1',
        total_pool: 10000,
        winning_pool: 6000,
        status: 'Completed',
        settled_at: new Date().toISOString(),
      };

      vi.mocked(api.completeSettlement).mockResolvedValueOnce(completedSettlement);

      const { result } = renderHook(() => useAppStore());

      await act(async () => {
        const settlement = await result.current.completeSettlement('settlement-1');
        expect(settlement).toEqual(completedSettlement);
      });

      expect(result.current.settlements).toContainEqual(completedSettlement);
    });
  });

  describe('Error Handling', () => {
    it('should handle backend health check failure', async () => {
      vi.mocked(api.checkHealth).mockResolvedValueOnce(false);

      const { result } = renderHook(() => useAppStore());

      await act(async () => {
        await result.current.checkBackendHealth();
      });

      expect(result.current.isBackendHealthy).toBe(false);
    });

    it('should clear errors', async () => {
      const error = new Error('Test error');
      
      const { result } = renderHook(() => useAppStore());

      // Set an error first
      act(() => {
        result.current.error = error;
      });

      expect(result.current.error).toEqual(error);

      // Clear the error
      act(() => {
        result.current.clearError();
      });

      expect(result.current.error).toBeNull();
    });
  });

  describe('Initialization', () => {
    it('should initialize app successfully', async () => {
      const mockMarkets = [
        {
          id: '1',
          question: 'Market 1',
          description: 'Description 1',
          category: 'Category 1',
          status: 'Active',
        },
      ];

      const mockOptions = [
        {
          id: 'opt1',
          market_id: '1',
          option_text: 'Yes',
          current_price: 0.5,
          volume: 100,
        },
      ];

      vi.mocked(api.checkHealth).mockResolvedValueOnce(true);
      vi.mocked(api.fetchMarkets).mockResolvedValueOnce(mockMarkets);
      vi.mocked(api.fetchOptionsByMarketId).mockResolvedValueOnce(mockOptions);

      const { result } = renderHook(() => useAppStore());

      await act(async () => {
        await result.current.initializeApp();
      });

      expect(result.current.isBackendHealthy).toBe(true);
      expect(result.current.markets).toEqual(mockMarkets);
      expect(result.current.marketOptions['1']).toEqual(mockOptions);
      expect(result.current.isLoadingMarkets).toBe(false);
    });

    it('should handle initialization errors', async () => {
      const error = new Error('Initialization failed');
      
      vi.mocked(api.checkHealth).mockRejectedValueOnce(error);

      const { result } = renderHook(() => useAppStore());

      await act(async () => {
        await result.current.initializeApp();
      });

      expect(result.current.error).toEqual(error);
      expect(result.current.isLoadingMarkets).toBe(false);
    });
  });
});