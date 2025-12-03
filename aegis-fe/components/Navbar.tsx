"use client";

import Link from "next/link";
import { useEffect, useState, useRef } from "react";
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
  const profileBtnRef = useRef<HTMLButtonElement | null>(null);
  const menuRef = useRef<HTMLDivElement | null>(null);

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

  useEffect(() => {
    if (!open) return;
    const el = menuRef.current;
    const trigger = profileBtnRef.current;
    if (!el) return;
    const focusables = el.querySelectorAll<HTMLElement>('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
    const first = focusables[0];
    const last = focusables[focusables.length - 1];
    first?.focus();
    const keyHandler = (ev: KeyboardEvent) => {
      if (ev.key === "Escape") {
        setOpen(false);
        trigger?.focus();
      } else if (ev.key === "Tab") {
        if (focusables.length === 0) return;
        const active = document.activeElement as HTMLElement;
        if (ev.shiftKey) {
          if (active === first) {
            ev.preventDefault();
            last?.focus();
          }
        } else {
          if (active === last) {
            ev.preventDefault();
            first?.focus();
          }
        }
      }
    };
    el.addEventListener("keydown", keyHandler);
    const clickOutside = (ev: MouseEvent) => {
      const path = ev.composedPath() as EventTarget[];
      if (!path.includes(el) && !path.includes(trigger as EventTarget)) {
        setOpen(false);
        trigger?.focus();
      }
    };
    document.addEventListener("mousedown", clickOutside);
    return () => el.removeEventListener("keydown", keyHandler);
  }, [open]);

  return (
    <nav className="bg-white border-b border-gray-200" role="navigation" aria-label="Main">
      <div className="px-4 sm:px-8 lg:px-16">
        <div className="flex items-center justify-between h-16">
          <div className="flex items-center gap-8">
            <Link href="/" className="flex items-center">
              <span className="text-xl font-bold text-black">Aegis</span>
            </Link>
            <Link href="/markets" className="text-sm text-gray-700 hover:text-[#151b4d] transition-colors">Markets</Link>
            {isAuthenticated && (
              <Link href="/transactions" className="text-sm text-gray-700 hover:text-[#151b4d] transition-colors">Transactions</Link>
            )}
            {isAuthenticated && (
              <Link href="/wallet" className="text-sm text-gray-700 hover:text-[#151b4d] transition-colors">Wallet</Link>
            )}
          </div>

          <div className="relative">
            {!isAuthenticated && pathname !== "/signin" && (
              <button type="button" onClick={handleSignIn} className="px-4 py-2 bg-[#151b4d] text-white rounded-md hover:bg-[#1a2159] transition-colors cursor-pointer" aria-label="Sign In">
                Sign In
              </button>
            )}
            {isAuthenticated && pathname !== "/signin" && (
              <>
                <button ref={profileBtnRef} onClick={handleToggleProfile} className="px-4 py-2 bg-[#151b4d] text-white rounded-md hover:bg-[#1a2159] transition-colors" aria-haspopup="menu" aria-expanded={open} aria-controls="profile-menu">
                  Profile
                </button>
                {open && (
                  <>
                    <div id="profile-menu" role="menu" className="absolute right-0 mt-2 w-80 bg-white border border-gray-200 rounded-md shadow-lg p-4" ref={menuRef}>
                      <div className="text-sm text-gray-800">
                        <div className="font-semibold mb-2">User Profile</div>
                        <div className="mb-1"><span className="font-medium">Wallet:</span> {wallet}</div>
                        <div className="mb-1"><span className="font-medium">Role:</span> {profile?.role}</div>
                        <div className="mb-1"><span className="font-medium">Balance:</span> {profile?.balance}</div>
                        <div className="mb-1"><span className="font-medium">Created:</span> {fmtTs(profile?.created_at)}</div>
                        <div className="mb-1"><span className="font-medium">Last Login:</span> {fmtTs(profile?.last_login)}</div>
                      </div>
                    </div>
                    <div className="sm:hidden fixed inset-0 z-50 bg-black/40 p-4" role="dialog" aria-modal="true" aria-labelledby="profile-mobile-title">
                      <div className="mt-auto w-full rounded-t-xl bg-white p-4 max-h-[85vh] overflow-y-auto">
                        <div className="text-sm text-gray-800">
                          <div id="profile-mobile-title" className="font-semibold mb-3">User Profile</div>
                          <div className="mb-2"><span className="font-medium">Wallet:</span> <span className="font-mono break-all">{wallet}</span></div>
                          <div className="mb-2"><span className="font-medium">Role:</span> {profile?.role}</div>
                          <div className="mb-2"><span className="font-medium">Balance:</span> {profile?.balance}</div>
                          <div className="mb-2"><span className="font-medium">Created:</span> {fmt(profile?.created_at)}</div>
                          <div className="mb-2"><span className="font-medium">Last Login:</span> {fmt(profile?.last_login)}</div>
                        </div>
                        <div className="mt-4 flex justify-end">
                          <button onClick={handleLogout} className="px-4 py-2 text-sm bg-red-600 text-white rounded hover:bg-red-700" aria-label="Logout">Logout</button>
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
