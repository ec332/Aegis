import type { Metadata } from "next";
import "./globals.css";
import Navbar from "@/components/Navbar";
import ErrorBoundary from "@/components/ErrorBoundary";
import AppInitializer from "@/components/AppInitializer";

export const metadata: Metadata = {
  title: "Aegis",
  description: "Aegis Application",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="bg-white text-black m-0 p-0">
        <ErrorBoundary>
          <AppInitializer />
          <Navbar/>
          {children}
        </ErrorBoundary>
      </body>
    </html>
  );
}
