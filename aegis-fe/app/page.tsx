"use client";

import MarketCard from "@/components/MarketCard";
import TradeModal from "@/components/TradeModal";
import WalletManager from "@/components/WalletManager";
import { useAppStore } from "@/store/appStore";
import { Market, Option } from "@/types";
import { useEffect, useState } from "react";

export default function Home() {
  const { markets, marketOptions, initializeApp, loadOptionsForMarket, isBackendHealthy } =
    useAppStore();
  const [selectedMarket, setSelectedMarket] = useState<{
    market: Market;
    options: Option[];
  } | null>(null);
  const [activeTab, setActiveTab] = useState<'markets' | 'wallet'>('markets');

  // Initialize app on mount
  useEffect(() => {
    initializeApp();
  }, [initializeApp]);

  const handleOptionClick = async (option: Option) => {
    const market = markets.find((m) => m.id === option.market_id);
    if (market) {
      // Load options if not already loaded
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
        {/* Backend Health Status */}
        {!isBackendHealthy && (
          <div className="mb-6 p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
            <div className="flex items-center">
              <svg className="w-5 h-5 text-yellow-600 mr-2" fill="currentColor" viewBox="0 0 20 20">
                <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
              </svg>
              <span className="text-yellow-800">Backend connection issues detected. Some features may be limited.</span>
            </div>
          </div>
        )}

        {/* Tab Navigation */}
        <div className="mb-8">
          <div className="border-b border-gray-200">
            <nav className="-mb-px flex space-x-8">
              <button
                onClick={() => setActiveTab('markets')}
                className={`py-2 px-1 border-b-2 font-medium text-sm ${
                  activeTab === 'markets'
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                Markets
              </button>
              <button
                onClick={() => setActiveTab('wallet')}
                className={`py-2 px-1 border-b-2 font-medium text-sm ${
                  activeTab === 'wallet'
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                Wallet
              </button>
            </nav>
          </div>
        </div>

        {/* Tab Content */}
        {activeTab === 'markets' && (
          <div>
            <h2 className="text-2xl font-bold text-[#151b4d] mb-8">
              Active Markets
            </h2>
            {markets.length > 0 ? (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
                {markets.map((market) => (
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

        {activeTab === 'wallet' && (
          <div>
            <h2 className="text-2xl font-bold text-[#151b4d] mb-8">
              Wallet Management
            </h2>
            <WalletManager userId="demo-user" />
          </div>
        )}

        {/* Trade Modal */}
        {selectedMarket && (
          <TradeModal
            market={selectedMarket.market}
            options={selectedMarket.options}
            onClose={() => setSelectedMarket(null)}
          />
        )}
      </div>
    </main>
  );
}