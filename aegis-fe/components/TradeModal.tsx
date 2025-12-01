"use client";

import { DEFAULT_USER_ID } from "@/constants";
import { createTransaction } from "@/services/api";
import { useAuthStore } from "@/store/authStore";
import {
  Market,
  Option,
  Transaction,
  TransactionType,
} from "@/types";
import { useState, useEffect } from "react";
import type { FormEvent } from "react";

interface TradeModalProps {
  market: Market;
  options: Option[];
  onClose: () => void;
  initialTransaction?: Transaction;
  userId?: string;
}

const TRANSACTION_TYPES: TransactionType[] = ["BUY", "SELL"];

export default function TradeModal({
  market,
  options,
  onClose,
  initialTransaction,
  userId,
}: TradeModalProps) {
  const [selectedOption, setSelectedOption] = useState<Option | null>(null);
  const [transactionType, setTransactionType] = useState<TransactionType>(
    initialTransaction?.transaction_type === "SELL" ? "SELL" : "BUY"
  );
  const [shares, setShares] = useState<string>(
    initialTransaction?.number_of_shares?.toString() || ""
  );
  const [pricePerShare, setPricePerShare] = useState<string>(
    initialTransaction?.price_per_share?.toString() ||
      initialTransaction?.price?.toString() ||
      ""
  );
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const profile = useAuthStore((state) => state.profile);
  const resolvedUserId = userId || profile?.id || DEFAULT_USER_ID;

  // Pre-fill form if editing
  useEffect(() => {
    if (initialTransaction) {
      const option = options.find(
        (opt) => opt.id === initialTransaction.option_id
      );
      if (option) {
        setSelectedOption(option);
      }
      setShares(
        initialTransaction.number_of_shares?.toString() ||
          initialTransaction.price?.toString() ||
          ""
      );
      setPricePerShare(
        initialTransaction.price_per_share?.toString() ||
          initialTransaction.price?.toString() ||
          ""
      );
      setTransactionType(
        initialTransaction.transaction_type === "SELL" ? "SELL" : "BUY"
      );
    } else {
      setShares("");
      setPricePerShare("");
      setTransactionType("BUY");
    }
  }, [initialTransaction, options]);

  // Default to the first option when creating a new trade
  useEffect(() => {
    if (!initialTransaction && options.length > 0 && !selectedOption) {
      const defaultOption = options[0];
      setSelectedOption(defaultOption);
      if (
        !pricePerShare &&
        typeof defaultOption.current_price === "number"
      ) {
        setPricePerShare(defaultOption.current_price.toString());
      }
    }
  }, [initialTransaction, options, pricePerShare, selectedOption]);

  // Keep price-per-share pre-filled with the selected option price if empty
  useEffect(() => {
    if (
      selectedOption &&
      !pricePerShare &&
      typeof selectedOption.current_price === "number"
    ) {
      setPricePerShare(selectedOption.current_price.toString());
    }
  }, [pricePerShare, selectedOption]);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setErrorMessage(null);
    setSuccessMessage(null);
    if (!selectedOption) {
      setErrorMessage("Please select an option before placing your trade.");
      return;
    }
    if (!resolvedUserId) {
      setErrorMessage("A valid user ID is required to place a trade.");
      return;
    }
    const shareCount = parseInt(shares, 10);
    if (!shares || Number.isNaN(shareCount) || shareCount <= 0) {
      setErrorMessage("Enter a valid number of shares (minimum 1).");
      return;
    }
    const parsedPricePerShare = parseFloat(pricePerShare);
    if (
      !pricePerShare ||
      Number.isNaN(parsedPricePerShare) ||
      parsedPricePerShare <= 0
    ) {
      setErrorMessage("Enter a valid price per share greater than 0.");
      return;
    }

    setIsSubmitting(true);
    try {
      await createTransaction({
        user_id: resolvedUserId,
        market_id: market.id,
        option_id: selectedOption.id,
        transaction_type: transactionType,
        number_of_shares: shareCount,
        price_per_share: parsedPricePerShare,
      });
      setSuccessMessage("Trade submitted successfully.");
      onClose();
    } catch (err) {
      const fallback = err instanceof Error ? err.message : null;
      setErrorMessage(fallback || "Failed to submit trade. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  };

  const marketTitle = market.question || (market as any).title || "Untitled Market";

  const optionLabel = (option: Option) => option.option_text || (option as any).title || "Option";

  return (
    <div className="fixed inset-0 bg-black/30 backdrop-blur-sm flex items-center justify-center z-50 p-2 sm:p-4">
      <div className="bg-white rounded-lg shadow-lg w-full max-w-[640px] sm:max-w-md max-h-[85vh] overflow-y-auto">
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
        <form className="px-4 sm:px-6 py-4" onSubmit={handleSubmit}>
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

          {errorMessage && (
            <div className="mb-4 rounded-md border border-red-200 bg-red-50 px-4 py-2 text-sm text-red-700">
              {errorMessage}
            </div>
          )}
          {successMessage && (
            <div className="mb-4 rounded-md border border-emerald-200 bg-emerald-50 px-4 py-2 text-sm text-emerald-700">
              {successMessage}
            </div>
          )}

          {/* Options Selection */}
          <div className="mb-6">
            <label className="block text-sm font-semibold text-gray-700 mb-3">
              Select Option
            </label>
            <div className="space-y-2">
              {options.map((option, index) => (
                <button
                  key={option.id}
                  type="button"
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

          {/* Transaction type */}
          <div className="mb-6">
            <label className="block text-sm font-semibold text-gray-700 mb-3">
              Transaction Type
            </label>
            <div className="grid grid-cols-2 gap-3">
              {TRANSACTION_TYPES.map((type) => (
                <button
                  key={type}
                  type="button"
                  onClick={() => setTransactionType(type)}
                  className={`rounded-md border px-4 py-3 text-sm font-semibold transition-colors ${
                    transactionType === type
                      ? "border-[#151b4d] bg-[#151b4d] text-white"
                      : "border-gray-200 bg-gray-50 text-gray-700 hover:border-[#151b4d]"
                  }`}
                >
                  {type}
                </button>
              ))}
            </div>
          </div>

          {/* Price Input */}
          <div className="mb-6">
            <label
              htmlFor="shares"
              className="block text-sm font-semibold text-gray-700 mb-2"
            >
              Number of shares
            </label>
            <input
              id="shares"
              type="number"
              step="1"
              min="0"
              placeholder="0"
              inputMode="numeric"
              value={shares}
              onChange={(e) => {
                const v = e.target.value;
                if (/^\d*$/.test(v)) {
                  setShares(v);
                }
              }}
              className="w-full px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:border-[#151b4d] focus:ring-2 focus:ring-[#151b4d] focus:ring-opacity-10"
            />
          </div>

          {/* Price per share */}
          <div className="mb-6">
            <label
              htmlFor="pricePerShare"
              className="block text-sm font-semibold text-gray-700 mb-2"
            >
              Price per share
            </label>
            <input
              id="pricePerShare"
              type="number"
              min="0"
              step="0.01"
              placeholder="0.00"
              inputMode="decimal"
              value={pricePerShare}
              onChange={(e) => {
                const v = e.target.value;
                if (/^\d*(\.\d{0,4})?$/.test(v)) {
                  setPricePerShare(v);
                }
              }}
              className="w-full px-4 py-2 border border-gray-300 rounded-md focus:outline-none focus:border-[#151b4d] focus:ring-2 focus:ring-[#151b4d] focus:ring-opacity-10"
            />
          </div>

          {/* Action Buttons */}
          <div className="flex flex-col sm:flex-row gap-3">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 bg-gray-100 text-gray-700 rounded-md hover:bg-gray-200 transition-colors font-medium"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={!selectedOption || isSubmitting}
              className="flex-1 px-4 py-2 bg-[#151b4d] text-white rounded-md hover:bg-[#1a2159] transition-colors font-medium disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isSubmitting
                ? "Submitting..."
                : initialTransaction
                  ? "Update Trade"
                  : "Submit Trade"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
