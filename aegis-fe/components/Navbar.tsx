"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter, usePathname } from "next/navigation";
import { useAuthStore } from "@/store/authStore";
function fmtTs(v: any): string {
  if (!v) return "-";
  if (typeof v === "string") return v;
  if (typeof v === "number") return new Date(v).toLocaleString();
  if (typeof v === "object" && "seconds" in v) {
    const s = (v as any).seconds;
    return new Date(Number(s) * 1000).toLocaleString();
  }
  return "-";
}

export default function Navbar() {
  const { isAuthenticated, profile, wallet, signOut, initAuth } = useAuthStore();
  const [open, setOpen] = useState(false);
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    initAuth();
  }, [initAuth]);

  const handleSignIn = () => {
    router.push("/signin");
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
            {isAuthenticated && (
              <Link href="/transactions" className="text-sm text-gray-700 hover:text-[#151b4d] transition-colors">Transactions</Link>
            )}
          </div>

          <div className="relative">
            {!isAuthenticated && pathname !== "/signin" && (
              <button type="button" onClick={handleSignIn} className="px-4 py-2 bg-[#151b4d] text-white rounded-md hover:bg-[#1a2159] transition-colors cursor-pointer">
                Sign In
              </button>
            )}
            {isAuthenticated && pathname !== "/signin" && (
              <>
                <button onClick={handleToggleProfile} className="px-4 py-2 bg-[#151b4d] text-white rounded-md hover:bg-[#1a2159] transition-colors">
                  Profile
                </button>
                {open && (
                  <div className="absolute right-0 mt-2 w-80 bg-white border border-gray-200 rounded-md shadow-lg p-4">
                    <div className="text-sm text-gray-800">
                      <div className="font-semibold mb-2">User Profile</div>
                      <div className="mb-1"><span className="font-medium">Wallet:</span> {wallet}</div>
                      <div className="mb-1"><span className="font-medium">Role:</span> {profile?.role}</div>
                      <div className="mb-1"><span className="font-medium">Balance:</span> {profile?.balance}</div>
                      <div className="mb-1"><span className="font-medium">Created:</span> {fmtTs(profile?.created_at)}</div>
                      <div className="mb-1"><span className="font-medium">Last Login:</span> {fmtTs(profile?.last_login)}</div>
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
