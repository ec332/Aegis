"use client";

import TransactionItem from "@/components/TransactionItem";
import TradeModal from "@/components/TradeModal";
import { Transaction, Market, Option } from "@/types";
import { useState } from "react";

// Sample data - replace with API call
const sampleTransactions: Transaction[] = [
  {
    id: "tx1",
    user_id: "user1",
    market_id: "1",
    option_id: "opt1",
    transaction_type: "buy",
    created_at: "2024-11-01T14:30:00Z",
  },
  {
    id: "tx2",
    user_id: "user1",
    market_id: "2",
    option_id: "opt3",
    transaction_type: "sell",
    created_at: "2024-11-01T12:15:00Z",
  },
  {
    id: "tx3",
    user_id: "user1",
    market_id: "3",
    option_id: "opt5",
    transaction_type: "buy",
    created_at: "2024-10-31T09:45:00Z",
  },
];

// Sample markets and options
const sampleMarkets: { [key: string]: Market } = {
  "1": {
    id: "1",
    title: "Will Bitcoin reach $100k?",
    description: "Predict if BTC will hit $100k by end of 2024",
    status: "Active",
  },
  "2": {
    id: "2",
    title: "Will Ethereum outperform Bitcoin?",
    description: "Predict ETH performance vs BTC in Q4",
    status: "Active",
  },
  "3": {
    id: "3",
    title: "Will AI stocks rally?",
    description: "Predict the movement of AI-focused stocks",
    status: "Active",
  },
};

const sampleOptions: { [key: string]: Option } = {
  opt1: { id: "opt1", market_id: "1", title: "Yes" },
  opt2: { id: "opt2", market_id: "1", title: "No" },
  opt3: { id: "opt3", market_id: "2", title: "Yes" },
  opt4: { id: "opt4", market_id: "2", title: "No" },
  opt5: { id: "opt5", market_id: "3", title: "Rally" },
  opt6: { id: "opt6", market_id: "3", title: "Decline" },
};

export default function TransactionsPage() {
  const [transactions, setTransactions] = useState(sampleTransactions);
  const [editingTransaction, setEditingTransaction] = useState<Transaction | null>(
    null
  );

  const handleEdit = (transaction: Transaction) => {
    setEditingTransaction(transaction);
  };

  const handleDelete = (transactionId: string) => {
    setTransactions(transactions.filter((t) => t.id !== transactionId));
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
              {transactions.map((transaction) => (
                <TransactionItem
                  key={transaction.id}
                  transaction={transaction}
                  market={sampleMarkets[transaction.market_id]}
                  option={sampleOptions[transaction.option_id]}
                  onEdit={handleEdit}
                  onDelete={handleDelete}
                />
              ))}
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
      {editingTransaction && (
        <TradeModal
          market={sampleMarkets[editingTransaction.market_id]}
          options={[
            sampleOptions[editingTransaction.option_id],
          ]}
          initialTransaction={editingTransaction}
          onClose={handleCloseModal}
        />
      )}
    </>
  );
}
