"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import WalletManager from "@/components/WalletManager";
import { useAuthStore } from "@/store/authStore";

export default function WalletPage() {
  const { isAuthenticated, profile } = useAuthStore();
  const router = useRouter();

  useEffect(() => {
    if (!isAuthenticated) {
      router.replace("/signin");
    }
  }, [isAuthenticated, router]);

  if (!isAuthenticated) {
    return (
      <main className="bg-white min-h-screen">
        <div className="px-4 sm:px-6 lg:px-8 py-20 max-w-3xl mx-auto">
          <p className="text-gray-600">Redirecting to sign in…</p>
        </div>
      </main>
    );
  }

  return (
    <main className="bg-white min-h-screen">
      <div className="px-4 sm:px-6 lg:px-8 py-20 max-w-7xl mx-auto">
        <h2 className="text-2xl font-bold text-[#151b4d] mb-8">Wallet Management</h2>
        <WalletManager userId={profile?.id || ""} />
      </div>
    </main>
  );
}
