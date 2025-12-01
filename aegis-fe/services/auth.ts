"use client";

// Client-side auth utilities for MetaMask-based Web3 login
// Detailed comments included to assist future extension

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
const TOKEN_KEY = "aegis.jwt";

export type UserProfile = {
  id: string;
  wallet_address: string;
  balance: number;
  nonce: string;
  role: string;
  created_at?: string;
  updated_at?: string;
  last_login?: string | null;
};

export async function requestNonce(wallet: string): Promise<string> {
  const res = await fetch(`${API_BASE_URL}/auth/nonce`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ wallet }),
  });
  if (!res.ok) throw new Error(`Nonce request failed: ${res.status}`);
  const data = await res.json();
  return data.nonce as string;
}

export async function signNonce(nonce: string, wallet: string): Promise<string> {
  if (!(window as any).ethereum) throw new Error("MetaMask not found");
  const ethereum = (window as any).ethereum;
  // personal_sign expects params: [message, address]
  const signature: string = await ethereum.request({
    method: "personal_sign",
    params: [nonce, wallet],
  });
  return signature;
}

export async function verifySignature(wallet: string, signature: string): Promise<string> {
  const res = await fetch(`${API_BASE_URL}/auth/verify`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ wallet, signature }),
  });
  if (!res.ok) throw new Error(`Verify failed: ${res.status}`);
  const data = await res.json();
  const token = data.token as string;
  localStorage.setItem(TOKEN_KEY, token);
  return token;
}

export function getStoredToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY);
  } catch {
    return null;
  }
}

export function clearStoredToken(): void {
  try {
    localStorage.removeItem(TOKEN_KEY);
  } catch {}
}

export async function loadProfile(): Promise<UserProfile | null> {
  const token = getStoredToken();
  if (!token) return null;
  const res = await fetch(`${API_BASE_URL}/auth/me`, {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) return null;
  const data = await res.json();
  // Response shape: { user: { ... } }
  const user = (data.user || null) as any;
  if (!user) return null;
  const normalize = (ts: any): string | null => {
    if (!ts) return null;
    if (typeof ts === "string") return ts;
    if (typeof ts === "object" && typeof ts.seconds === "number") {
      const ms = ts.seconds * 1000 + (ts.nanos ? ts.nanos / 1e6 : 0);
      return new Date(ms).toISOString();
    }
    return null;
  };
  return {
    id: user.id,
    wallet_address: user.wallet_address,
    balance: user.balance,
    nonce: user.nonce,
    role: user.role,
    created_at: normalize(user.created_at) || undefined,
    updated_at: normalize(user.updated_at) || undefined,
    last_login: normalize(user.last_login),
  };
}

export async function logout(): Promise<void> {
  clearStoredToken();
}

export async function startWeb3Login(): Promise<{ token: string; profile: UserProfile | null; wallet: string }>
{
  if (!(window as any).ethereum) throw new Error("MetaMask not found");
  const ethereum = (window as any).ethereum;
  // 1) Request accounts
  const accounts: string[] = await ethereum.request({ method: "eth_requestAccounts" });
  const wallet = accounts[0];

  // 2) Request nonce
  const nonce = await requestNonce(wallet);

  // 3) Sign nonce
  const signature = await signNonce(nonce, wallet);

  // 4) Verify signature → JWT
  const token = await verifySignature(wallet, signature);

  // 5) Load profile
  const profile = await loadProfile();
  return { token, profile, wallet };
}

export async function devLogin(wallet?: string): Promise<{ token: string; profile: UserProfile | null; wallet: string }>
{
  const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
  // Prefer /api/auth/dev-login, fallback to /auth/dev-login
  const tryPaths = ["/api/auth/dev-login", "/auth/dev-login"];
  let lastErr: any = null;
  for (const p of tryPaths) {
    try {
      const res = await fetch(`${API_BASE_URL}${p}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ wallet }),
      });
      if (!res.ok) {
        lastErr = new Error(`Dev login failed: ${res.status}`);
        continue;
      }
      const data = await res.json();
      const token = data.token as string;
      localStorage.setItem("aegis.jwt", token);
      const profile = await loadProfile();
      return { token, profile, wallet: wallet || "0xTESTUSER" };
    } catch (e) {
      lastErr = e;
    }
  }
  throw (lastErr || new Error("Dev login failed"));
}
