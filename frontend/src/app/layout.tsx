import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { Activity, Database, Server, Shield } from "lucide-react";
import Link from "next/link";
import { fetchHealth } from "@/lib/api";

const inter = Inter({ subsets: ["latin"] });

export const metadata: Metadata = {
  title: "TraceLayer Forensic Dashboard",
  description: "Offline forensic evidence-correlation prototype",
};

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  // Server-side fetch for health (if down, we can still render)
  const health = await fetchHealth().catch(() => ({ status: "DOWN", timestamp: "", services: { postgres: "DOWN" } }));

  return (
    <html lang="en" className="dark">
      <body className={`${inter.className} bg-background text-foreground flex h-screen overflow-hidden`}>
        
        {/* Sidebar */}
        <aside className="w-64 border-r border-border bg-card flex flex-col">
          <div className="h-16 flex items-center px-6 border-b border-border">
            <Shield className="w-6 h-6 text-primary mr-2" />
            <span className="font-bold text-lg tracking-tight">TraceLayer</span>
          </div>
          
          <nav className="flex-1 p-4 space-y-2">
            <Link href="/" className="flex items-center px-4 py-2 text-sm rounded-md bg-accent text-accent-foreground font-medium">
              <Activity className="w-4 h-4 mr-3" />
              Operations & Leads
            </Link>
          </nav>

          <div className="p-4 border-t border-border text-xs text-muted-foreground">
            <div className="mb-2 font-medium">System Health</div>
            
            <div className="flex items-center justify-between mb-1">
              <span className="flex items-center"><Server className="w-3 h-3 mr-1" /> API</span>
              <span className={`w-2 h-2 rounded-full ${health.status === 'HEALTHY' ? 'bg-primary' : 'bg-destructive'}`}></span>
            </div>
            
            <div className="flex items-center justify-between mb-1">
              <span className="flex items-center"><Database className="w-3 h-3 mr-1" /> Postgres</span>
              <span className={`w-2 h-2 rounded-full ${health.services?.postgres === 'UP' ? 'bg-primary' : 'bg-destructive'}`}></span>
            </div>
            
            <div className="flex items-center justify-between">
              <span className="flex items-center"><Activity className="w-3 h-3 mr-1" /> Intelligence</span>
              <span className="text-muted-foreground">External worker</span>
            </div>
          </div>
        </aside>

        {/* Main Content */}
        <main className="flex-1 flex flex-col h-full overflow-hidden">
          <header className="h-16 border-b border-border flex items-center justify-between px-8 bg-background">
            <h1 className="text-sm font-medium text-muted-foreground">Forensic Evidence-Correlation Terminal</h1>
            <div className="flex items-center space-x-2 text-xs">
              <span className="px-2 py-1 rounded bg-muted text-muted-foreground border border-border">Offline Mode</span>
              <span className="px-2 py-1 rounded bg-muted text-muted-foreground border border-border">Dataset: seed-42</span>
            </div>
          </header>
          
          <div className="flex-1 overflow-y-auto p-8 bg-background">
            {children}
          </div>
        </main>

      </body>
    </html>
  );
}
