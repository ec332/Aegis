"use client";

import MarketCard from "@/components/MarketCard";
import TradeModal from "@/components/TradeModal";
import MarketForm from "@/components/MarketForm";
import { useAppStore } from "@/store/appStore";
import { useAuthStore } from "@/store/authStore";
import { Market, Option } from "@/types";
import { useEffect, useMemo, useState } from "react";
import { DEFAULT_USER_ID } from "@/constants";
import { fetchUserTransactions } from "@/services/api";

export default function MarketsPage() {
  const { markets, marketOptions, loadOptionsForMarket, isBackendHealthy } = useAppStore();
  const { isAuthenticated, profile } = useAuthStore();
  const [selectedMarket, setSelectedMarket] = useState<{ market: Market; options: Option[] } | null>(null);
  const [isFormOpen, setIsFormOpen] = useState(false);
  const [subTab, setSubTab] = useState<"active" | "mine">("active");
  const [tick, setTick] = useState<number>(Date.now());
  const [myMarketIds, setMyMarketIds] = useState<string[]>([]);

  useEffect(() => {}, []);

  useEffect(() => {
    const id = setInterval(() => setTick(Date.now()), 300000);
    return () => clearInterval(id);
  }, []);

  useEffect(() => {
    if (subTab !== "mine") return;
    const userId = profile?.id || DEFAULT_USER_ID;
    if (!userId) return;
    let cancelled = false;
    (async () => {
      try {
        const txs = await fetchUserTransactions(userId);
        const ids = Array.from(new Set(txs.map((t) => t.market_id).filter(Boolean)));
        if (!cancelled) setMyMarketIds(ids);
      } catch {}
    })();
    return () => { cancelled = true; };
  }, [subTab, profile?.id]);

  const parseTs = (value: unknown): Date | null => {
    if (value == null) return null;
    if (value instanceof Date) {
      return Number.isNaN(value.getTime()) ? null : value;
    }
    if (typeof value === "object") {
      const obj = value as { seconds?: number; nanos?: number };
      if (typeof obj.seconds === "number") {
        const nanos = typeof obj.nanos === "number" ? obj.nanos : 0;
        const d = new Date(obj.seconds * 1000 + nanos / 1_000_000);
        return Number.isNaN(d.getTime()) ? null : d;
      }
    }
    const s = typeof value === "string" ? value : String(value);
    const n = s.includes("T") ? s : s.replace(" ", "T");
    const tries = [n, `${n}Z`];
    for (const cand of tries) {
      const d = new Date(cand);
      if (!Number.isNaN(d.getTime())) return d;
    }
    return null;
  };

  const activeMarkets = useMemo(() => {
    const now = tick;
    return markets.filter((m) => {
      const anyM = m as unknown as Record<string, unknown>;
      const raw = anyM["resolution_time"] ?? anyM["resolutionTime"] ?? anyM["end_time"] ?? anyM["endTime"];
      const d = parseTs(raw);
      if (!d) return true;
      return d.getTime() >= now;
    });
  }, [markets, tick]);

  const myMarkets = useMemo(() => {
    if (!myMarketIds.length) return [] as Market[];
    const set = new Set(myMarketIds);
    return markets.filter((m) => set.has(m.id));
  }, [markets, myMarketIds]);

  const handleOptionClick = async (option: Option) => {
    const market = markets.find((m) => m.id === option.market_id);
    if (market) {
      if (!marketOptions[option.market_id]) {
        await loadOptionsForMarket(option.market_id);
      }
      const options = marketOptions[option.market_id] || [];
      setSelectedMarket({ market, options });
    }
  };

  return (
    <main className="bg-white min-h-screen">
      <div className="px-4 sm:px-6 lg:px-8 py-20 max-w-7xl mx-auto">
        {!isBackendHealthy && (
          <div className="mb-6 p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
            <div className="flex items-center">
              <svg className="w-5 h-5 text-yellow-600 mr-2" fill="currentColor" viewBox="0 0 20 20">
                <path
                  fillRule="evenodd"
                  d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                  clipRule="evenodd"
                />
              </svg>
              <span className="text-yellow-800">Backend connection issues detected. Some features may be limited.</span>
            </div>
          </div>
        )}

        <div className="mb-6" />
        <div className="mb-8 flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-4">
            <h2 className="text-2xl font-bold text-[#151b4d]">Markets</h2>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setSubTab("active")}
                className={`px-3 py-1 rounded-md text-sm font-semibold border ${
                  subTab === "active" ? "bg-[#151b4d] text-white border-[#151b4d]" : "bg-gray-50 text-gray-700 border-gray-200"
                }`}
              >
                Active
              </button>
              <button
                type="button"
                onClick={() => setSubTab("mine")}
                className={`px-3 py-1 rounded-md text-sm font-semibold border ${
                  subTab === "mine" ? "bg-[#151b4d] text-white border-[#151b4d]" : "bg-gray-50 text-gray-700 border-gray-200"
                }`}
              >
                My Markets
              </button>
            </div>
          </div>
          {isAuthenticated && (
            <button
              type="button"
              onClick={() => setIsFormOpen(true)}
              className="rounded bg-blue-600 px-4 py-2 text-white shadow hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              Create market
            </button>
          )}
        </div>

        {(subTab === "active" ? activeMarkets : myMarkets).length > 0 ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
            {(subTab === "active" ? activeMarkets : myMarkets).map((market) => (
              <MarketCard
                key={market.id}
                market={market}
                options={marketOptions[market.id] || []}
                onOptionClick={handleOptionClick}
              />
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <p className="text-gray-600 text-lg">No markets to display.</p>
          </div>
        )}

        {selectedMarket && (
          <TradeModal
            market={selectedMarket.market}
            options={selectedMarket.options}
            onClose={() => setSelectedMarket(null)}
          />
        )}

        {isFormOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-2 sm:px-4">
            <div className="relative w-full max-w-[640px] sm:max-w-lg rounded-2xl bg-white p-4 sm:p-6 shadow-2xl max-h-[85vh] overflow-y-auto">
              <div className="mb-4 flex items-center justify-between">
                <h3 className="text-xl font-semibold text-[#151b4d]">Create Market</h3>
                <button
                  type="button"
                  onClick={() => setIsFormOpen(false)}
                  className="rounded-full p-2 text-gray-500 hover:bg-gray-100"
                  aria-label="Close create market form"
                >
                  ✕
                </button>
              </div>
              <MarketForm onCreated={() => setIsFormOpen(false)} />
            </div>
          </div>
        )}
      </div>
    </main>
  );
}
