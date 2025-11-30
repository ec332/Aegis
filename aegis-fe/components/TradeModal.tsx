"use client";

import { Market, Option, Transaction } from "@/types";
import { useState, useEffect } from "react";

interface TradeModalProps {
  market: Market;
  options: Option[];
  onClose: () => void;
  initialTransaction?: Transaction;
}

export default function TradeModal({
  market,
  options,
  onClose,
  initialTransaction,
}: TradeModalProps) {
  const [selectedOption, setSelectedOption] = useState<Option | null>(null);
  const [price, setPrice] = useState<string>(
    initialTransaction?.price.toString() || ""
  );

  // Pre-fill form if editing
  useEffect(() => {
    if (initialTransaction) {
      const option = options.find(
        (opt) => opt.id === initialTransaction.option_id
      );
      if (option) {
        setSelectedOption(option);
      }
      setPrice(initialTransaction.price.toString());
    }
  }, [initialTransaction, options]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedOption) {
      alert("Please select an option");
      return;
    }
    if (!price || parseFloat(price) <= 0) {
      alert("Please enter a valid price");
      return;
    }
    console.log("Trade submitted:", {
      market: market.id,
      option: selectedOption.id,
      price: parseFloat(price),
      isEdit: !!initialTransaction,
      transactionId: initialTransaction?.id,
    });
    // Handle trade submission here
    onClose();
  };

  const marketTitle = market.question || (market as any).title || "Untitled Market";

  const optionLabel = (option: Option) => option.option_text || (option as any).title || "Option";

  return (
    <div className="fixed inset-0 bg-black/20 backdrop-blur-sm flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg shadow-lg max-w-md w-full">
        {/* Header */}
        <div className="border-b border-gray-200 px-6 py-4 flex justify-between items-center">
          <h2 className="text-xl font-bold text-[#151b4d]">
            {initialTransaction ? "Edit Trade" : "Place Trade"}
          </h2>
          <button
            onClick={onClose}
            className="text-gray-500 hover:text-gray-700 text-2xl leading-none"
          >
            ×
          </button>
        </div>

        {/* Content */}
        <div className="px-6 py-4">
          {/* Market Details */}
          <div className="mb-6">
            <h3 className="text-lg font-semibold text-[#151b4d] mb-2">
              {marketTitle}
            </h3>
            <p className="text-sm text-gray-600 mb-3">{market.description}</p>
            <div className="inline-block px-3 py-1 bg-gray-100 text-gray-700 text-xs font-semibold rounded-full">
              {market.status}
            </div>
          </div>

          {/* Options Selection */}
          <div className="mb-6">
            <label className="block text-sm font-semibold text-gray-700 mb-3">
              Select Option
            </label>
            <div className="space-y-2">
              {options.map((option, index) => (
                <button
                  key={option.id}
                  onClick={() => setSelectedOption(option)}
                  className={`w-full px-4 py-3 rounded-md border-2 transition-colors font-medium text-left ${
                    selectedOption?.id === option.id
                      ? index % 2 === 0
                        ? "bg-[#151b4d] text-white border-[#151b4d]"
                        : "bg-[#8a704d] text-white border-[#8a704d]"
                      : "bg-gray-50 text-gray-700 border-gray-200 hover:border-[#151b4d]"
                  }`}
                >
                  <div className="flex flex-col">
                    <span>{optionLabel(option)}</span>
                    {typeof option.current_price === "number" && (
                      <span className="text-xs text-white/80">
                        Price: ${option.current_price.toFixed(2)}
                      </span>
                    )}
                  </div>
                </button>
              ))}
            </div>
          </div>

          {/* Price Input */}
          <div className="mb-6">
            <label
              htmlFor="price"
              className="block text-sm font-semibold text-gray-700 mb-2"
            >
              Number of shares
            </label>
            <input
              id="price"
              type="number"
              step="1"
              min="0"
              placeholder="0"
              value={price}
              onChange={(e) => {
                const v = e.target.value;
                if (/^\d*$/.test(v)) {     // allow only digits
                  setPrice(v);
                }
              }}
              className="w-full px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:border-[#151b4d] focus:ring-2 focus:ring-[#151b4d] focus:ring-opacity-10"
            />
          </div>

          {/* Action Buttons */}
          <div className="flex gap-3">
            <button
              onClick={onClose}
              className="flex-1 px-4 py-2 bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 transition-colors font-medium"
            >
              Cancel
            </button>
            <button
              onClick={handleSubmit}
              disabled={!selectedOption}
              className="flex-1 px-4 py-2 bg-[#151b4d] text-white rounded-md hover:bg-[#1a2159] transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {initialTransaction ? "Update Trade" : "Submit Trade"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
