"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useAuthStore } from "@/store/authStore";

export default function Navbar() {
  const { isAuthenticated, profile, wallet, signIn, signOut, initAuth } = useAuthStore();
  const [open, setOpen] = useState(false);
  const [signing, setSigning] = useState(false);

  useEffect(() => {
    initAuth();
  }, [initAuth]);

  const handleSignIn = async () => {
    try {
      setSigning(true);
      await signIn();
    } catch (e) {
      console.error(e);
      alert("Sign-in failed. Ensure MetaMask is installed and unlocked.");
    } finally {
      setSigning(false);
    }
  };

  const handleToggleProfile = () => setOpen((o) => !o);

  const handleLogout = async () => {
    await signOut();
    setOpen(false);
  };

  const fmt = (v: any): string => {
    if (!v) return "-";
    if (typeof v === "string") return v;
    if (typeof v === "object" && typeof v.seconds === "number") {
      const ms = v.seconds * 1000 + (v.nanos ? v.nanos / 1e6 : 0);
      return new Date(ms).toISOString();
    }
    return String(v);
  };

  return (
    <nav className="bg-white border-b border-gray-200">
      <div className="px-4 sm:px-8 lg:px-16">
        <div className="flex items-center justify-between h-16">
          <div className="flex items-center gap-8">
            <Link href="/" className="flex items-center">
              <span className="text-xl font-bold text-black">Aegis</span>
            </Link>
            <Link href="/transactions" className="text-sm text-gray-700 hover:text-[#151b4d] transition-colors">Transactions</Link>
          </div>

          <div className="relative z-10">
            {!isAuthenticated ? (
              <button type="button" onClick={handleSignIn} className="px-4 py-2 bg-[#151b4d] text-white rounded-md hover:bg-[#1a2159] transition-colors cursor-pointer">
                {signing ? "Signing In..." : "Sign In"}
              </button>
            ) : (
              <>
                <button onClick={handleToggleProfile} className="px-4 py-2 bg-[#151b4d] text-white rounded-md hover:bg-[#1a2159] transition-colors">
                  Profile
                </button>
                {open && (
                  <>
                    <div className="hidden sm:block absolute right-0 mt-2 w-[28rem] max-h-[70vh] overflow-y-auto bg-white border border-gray-200 rounded-md shadow-lg p-6">
                      <div className="text-sm text-gray-800">
                        <div className="font-semibold mb-3">User Profile</div>
                        <div className="mb-2"><span className="font-medium">Wallet:</span> <span className="font-mono break-all">{wallet}</span></div>
                        <div className="mb-2"><span className="font-medium">Role:</span> {profile?.role}</div>
                        <div className="mb-2"><span className="font-medium">Balance:</span> {profile?.balance}</div>
                        <div className="mb-2"><span className="font-medium">Created:</span> {fmt(profile?.created_at)}</div>
                        <div className="mb-2"><span className="font-medium">Last Login:</span> {fmt(profile?.last_login)}</div>
                      </div>
                      <div className="mt-4 flex justify-end">
                        <button onClick={handleLogout} className="px-4 py-2 text-sm bg-red-600 text-white rounded hover:bg-red-700">Logout</button>
                      </div>
                    </div>
                    <div className="sm:hidden fixed inset-0 z-50 bg-black/40 p-4">
                      <div className="mt-auto w-full rounded-t-xl bg-white p-4 max-h-[85vh] overflow-y-auto">
                        <div className="text-sm text-gray-800">
                          <div className="font-semibold mb-3">User Profile</div>
                          <div className="mb-2"><span className="font-medium">Wallet:</span> <span className="font-mono break-all">{wallet}</span></div>
                          <div className="mb-2"><span className="font-medium">Role:</span> {profile?.role}</div>
                          <div className="mb-2"><span className="font-medium">Balance:</span> {profile?.balance}</div>
                          <div className="mb-2"><span className="font-medium">Created:</span> {fmt(profile?.created_at)}</div>
                          <div className="mb-2"><span className="font-medium">Last Login:</span> {fmt(profile?.last_login)}</div>
                        </div>
                        <div className="mt-4 flex justify-end">
                          <button onClick={handleLogout} className="px-4 py-2 text-sm bg-red-600 text-white rounded hover:bg-red-700">Logout</button>
                        </div>
                      </div>
                    </div>
                  </>
                )}
              </>
            )}
          </div>
        </div>
      </div>
    </nav>
  );
}
