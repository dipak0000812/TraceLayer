import type { Metadata } from "next";
import "./globals.css";
import { Sidebar } from "@/components/layout/Sidebar";
import { CommandPalette } from "@/components/layout/CommandPalette";
import { fetchHealth } from "@/lib/api";

export const metadata: Metadata = {
  title: "TraceLayer — Evidence Intelligence",
  description: "Offline forensic evidence-correlation workstation for Bitcoin transaction and network evidence.",
};

export default async function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const health = await fetchHealth().catch(() => null);
  const apiStatus: "UP" | "DOWN" | "UNKNOWN" = health
    ? health.status.toUpperCase() === "HEALTHY" || health.status.toUpperCase() === "UP"
      ? "UP"
      : "DOWN"
    : "DOWN";

  return (
    <html lang="en">
      <body className="flex h-screen overflow-hidden bg-bg font-sans text-text antialiased">
        <Sidebar apiStatus={apiStatus} />
        <div className="flex h-full flex-1 flex-col overflow-hidden">
          <div className="flex-1 overflow-y-auto">
            <div className="mx-auto max-w-[1400px] px-8 py-7">{children}</div>
          </div>
        </div>
        <CommandPalette />
      </body>
    </html>
  );
}
