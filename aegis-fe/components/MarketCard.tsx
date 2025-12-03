"use client";

import { Market, Option } from "@/types";
import { useEffect, useState } from "react";
import { useAuthStore } from "@/store/authStore";
import { DEFAULT_USER_ID } from "@/constants";
import { fetchUserTransactionsForMarket } from "@/services/api";

interface MarketCardProps {
  market: Market;
  options: Option[];
  onOptionClick?: (option: Option) => void;
}

export default function MarketCard({
  market,
  options,
  onOptionClick,
}: MarketCardProps) {
  const marketTitle = market.question || (market as any).title || "Untitled Market";
  const formatPrice = (price?: number) =>
    typeof price === "number" ? `$${price.toFixed(2)}` : "–";
  const badgeText = market.description || market.status;
  const [holdings, setHoldings] = useState<Record<string, number>>({});
  const userId = useAuthStore((s) => s.profile?.id) || DEFAULT_USER_ID;
  const parseTimestamp = (value: unknown): Date | null => {
    if (value == null) return null;
    if (value instanceof Date) {
      return Number.isNaN(value.getTime()) ? null : value;
    }
    if (typeof value === "object") {
      const obj = value as { seconds?: number; nanos?: number };
      if (typeof obj.seconds === "number") {
        const nanos = typeof obj.nanos === "number" ? obj.nanos : 0;
        const date = new Date(obj.seconds * 1000 + nanos / 1_000_000);
        return Number.isNaN(date.getTime()) ? null : date;
      }
    }
    const asString = typeof value === "string" ? value : String(value);
    const normalized = asString.includes("T") ? asString : asString.replace(" ", "T");
    const attempts = [normalized, `${normalized}Z`];
    for (const candidate of attempts) {
      const date = new Date(candidate);
      if (!Number.isNaN(date.getTime())) return date;
    }
    return null;
  };
  const m = market as unknown as Record<string, unknown>;
  const rawResolutionValue = (m["resolution_time"] ?? m["resolutionTime"] ?? m["end_time"] ?? m["endTime"]) as unknown;
  const resolutionDate = parseTimestamp(rawResolutionValue);
  const isExpired = !!resolutionDate && resolutionDate.getTime() < Date.now();
  const formattedResolutionDate = resolutionDate
    ? resolutionDate.toLocaleString("en-US", {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
      })
    : "—";

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        if (!userId) return;
        const txs = await fetchUserTransactionsForMarket(userId, market.id);
        const byOption: Record<string, number> = {};
        for (const tx of txs) {
          const shares = tx.number_of_shares || 0;
          const isSell = typeof tx.transaction_type === "string"
            ? tx.transaction_type.toUpperCase() === "SELL" || tx.transaction_type === "1"
            : typeof tx.transaction_type === "number"
              ? tx.transaction_type === 1
              : false;
          const sign = isSell ? -1 : 1;
          if (tx.option_id) {
            byOption[tx.option_id] = (byOption[tx.option_id] || 0) + sign * shares;
          }
        }
        if (!cancelled) setHoldings(byOption);
      } catch {}
    })();
    return () => { cancelled = true; };
  }, [userId, market.id]);

  return (
    <div className="bg-white border border-gray-200 rounded-lg shadow-md p-4 sm:p-6 lg:p-8 hover:shadow-lg transition-shadow">
      {/* Market Title (Center) */}
      <div className="text-center mb-4 sm:mb-6">
        <h2 className="text-xl sm:text-2xl font-bold text-[#151b4d] mb-2">
          {marketTitle}
        </h2>
        <p className="text-sm text-gray-600 mb-3">{market.description}</p>
        <div className="text-xs text-gray-500">Resolves: {formattedResolutionDate}</div>
        {/* <span className="inline-block px-3 py-1 bg-gray-100 text-gray-700 text-xs font-semibold rounded-full">
          {badgeText}
        </span> */}
      </div>

      <div className="flex flex-col gap-3 sm:flex-row sm:gap-4 justify-center">
        {options.map((option, index) => {
          const label = option.option_text || (option as any).title || "Option";
          const price = formatPrice(option.current_price);
          const owned = holdings[option.id] || 0;
          if (isExpired) {
            return (
              <div
                key={option.id}
                className={`px-6 py-3 text-white rounded-md transition-colors font-medium flex-1 ${
                  index % 2 === 0 ? "bg-gray-600" : "bg-gray-500"
                }`}
              >
                <div className="flex flex-col">
                  <span className="text-base sm:text-lg font-semibold">{label}</span>
                  <span className="text-xs text-white/80">Owned: {owned.toFixed(0)} shares</span>
                </div>
              </div>
            );
          }
          return (
            <button
              key={option.id}
              onClick={() => onOptionClick?.(option)}
              className={`px-6 py-3 text-white rounded-md transition-colors font-medium flex-1 ${
                index % 2 === 0
                  ? "bg-[#151b4d] hover:bg-[#1a2159]"
                  : "bg-[#8a704d] hover:bg-[#9d7e5a]"
              }`}
            >
              <div className="flex flex-col">
                <span className="text-base sm:text-lg font-semibold">{label}</span>
                <span className="text-xs text-white/80">{price}</span>
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
