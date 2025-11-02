"use client";

import TransactionItem from "@/components/TransactionItem";
import TradeModal from "@/components/TradeModal";
import { useAppStore } from "@/store";
import { Transaction, Market, Option } from "@/types";
import { useEffect, useState } from "react";
import { fetchMarketById, fetchOptionById } from "@/services/api";

export default function TransactionsPage() {
  const { transactions, loadTransactions, removeTransaction } = useAppStore();
  const [editingTransaction, setEditingTransaction] = useState<Transaction | null>(
    null
  );
  const [transactionDetails, setTransactionDetails] = useState<{
    [key: string]: { market: Market; option: Option };
  }>({});

  // Load transactions on mount
  useEffect(() => {
    loadTransactions();
  }, [loadTransactions]);

  // Load market and option details for each transaction
  useEffect(() => {
    const loadDetails = async () => {
      const details: { [key: string]: { market: Market; option: Option } } = {};
      
      for (const transaction of transactions) {
        if (!transactionDetails[transaction.id]) {
          const market = await fetchMarketById(transaction.market_id);
          const option = await fetchOptionById(transaction.option_id);
          
          if (market && option) {
            details[transaction.id] = { market, option };
          }
        }
      }
      
      if (Object.keys(details).length > 0) {
        setTransactionDetails((prev) => ({ ...prev, ...details }));
      }
    };
    
    if (transactions.length > 0) {
      loadDetails();
    }
  }, [transactions, transactionDetails]);

  const handleEdit = (transaction: Transaction) => {
    setEditingTransaction(transaction);
  };

  const handleDelete = async (transactionId: string) => {
    await removeTransaction(transactionId);
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
          {transactions.length > 0 ? (
            <div className="space-y-4">
              {transactions.map((transaction) => {
                const details = transactionDetails[transaction.id];
                if (!details) return null;
                
                return (
                  <TransactionItem
                    key={transaction.id}
                    transaction={transaction}
                    market={details.market}
                    option={details.option}
                    onEdit={handleEdit}
                    onDelete={handleDelete}
                  />
                );
              })}
            </div>
          ) : (
            <div className="text-center py-12">
              <p className="text-gray-600 text-lg mb-4">
                No transactions yet
              </p>
              <p className="text-gray-500">
                Start trading to see your transactions here
              </p>
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
