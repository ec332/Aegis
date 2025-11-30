"use client";

import { create } from "zustand";
import { startWeb3Login, loadProfile, logout as apiLogout, getStoredToken, clearStoredToken, UserProfile } from "@/services/auth";

interface AuthState {
  isAuthenticated: boolean;
  wallet: string | null;
  token: string | null;
  profile: UserProfile | null;
  isLoading: boolean;
  error: string | null;
  initAuth: () => Promise<void>;
  signIn: () => Promise<void>;
  signOut: () => Promise<void>;
}

// Global auth store powered by Zustand.
// Handles initial token load, MetaMask sign-in, and logout.
export const useAuthStore = create<AuthState>((set) => ({
  isAuthenticated: false,
  wallet: null,
  token: null,
  profile: null,
  isLoading: false,
  error: null,

  // On app load: read token and fetch profile
  initAuth: async () => {
    set({ isLoading: true, error: null });
    try {
      const token = getStoredToken();
      if (!token) {
        set({ isAuthenticated: false, token: null, wallet: null, profile: null, isLoading: false });
        return;
      }
      const profile = await loadProfile();
      if (profile) {
        set({ isAuthenticated: true, token, wallet: profile.wallet_address, profile, isLoading: false });
      } else {
        clearStoredToken();
        set({ isAuthenticated: false, token: null, wallet: null, profile: null, isLoading: false });
      }
    } catch (e: any) {
      set({ error: e?.message || "Auth init failed", isLoading: false });
    }
  },

  // Full MetaMask authentication
  signIn: async () => {
    set({ isLoading: true, error: null });
    try {
      const { token, profile, wallet } = await startWeb3Login();
      set({ isAuthenticated: !!token, token, wallet, profile, isLoading: false });
    } catch (e: any) {
      set({ error: e?.message || "Sign-in failed", isLoading: false });
    }
  },

  // Logout clears token and state
  signOut: async () => {
    set({ isLoading: true, error: null });
    try {
      await apiLogout();
      set({ isAuthenticated: false, token: null, wallet: null, profile: null, isLoading: false });
    } catch (e: any) {
      set({ error: e?.message || "Logout failed", isLoading: false });
    }
  }
}));
