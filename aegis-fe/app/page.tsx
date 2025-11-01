"use client";

import MarketCard from "@/components/MarketCard";
import TradeModal from "@/components/TradeModal";
import { Market, Option } from "@/types";
import { useState } from "react";

// Sample data - replace with API call
const sampleMarkets: Market[] = [
  {
    id: "1",
    title: "Will Bitcoin reach $100k?",
    description: "Predict if BTC will hit $100k by end of 2024",
    status: "Active",
  },
  {
    id: "2",
    title: "Will Ethereum outperform Bitcoin?",
    description: "Predict ETH performance vs BTC in Q4",
    status: "Active",
  },
  {
    id: "3",
    title: "Will AI stocks rally?",
    description: "Predict the movement of AI-focused stocks",
    status: "Active",
  },
  {
    id: "5",
    title: "Tech IPO Q4 2024",
    description: "Will there be a major tech IPO?",
    status: "Active",
  },
  {
    id: "6",
    title: "Gold price forecast",
    description: "Will gold break $2500/oz?",
    status: "Active",
  },
];

const sampleOptions: { [key: string]: Option[] } = {
  "1": [
    { id: "opt1", market_id: "1", title: "Yes" },
    { id: "opt2", market_id: "1", title: "No" },
  ],
  "2": [
    { id: "opt3", market_id: "2", title: "Yes" },
    { id: "opt4", market_id: "2", title: "No" },
  ],
  "3": [
    { id: "opt5", market_id: "3", title: "Rally" },
    { id: "opt6", market_id: "3", title: "Decline" },
  ],
  "5": [
    { id: "opt9", market_id: "5", title: "Yes" },
    { id: "opt10", market_id: "5", title: "No" },
  ],
};

export default function Home() {
  const [selectedMarket, setSelectedMarket] = useState<{
    market: Market;
    options: Option[];
  } | null>(null);

  const handleOptionClick = (option: Option) => {
    const market = sampleMarkets.find((m) => m.id === option.market_id);
    const options = sampleOptions[option.market_id] || [];
    if (market) {
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
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6">
            {sampleMarkets.map((market) => (
              <MarketCard
                key={market.id}
                market={market}
                options={sampleOptions[market.id] || []}
                onOptionClick={handleOptionClick}
              />
            ))}
          </div>
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