"use client";

import TransactionItem from "@/components/TransactionItem";
import TradeModal from "@/components/TradeModal";
import { fetchMarketById, fetchOptionsByMarketId, fetchUserTransactions } from "@/services/api";
import { useAuthStore } from "@/store/authStore";
import { Market, Option, Transaction } from "@/types";
import { useEffect, useMemo, useRef, useState } from "react";

export default function TransactionsPage() {
  const { isAuthenticated, profile } = useAuthStore();
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [editingTransaction, setEditingTransaction] = useState<Transaction | null>(
    null
  );
  const [transactionDetails, setTransactionDetails] = useState<{
    [key: string]: { market: Market; option: Option };
  }>({});
  const transactionDetailsRef = useRef<{ [key: string]: { market: Market; option: Option } }>({});

  useEffect(() => {
    transactionDetailsRef.current = transactionDetails;
  }, [transactionDetails]);

  // Load user's transactions on mount and when auth changes
  useEffect(() => {
    if (!isAuthenticated || !profile?.id) {
      setTransactions([]);
      return;
    }

    let cancelled = false;
    const loadTransactions = async () => {
      setIsLoading(true);
      setErrorMessage(null);
      try {
        const result = await fetchUserTransactions(profile.id);
        if (!cancelled) {
          setTransactions(result);
        }
      } catch (error) {
        if (!cancelled) {
          const fallback = error instanceof Error ? error.message : "Failed to load transactions.";
          setErrorMessage(fallback);
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    loadTransactions();
    return () => {
      cancelled = true;
    };
  }, [isAuthenticated, profile?.id]);

  // Load market and option details for each transaction
  useEffect(() => {
    if (transactions.length === 0) {
      setTransactionDetails({});
      return;
    }

    let cancelled = false;

    const loadDetails = async () => {
      const currentDetails = transactionDetailsRef.current;
      const missing = transactions.filter((transaction) => !currentDetails[transaction.id]);
      if (missing.length === 0) return;

      try {
        const detailEntries = await Promise.all(
          missing.map(async (transaction) => {
            const [market, options] = await Promise.all([
              fetchMarketById(transaction.market_id),
              fetchOptionsByMarketId(transaction.market_id),
            ]);
            const option = options.find((o) => o.id === transaction.option_id);
            if (market && option) {
              return [transaction.id, { market, option }] as const;
            }
            return null;
          })
        );

        const validEntries = detailEntries.filter(
          (entry): entry is [string, { market: Market; option: Option }] => entry !== null
        );

        if (!cancelled && validEntries.length > 0) {
          setTransactionDetails((prev) => {
            const next = { ...prev };
            for (const [id, detail] of validEntries) {
              next[id] = detail;
            }
            return next;
          });
        }
      } catch (error) {
        console.error("Failed to load market/option details", error);
      }
    };

    loadDetails();
    return () => {
      cancelled = true;
    };
  }, [transactions]);

  const transactionList = useMemo(() => {
    return transactions.filter((transaction) => transactionDetails[transaction.id]);
  }, [transactions, transactionDetails]);

  const handleEdit = (transaction: Transaction) => {
    setEditingTransaction(transaction);
  };

  const handleCloseModal = () => {
    setEditingTransaction(null);
  };

  return (
    <>
      <main className="bg-white min-h-screen">
        <div className="px-4 sm:px-6 lg:px-8 py-20 max-w-7xl mx-auto">
          {/* Header */}
          <div className="mb-8">
            <h1 className="text-3xl font-bold text-[#151b4d] mb-2">
              My Transactions
            </h1>
            <p className="text-gray-600">
              Manage and track all your trades and transactions
            </p>
          </div>

          {/* Transactions List */}
          {!isAuthenticated && (
            <div className="text-center py-12">
              <p className="text-gray-600 text-lg mb-4">
                Please sign in to view your transactions.
              </p>
            </div>
          )}

          {isAuthenticated && (
            <div className="space-y-4">
              {errorMessage && (
                <div className="rounded-md border border-red-200 bg-red-50 px-4 py-2 text-sm text-red-700">
                  {errorMessage}
                </div>
              )}
              {isLoading && (
                <div className="text-center py-8 text-gray-500">Loading transactions…</div>
              )}
              {!isLoading && transactionList.length === 0 && !errorMessage && (
                <div className="text-center py-12">
                  <p className="text-gray-600 text-lg mb-4">No transactions yet</p>
                  <p className="text-gray-500">Start trading to see your transactions here</p>
                </div>
              )}
              {transactionList.length > 0 && (
                <div className="space-y-4">
                  {transactionList.map((transaction) => {
                    const details = transactionDetails[transaction.id]!;
                    return (
                      <TransactionItem
                        key={transaction.id}
                        transaction={transaction}
                        market={details.market}
                        option={details.option}
                      />
                    );
                  })}
                </div>
              )}
            </div>
          )}
        </div>
      </main>

      {/* Edit Modal */}
      {editingTransaction && transactionDetails[editingTransaction.id] && (
        <TradeModal
          market={transactionDetails[editingTransaction.id].market}
          options={[transactionDetails[editingTransaction.id].option]}
          initialTransaction={editingTransaction}
          onClose={handleCloseModal}
        />
      )}
    </>
  );
}
