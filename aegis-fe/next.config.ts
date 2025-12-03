import type { NextConfig } from "next";
import path from "path";

const nextConfig: NextConfig = {
  turbopack: {
    root: path.resolve(__dirname, ".."),
  },
  async rewrites() {
    const base = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
    return [
      {
        source: "/api/:path*",
        destination: `${base}/api/:path*`,
      },
      {
        source: "/health",
        destination: `${base}/health`,
      },
    ];
  },
};

export default nextConfig;
