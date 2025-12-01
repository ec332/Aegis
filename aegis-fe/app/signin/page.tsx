"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/store/authStore";
import { devLogin } from "@/services/auth";

export default function SignInPage() {
  const { isAuthenticated, isLoading, error, signIn } = useAuthStore();
  const router = useRouter();

  useEffect(() => {
    if (isAuthenticated) {
      router.replace("/?tab=wallet");
    }
  }, [isAuthenticated, router]);

  const handleConnect = async () => {
    try {
      const { token, profile, wallet } = await devLogin("0xTESTUSER");
      useAuthStore.setState({ isAuthenticated: !!token, token, wallet, profile, isLoading: false, error: null });
      router.replace("/?tab=wallet");
    } catch (e: any) {
      useAuthStore.setState({ error: e?.message || "Dev login failed", isLoading: false });
    }
  };

  return (
    <main className="bg-white min-h-screen">
      <div className="px-4 sm:px-6 lg:px-8 py-20 max-w-3xl mx-auto">
        <h1 className="text-3xl font-bold text-[#151b4d] mb-6">Sign In</h1>
        <p className="text-gray-600 mb-8">Connect your wallet to sign in securely.</p>

        <div className="rounded-2xl border border-gray-200 p-6 shadow-sm">
          <div className="mb-4">
            <h2 className="text-xl font-semibold text-[#151b4d]">MetaMask</h2>
            <p className="text-gray-600">We use a signed message to authenticate you without passwords.</p>
          </div>

          <button
            type="button"
            onClick={handleConnect}
            className="rounded bg-blue-600 px-4 py-2 text-white shadow hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            {isLoading ? "Connecting..." : "Connect Wallet"}
          </button>

          {error && (
            <div className="mt-3 text-sm text-red-600">{error}</div>
          )}
        </div>

        <div className="mt-8 text-sm text-gray-500">
          Don’t have MetaMask?
          <button
            type="button"
            onClick={() => window.open("https://metamask.io/download/", "_blank")}
            className="ml-2 text-blue-600 hover:underline"
          >
            Install MetaMask
          </button>
        </div>
      </div>
    </main>
  );
}
