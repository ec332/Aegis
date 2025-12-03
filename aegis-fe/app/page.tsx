"use client";

import MarketCard from "@/components/MarketCard";
import TradeModal from "@/components/TradeModal";
import WalletManager from "@/components/WalletManager";
import MarketForm from "@/components/MarketForm";
import { useAppStore } from "@/store/appStore";
import { Market, Option } from "@/types";
import { DEFAULT_USER_ID } from "@/constants";
import { useEffect, useState } from "react";
import { useAuthStore } from "@/store/authStore";

export default function Home() {
  const { markets, marketOptions, loadOptionsForMarket, isBackendHealthy } =
    useAppStore();
  const { isAuthenticated, profile } = useAuthStore();
  const [selectedMarket, setSelectedMarket] = useState<{
    market: Market;
    options: Option[];
  } | null>(null);
  const [activeTab, setActiveTab] = useState<"markets" | "wallet">("markets");
  const [isFormOpen, setIsFormOpen] = useState(false);

  useEffect(() => {
    // No-op: app initialization is handled globally by AppInitializer
  }, []);

  useEffect(() => {
    if (!isAuthenticated && activeTab === "wallet") {
      setActiveTab("markets");
    }
  }, [isAuthenticated, activeTab]);

  useEffect(() => {
    if (typeof window === "undefined") return;
    const sp = new URLSearchParams(window.location.search);
    const tab = sp.get("tab");
    if (tab === "wallet" && isAuthenticated) {
      setActiveTab("wallet");
    } else if (tab === "markets") {
      setActiveTab("markets");
    }
  }, [isAuthenticated]);

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

  const activeMarkets = markets.filter((m) => {
    const anyM = m as unknown as Record<string, unknown>;
    const raw = anyM["resolution_time"] ?? anyM["resolutionTime"] ?? anyM["end_time"] ?? anyM["endTime"];
    const d = parseTs(raw);
    if (!d) return true;
    return d.getTime() >= Date.now();
  });

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
              <span className="text-yellow-800">
                Backend connection issues detected. Some features may be limited.
              </span>
            </div>
          </div>
        )}

        <div className="mb-6" />

        {activeTab === "markets" && (
          <div>
            <div className="mb-8 flex flex-wrap items-center justify-between gap-4">
              <h2 className="text-2xl font-bold text-[#151b4d]">Active Markets</h2>
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
            {activeMarkets.length > 0 ? (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
                {activeMarkets.map((market) => (
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
                <p className="text-gray-600 text-lg">Loading markets...</p>
              </div>
            )}
          </div>
        )}

        {isAuthenticated && activeTab === "wallet" && (
          <div>
            <h2 className="text-2xl font-bold text-[#151b4d] mb-8">Wallet Management</h2>
            <WalletManager userId={profile?.id || ""} />
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
