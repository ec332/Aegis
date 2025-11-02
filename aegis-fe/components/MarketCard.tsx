"use client";

import { Market, Option } from "@/types";

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
  return (
    <div className="bg-white border border-gray-200 rounded-lg shadow-md p-8 hover:shadow-lg transition-shadow">
      {/* Market Title (Center) */}
      <div className="text-center mb-6">
        <h2 className="text-2xl font-bold text-[#151b4d] mb-2">
          {market.title}
        </h2>
        <p className="text-sm text-gray-600 mb-3">{market.description}</p>
        <span className="inline-block px-3 py-1 bg-gray-100 text-gray-700 text-xs font-semibold rounded-full">
          {market.status}
        </span>
      </div>

      {/* Options (Buttons) */}
      <div className="flex flex-col gap-3 sm:flex-row sm:gap-4 justify-center">
        {options.map((option, index) => (
          <button
            key={option.id}
            onClick={() => onOptionClick?.(option)}
            className={`px-6 py-3 text-white rounded-md transition-colors font-medium flex-1 ${
              index % 2 === 0
                ? "bg-[#151b4d] hover:bg-[#1a2159]"
                : "bg-[#8a704d] hover:bg-[#9d7e5a]"
            }`}
          >
            {option.title}
          </button>
        ))}
      </div>
    </div>
  );
}
