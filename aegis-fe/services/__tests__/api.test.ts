import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  fetchMarkets,
  fetchMarketById,
  createMarket,
  updateMarket,
  fetchOptionsByMarketId,
  createWallet,
  getWallet,
  deposit,
  withdraw,
  createSettlement,
  getSettlement,
  completeSettlement,
  checkHealth,
} from '@/services/api';
import { Market, WalletAccount, WalletTransaction, Settlement } from '@/types';

// Mock fetch globally
global.fetch = vi.fn();

describe('API Service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset process.env for each test
    process.env.NEXT_PUBLIC_API_URL = 'http://localhost:8080';
  });

  describe('Market APIs', () => {
    it('should fetch markets successfully', async () => {
      const mockMarkets: Market[] = [
        {
          id: '1',
          question: 'Will Bitcoin reach $100k?',
          description: 'Predict if BTC will hit $100k by end of 2024',
          category: 'Crypto',
          status: 'Active',
        },
      ];

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ markets: mockMarkets }),
      } as Response);

      const result = await fetchMarkets();
      expect(result).toEqual(mockMarkets);
      expect(fetch).toHaveBeenCalledWith('http://localhost:8080/api/markets', {
        headers: { 'Content-Type': 'application/json' },
      });
    });

    it('should handle fetch markets error', async () => {
      vi.mocked(fetch).mockRejectedValueOnce(new Error('Network error'));

      await expect(fetchMarkets()).rejects.toThrow('Network error');
    });

    it('should create market successfully', async () => {
      const newMarket = {
        question: 'Will it rain tomorrow?',
        description: 'Weather prediction',
        category: 'Weather',
        status: 'Active',
      };

      const createdMarket: Market = {
        id: '2',
        ...newMarket,
      };

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ market: createdMarket }),
      } as Response);

      const result = await createMarket(newMarket);
      expect(result).toEqual(createdMarket);
      expect(fetch).toHaveBeenCalledWith('http://localhost:8080/api/markets', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newMarket),
      });
    });
  });

  describe('Wallet APIs', () => {
    it('should create wallet successfully', async () => {
      const mockWallet: WalletAccount = {
        id: 'wallet-1',
        user_id: 'user-123',
        address: '0x123...',
        currency: 'USD',
        total_balance: 1000,
        available_balance: 1000,
        status: 'Active',
      };

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ account: mockWallet }),
      } as Response);

      const result = await createWallet('user-123', 'USD');
      expect(result).toEqual(mockWallet);
    });

    it('should deposit funds successfully', async () => {
      const mockTransaction: WalletTransaction = {
        id: 'tx-1',
        wallet_id: 'wallet-1',
        type: 'deposit',
        amount: 100,
        status: 'completed',
      };

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ transaction: mockTransaction }),
      } as Response);

      const result = await deposit('wallet-1', 100, 'ref-123');
      expect(result).toEqual(mockTransaction);
    });

    it('should withdraw funds successfully', async () => {
      const mockTransaction: WalletTransaction = {
        id: 'tx-2',
        wallet_id: 'wallet-1',
        type: 'withdrawal',
        amount: 50,
        status: 'completed',
      };

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ transaction: mockTransaction }),
      } as Response);

      const result = await withdraw('wallet-1', 50, 'ref-456');
      expect(result).toEqual(mockTransaction);
    });
  });

  describe('Settlement APIs', () => {
    it('should create settlement successfully', async () => {
      const mockSettlement: Settlement = {
        id: 'settlement-1',
        market_id: 'market-1',
        winning_option_id: 'option-1',
        total_pool: 10000,
        winning_pool: 6000,
        status: 'Pending',
      };

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ settlement: mockSettlement }),
      } as Response);

      const result = await createSettlement('market-1', 'option-1');
      expect(result).toEqual(mockSettlement);
    });

    it('should complete settlement successfully', async () => {
      const completedSettlement: Settlement = {
        id: 'settlement-1',
        market_id: 'market-1',
        winning_option_id: 'option-1',
        total_pool: 10000,
        winning_pool: 6000,
        status: 'Completed',
        settled_at: new Date().toISOString(),
      };

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ settlement: completedSettlement }),
      } as Response);

      const result = await completeSettlement('settlement-1');
      expect(result).toEqual(completedSettlement);
    });
  });

  describe('Health Check', () => {
    it('should return true for healthy backend', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
      } as Response);

      const result = await checkHealth();
      expect(result).toBe(true);
    });

    it('should return false for unhealthy backend', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
      } as Response);

      const result = await checkHealth();
      expect(result).toBe(false);
    });
  });

  describe('Retry Logic', () => {
    it('should retry failed requests', async () => {
      const mockMarket: Market = {
        id: '1',
        question: 'Test question',
        description: 'Test description',
        category: 'Test',
        status: 'Active',
      };

      // First call fails, second succeeds
      vi.mocked(fetch)
        .mockRejectedValueOnce(new Error('Network error'))
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ market: mockMarket }),
        } as Response);

      const result = await fetchMarketById('1');
      expect(result).toEqual(mockMarket);
      expect(fetch).toHaveBeenCalledTimes(2);
    });

    it('should handle 202 Accepted responses (circuit breaker fallback)', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        status: 202,
        json: async () => ({ status: 'accepted', message: 'Request queued' }),
      } as Response);

      const result = await fetchMarkets();
      expect(result).toEqual([]); // Empty array for 202 responses
    });
  });

  describe('Error Handling', () => {
    it('should throw APIError for HTTP errors', async () => {
      vi.mocked(fetch).mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
      } as Response);

      await expect(fetchMarketById('invalid-id')).rejects.toThrow('HTTP 404: Not Found');
    });

    it('should handle network failures gracefully', async () => {
      vi.mocked(fetch).mockRejectedValue(new Error('Network failure'));

      await expect(fetchMarkets()).rejects.toThrow('Network failure');
    });
  });
});