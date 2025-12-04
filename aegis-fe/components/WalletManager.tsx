'use client';

import { useState, useEffect } from 'react';
import { useAppStore } from '@/store/appStore';
import { getUserByWallet, createUser } from '@/services/api';
import { useAuthStore } from '@/store/authStore';
import { WalletAccount, WalletTransaction } from '@/types';

export default function WalletManager({ userId }: { userId: string }) {
  const { 
    currentWallet, 
    walletTransactions, 
    isLoadingWallet, 
    error,
    loadCurrentUserWallet, 
    createWalletForUser,
    depositFunds, 
    withdrawFunds 
  } = useAppStore();
  const { isAuthenticated, token, wallet } = useAuthStore();
  const [walletUserId, setWalletUserId] = useState<string>("");
  
  const [depositAmount, setDepositAmount] = useState('');
  const [withdrawAmount, setWithdrawAmount] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [creatingWallet, setCreatingWallet] = useState(false);

  useEffect(() => {
    let cancelled = false;
    async function ensureWalletUser() {
      if (!wallet) return;
      const existing = await getUserByWallet(wallet);
      if (cancelled) return;
      if (existing) {
        setWalletUserId(existing.id);
      } else {
        try {
          const created = await createUser(wallet);
          if (!cancelled) setWalletUserId(created.id);
        } catch {}
      }
    }
    ensureWalletUser();
    return () => { cancelled = true; };
  }, [wallet]);

  useEffect(() => {
    if (!isAuthenticated || !token) {
      return;
    }
    const uid = walletUserId || userId;
    if (uid && !currentWallet) {
      loadCurrentUserWallet(uid);
    }
  }, [isAuthenticated, token, walletUserId, userId, currentWallet, loadCurrentUserWallet]);

  const handleDeposit = async () => {
    if (!depositAmount || !currentWallet) return;
    
    setIsLoading(true);
    try {
      await depositFunds(currentWallet.id, parseFloat(depositAmount), `deposit_${Date.now()}`);
      setDepositAmount('');
    } catch (err) {
      console.error('Deposit failed:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleWithdraw = async () => {
    if (!withdrawAmount || !currentWallet) return;
    
    const amount = parseFloat(withdrawAmount);
    const availableBalance = currentWallet.available_balance ?? 0;
    if (amount > availableBalance) {
      alert('Insufficient funds');
      return;
    }
    
    setIsLoading(true);
    try {
      await withdrawFunds(currentWallet.id, amount, `withdraw_${Date.now()}`);
      setWithdrawAmount('');
    } catch (err) {
      console.error('Withdrawal failed:', err);
    } finally {
      setIsLoading(false);
    }
  };

  if (isLoadingWallet) {
    return (
      <div className="bg-white rounded-lg shadow-md p-6">
        <div className="animate-pulse">
          <div className="h-4 bg-gray-200 rounded w-1/4 mb-2"></div>
          <div className="h-8 bg-gray-200 rounded w-1/2 mb-4"></div>
          <div className="h-4 bg-gray-200 rounded w-3/4 mb-2"></div>
          <div className="h-4 bg-gray-200 rounded w-1/2"></div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-6">
        <div className="flex items-center">
          <svg className="w-6 h-6 text-red-600 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span className="text-red-800">Failed to load wallet</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Wallet Overview */}
      <div className="bg-white rounded-lg shadow-md p-6">
        <h2 className="text-2xl font-bold text-gray-900 mb-4">Wallet</h2>
        
        {currentWallet ? (
          <div className="space-y-4">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div className="bg-blue-50 p-4 rounded-lg">
                <p className="text-sm text-blue-600 font-medium">Total Balance</p>
                <p className="text-2xl font-bold text-blue-900">
                  ${(currentWallet.total_balance ?? 0).toFixed(2)}
                </p>
              </div>
              <div className="bg-green-50 p-4 rounded-lg">
                <p className="text-sm text-green-600 font-medium">Available</p>
                <p className="text-2xl font-bold text-green-900">
                  ${(currentWallet.available_balance ?? 0).toFixed(2)}
                </p>
              </div>
            </div>
            
            <div className="text-sm text-gray-600">
              <p><strong>Wallet ID:</strong> {currentWallet.id}</p>
              <p><strong>Address:</strong> {currentWallet.address}</p>
              <p><strong>Currency:</strong> {currentWallet.currency}</p>
              <p><strong>Status:</strong> 
                <span className={`ml-2 px-2 py-1 rounded-full text-xs font-medium ${
                  currentWallet.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                }`}>
                  {currentWallet.status}
                </span>
              </p>
            </div>
          </div>
        ) : (
          <div className="text-center py-8">
            <p className="text-gray-500 mb-4">No wallet found</p>
              <button
                onClick={async () => {
                  if (!isAuthenticated || !token) {
                    alert('Please sign in first');
                    return;
                  }
                  if (creatingWallet) return;
                  setCreatingWallet(true);
                  try {
                    let resolvedUserId = walletUserId || userId;
                    if (wallet && !walletUserId) {
                      const u = await getUserByWallet(wallet);
                      if (u) {
                        resolvedUserId = u.id;
                        setWalletUserId(u.id);
                      } else {
                        const created = await createUser(wallet);
                        resolvedUserId = created.id;
                        setWalletUserId(created.id);
                      }
                    }
                    await createWalletForUser(resolvedUserId);
                  } finally {
                    setCreatingWallet(false);
                  }
                }}
              className={`bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 transition-colors ${creatingWallet ? 'opacity-60 cursor-not-allowed' : ''}`}
              disabled={creatingWallet}
            >
              {creatingWallet ? 'Creating…' : 'Create Wallet'}
            </button>
          </div>
        )}
      </div>

      {/* Deposit/Withdraw */}
      {currentWallet && (
        <div className="grid md:grid-cols-2 gap-6">
          <div className="bg-white rounded-lg shadow-md p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Deposit Funds</h3>
            <div className="space-y-4">
              <div>
                <label htmlFor="deposit-amount" className="block text-sm font-medium text-gray-700 mb-1">
                  Amount (USD)
                </label>
                <input
                  id="deposit-amount"
                  type="number"
                  step="0.01"
                  min="0"
                  value={depositAmount}
                  onChange={(e) => setDepositAmount(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="0.00"
                />
              </div>
              <button
                onClick={handleDeposit}
                disabled={!depositAmount || isLoading}
                className="w-full bg-green-600 text-white py-2 px-4 rounded hover:bg-green-700 disabled:bg-gray-300 disabled:cursor-not-allowed transition-colors"
              >
                {isLoading ? 'Processing...' : 'Deposit'}
              </button>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow-md p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Withdraw Funds</h3>
            <div className="space-y-4">
              <div>
                <label htmlFor="withdraw-amount" className="block text-sm font-medium text-gray-700 mb-1">
                  Amount (USD)
                </label>
                <input
                  id="withdraw-amount"
                  type="number"
                  step="0.01"
                  min="0"
                  max={currentWallet.available_balance ?? 0}
                  value={withdrawAmount}
                  onChange={(e) => setWithdrawAmount(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="0.00"
                />
              </div>
              <button
                onClick={handleWithdraw}
                disabled={!withdrawAmount || isLoading}
                className="w-full bg-red-600 text-white py-2 px-4 rounded hover:bg-red-700 disabled:bg-gray-300 disabled:cursor-not-allowed transition-colors"
              >
                {isLoading ? 'Processing...' : 'Withdraw'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Transaction History */}
      {currentWallet && (
        <div className="bg-white rounded-lg shadow-md p-6">
          <h3 className="text-lg font-semibold text-gray-900 mb-4">Recent Transactions</h3>
          {walletTransactions.length === 0 ? (
            <p className="text-gray-500">No transactions yet. Use Deposit or Withdraw to get started.</p>
          ) : (
          <div className="space-y-3">
            {walletTransactions.slice(-5).reverse().map((transaction) => (
              <div key={transaction.id} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                <div className="flex items-center space-x-3">
                  <div className={`w-8 h-8 rounded-full flex items-center justify-center ${
                    transaction.type === 'deposit' ? 'bg-green-100' : 
                    transaction.type === 'withdrawal' ? 'bg-red-100' : 'bg-blue-100'
                  }`}>
                    {transaction.type === 'deposit' && (
                      <svg className="w-4 h-4 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                      </svg>
                    )}
                    {transaction.type === 'withdrawal' && (
                      <svg className="w-4 h-4 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 12H4" />
                      </svg>
                    )}
                  </div>
                  <div>
                    <p className="font-medium text-gray-900 capitalize">{transaction.type}</p>
                    <p className="text-sm text-gray-500">
                      {(() => {
                        const v: any = (transaction as any).created_at
                        if (!v) return 'Unknown date'
                        if (typeof v === 'string') {
                          const d = new Date(v)
                          return isNaN(d.getTime()) ? 'Unknown date' : d.toLocaleString()
                        }
                        if (typeof v === 'number') {
                          const d = new Date(v)
                          return isNaN(d.getTime()) ? 'Unknown date' : d.toLocaleString()
                        }
                        if (typeof v === 'object') {
                          const secs = (v.seconds ?? v.Seconds ?? v._seconds)
                          const nanos = (v.nanos ?? v.Nanos ?? v._nanoseconds ?? 0)
                          if (typeof secs === 'number') {
                            const d = new Date(secs * 1000 + Math.floor(nanos / 1e6))
                            return isNaN(d.getTime()) ? 'Unknown date' : d.toLocaleString()
                          }
                        }
                        return 'Unknown date'
                      })()}
                    </p>
                  </div>
                </div>
                <div className="text-right">
                  <p className={`font-semibold ${
                    transaction.type === 'deposit' ? 'text-green-600' : 'text-red-600'
                  }`}>
                    {transaction.type === 'deposit' ? '+' : '-'}${Math.abs(transaction.amount).toFixed(2)}
                  </p>
                  <p className="text-sm text-gray-500">{transaction.status}</p>
                </div>
              </div>
            ))}
          </div>
          )}
        </div>
      )}
    </div>
  );
}
