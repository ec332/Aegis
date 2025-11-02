"use client";

import MarketCard from "@/components/MarketCard";
import TradeModal from "@/components/TradeModal";
import { useAppStore } from "@/store";
import { Market, Option } from "@/types";
import { useEffect, useState } from "react";

export default function Home() {
  const { markets, marketOptions, initializeApp, loadOptionsForMarket } =
    useAppStore();
  const [selectedMarket, setSelectedMarket] = useState<{
    market: Market;
    options: Option[];
  } | null>(null);

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
        {/* Hero Section
        <div className="text-center mb-16">
          <h1 className="text-4xl font-bold text-[#151b4d] mb-4">
            Welcome to Aegis Market
          </h1>
          <p className="text-lg text-gray-600 mb-8">
            Your secure marketplace for trading and transactions
          </p>

          <div className="flex justify-center gap-4">
            <button className="px-6 py-3 bg-[#151b4d] text-white rounded-md hover:bg-[#1a2159] transition-colors font-medium">
              Get Started
            </button>
            <button className="px-6 py-3 bg-white text-[#151b4d] border-2 border-[#151b4d] rounded-md hover:bg-[#151b4d] hover:text-white transition-colors font-medium">
              Learn More
            </button>
          </div>
        </div> */}

        {/* Markets Grid */}
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