"use client";

import { Transaction, Market, Option } from "@/types";

interface TransactionItemProps {
  transaction: Transaction;
  market: Market;
  option: Option;
  // onEdit: (transaction: Transaction) => void;
  // onDelete: (transactionId: string) => void;
}

export default function TransactionItem({
  transaction,
  market,
  option,
  // onEdit,
  // onDelete,
}: TransactionItemProps) {
  const normalizeTransactionType = (value: unknown): string => {
    if (typeof value === "string" && value.trim().length > 0) {
      return value;
    }

    if (typeof value === "number") {
      return value === 0 ? "BUY" : "SELL";
    }

    if (value && typeof value === "object") {
      const candidateKeys = ["transaction_type", "type", "value", "name", "label"] as const;
      for (const key of candidateKeys) {
        const candidate = (value as Record<string, unknown>)[key];
        if (typeof candidate === "string" && candidate.trim().length > 0) {
          return candidate;
        }
      }
    }

    return "BUY";
  };

  const parseTimestamp = (value?: string | number | Date | { seconds?: number; nanos?: number } | null) => {
    if (!value) return null;

    if (value instanceof Date) {
      return Number.isNaN(value.getTime()) ? null : value;
    }

    if (typeof value === "object") {
      const seconds = typeof (value as any).seconds === "number" ? (value as any).seconds : null;
      if (seconds !== null) {
        const nanos = typeof (value as any).nanos === "number" ? (value as any).nanos : 0;
        const date = new Date(seconds * 1000 + nanos / 1_000_000);
        return Number.isNaN(date.getTime()) ? null : date;
      }
    }

    const asString = typeof value === "string" ? value : String(value);
    const normalized = asString.includes("T") ? asString : asString.replace(" ", "T");
    const attempts = [normalized, `${normalized}Z`];
    for (const candidate of attempts) {
      const date = new Date(candidate);
      if (!Number.isNaN(date.getTime())) {
        return date;
      }
    }
    return null;
  };

  const rawResolutionValue =
    (market as any).resolution_time ??
    market.resolution_time ??
    (market as any).resolutionTime ??
    (market as any).end_time ??
    (market as any).endTime ??
    null;

  const parsedDate = parseTimestamp(transaction.created_at);
  const formattedDate = parsedDate
    ? parsedDate.toLocaleString("en-US", {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      })
    : "Invalid Date";
  const resolutionDate = parseTimestamp(rawResolutionValue);
  const formattedResolutionDate = resolutionDate
    ? resolutionDate.toLocaleString("en-US", {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      })
    : "—";

  const marketTitle = (market as any).question || (market as any).title || "Untitled Market";
  const optionTitle = (option as any).option_text || (option as any).title || "Option";
  const numberOfShares = transaction.number_of_shares ?? null;
  const pricePerShare = transaction.price_per_share ?? (transaction.price && numberOfShares ? transaction.price / numberOfShares : transaction.price);
  const notional = numberOfShares && pricePerShare ? numberOfShares * pricePerShare : transaction.price;
  const transactionType = normalizeTransactionType(transaction.transaction_type).toUpperCase();
  const typeBadgeClass = transactionType === "SELL"
    ? "border-red-200 bg-red-50 text-red-700"
    : "border-emerald-200 bg-emerald-50 text-emerald-700";

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-6 hover:shadow-md transition-shadow">
      <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
        {/* Left: Transaction Details */}
        <div className="flex-1 space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
            {/* Market Title */}
            <div>
              <label className="text-xs font-semibold text-gray-500 uppercase">
                Market
              </label>
              <p className="text-sm font-medium text-gray-900">{marketTitle}</p>
            </div>

            {/* Market Description */}
            <div>
              <label className="text-xs font-semibold text-gray-500 uppercase">
                Description
              </label>
              <p className="text-sm text-gray-600">{market.description || "—"}</p>
            </div>

            {/* Resolution Time */}
            <div>
              <label className="text-xs font-semibold text-gray-500 uppercase">
                Resolution
              </label>
              <p className="text-sm font-medium text-gray-600">{formattedResolutionDate}</p>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-6 gap-4">
            {/* Option */}
            <div>
              <label className="text-xs font-semibold text-gray-500 uppercase">
                Option
              </label>
              <p className="text-sm font-medium text-gray-900">{optionTitle}</p>
            </div>

            {/* Transaction Type */}
            <div>
              <label className="text-xs font-semibold text-gray-500 uppercase">
                Type
              </label>
              <div>
                <span className={`inline-flex items-center rounded-full border px-3 py-0.5 text-xs font-semibold ${typeBadgeClass}`}>
                  {transactionType}
                </span>
              </div>
            </div>

            {/* Shares */}
            <div>
              <label className="text-xs font-semibold text-gray-500 uppercase">
                Shares
              </label>
              <p className="text-sm font-medium text-gray-900">
                {typeof numberOfShares === "number" && !Number.isNaN(numberOfShares)
                  ? numberOfShares.toLocaleString()
                  : "—"}
              </p>
            </div>

            {/* Price per share */}
            <div>
              <label className="text-xs font-semibold text-gray-500 uppercase">
                Price / Share
              </label>
              <p className="text-sm font-medium text-gray-900">
                {typeof pricePerShare === "number" && !Number.isNaN(pricePerShare)
                  ? `$${pricePerShare.toFixed(2)}`
                  : "—"}
              </p>
            </div>

            {/* Notional */}
            <div>
              <label className="text-xs font-semibold text-gray-500 uppercase">
                Total Value
              </label>
              <p className="text-sm font-medium text-gray-900">
                {typeof notional === "number" && !Number.isNaN(notional)
                  ? `$${notional.toFixed(2)}`
                  : "—"}
              </p>
            </div>

            {/* Time Placed */}
            <div>
              <label className="text-xs font-semibold text-gray-500 uppercase">
                Time Placed
              </label>
              <p className="text-sm font-medium text-gray-600">{formattedDate}</p>
            </div>
          </div>
        </div>

        {/* Right: Action Buttons */}
        {/* <div className="flex gap-3 lg:flex-col lg:w-auto">
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
        </div> */}
      </div>
    </div>
  );
}
