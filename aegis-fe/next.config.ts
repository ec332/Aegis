import type { NextConfig } from "next";
import path from "path";

const nextConfig: NextConfig = {
  turbopack: {
    // Set root to the monorepo root to silence additional lockfile warning
    // and allow resolving linked packages outside the app directory.
    root: path.resolve(__dirname, ".."),
  },
};

export default nextConfig;
