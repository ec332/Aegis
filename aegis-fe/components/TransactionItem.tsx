"use client";

import { Transaction, Market, Option } from "@/types";

interface TransactionItemProps {
  transaction: Transaction;
  market: Market;
  option: Option;
  onEdit: (transaction: Transaction) => void;
  onDelete: (transactionId: string) => void;
}

export default function TransactionItem({
  transaction,
  market,
  option,
  onEdit,
  onDelete,
}: TransactionItemProps) {
  const date = new Date(transaction.created_at).toLocaleDateString("en-US", {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-6 hover:shadow-md transition-shadow">
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
        {/* Left: Transaction Details */}
        <div className="flex-1">
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
            {/* Market Title */}
            <div>
              <label className="text-xs font-semibold text-gray-500 uppercase">
                Market
              </label>
              <p className="text-sm font-medium text-gray-900">{market.title}</p>
            </div>

            {/* Market Description */}
            <div>
              <label className="text-xs font-semibold text-gray-500 uppercase">
                Description
              </label>
              <p className="text-sm text-gray-600">{market.description}</p>
            </div>

            {/* Option */}
            <div>
              <label className="text-xs font-semibold text-gray-500 uppercase">
                Option
              </label>
              <p className="text-sm font-medium text-gray-900">{option.title}</p>
            </div>

            {/* Price */}
            <div>
              <label className="text-xs font-semibold text-gray-500 uppercase">
                Price
              </label>
              <p className="text-sm font-medium text-gray-900">${transaction.price.toFixed(2)}</p>
            </div>
            
            {/* Date */}
            <div>
              <label className="text-xs font-semibold text-gray-500 uppercase">
                Date
              </label>
              <p className="text-sm font-medium text-gray-600">{date}</p>
            </div>
          </div>
        </div>

        {/* Right: Action Buttons */}
        <div className="flex gap-3 lg:flex-col lg:w-auto">
          <button
            onClick={() => onEdit(transaction)}
            className="flex-1 lg:flex-initial px-4 py-2 bg-[#151b4d] text-white rounded-md hover:bg-[#1a2159] transition-colors font-medium text-sm"
          >
            Edit
          </button>
          <button
            onClick={() => onDelete(transaction.id)}
            className="flex-1 lg:flex-initial px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 transition-colors font-medium text-sm"
          >
            Delete
          </button>
        </div>
      </div>
    </div>
  );
}
