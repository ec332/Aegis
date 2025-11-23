"use client";

import Link from "next/link";

export default function Navbar() {
  return (
    <nav className="bg-white border-b border-gray-200">
      <div className="px-16">
        <div className="flex items-center justify-between h-16">
          {/* Left: Logo and Navigation Links */}
          <div className="flex items-center gap-8">
            {/* Aegis Logo */}
            <Link href="/" className="flex items-center">
              <span className="text-xl font-bold text-black">
                Aegis
              </span>
            </Link>

            {/* Navigation Links */}
            <Link
              href="/transactions"
              className="text-sm text-gray-700 hover:text-[#151b4d] transition-colors"
            >
              Transactions
            </Link>
          </div>

          {/* Right: Profile */}
          <div className="flex items-center gap-8">
            <button className="px-4 py-2 bg-[#151b4d] text-white rounded-md hover:bg-[#1a2159] transition-colors">
              Profile
            </button>
          </div>
        </div>
      </div>
    </nav>
  );
}