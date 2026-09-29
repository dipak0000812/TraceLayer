"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Activity,
  Database,
  FileSearch,
  GitMerge,
  LayoutGrid,
  ListChecks,
  Radio,
  Upload,
} from "lucide-react";
import { DATASET_METADATA } from "@/lib/api";
import { StatusDot } from "@/components/ui/Badge";

const NAV = [
  {
    group: "Workspace",
    items: [
      { href: "/", label: "Overview", icon: LayoutGrid },
      { href: "/leads", label: "Leads", icon: ListChecks },
      { href: "/evidence", label: "Evidence", icon: FileSearch },
    ],
  },
  {
    group: "Pipeline",
    items: [
      { href: "/ingestion", label: "Ingestion", icon: Upload },
      { href: "/correlation", label: "Correlation", icon: GitMerge },
    ],
  },
  {
    group: "System",
    items: [{ href: "/health", label: "Health", icon: Activity }],
  },
];

export function Sidebar({ apiStatus }: { apiStatus: "UP" | "DOWN" | "UNKNOWN" }) {
  const pathname = usePathname();

  return (
    <aside className="flex h-full w-60 shrink-0 flex-col border-r border-border bg-bg-raised">
      <div className="flex h-14 items-center gap-2 border-b border-border px-4">
        <div className="flex h-6 w-6 items-center justify-center rounded bg-accent-muted text-accent-strong">
          <Radio className="h-3.5 w-3.5" />
        </div>
        <div className="leading-tight">
          <div className="text-[13px] font-semibold text-text">TraceLayer</div>
          <div className="text-2xs text-text-muted">Evidence intelligence</div>
        </div>
      </div>

      <nav className="flex-1 overflow-y-auto px-3 py-4">
        {NAV.map((section) => (
          <div key={section.group} className="mb-5">
            <div className="px-2 pb-1.5 text-2xs font-medium uppercase tracking-wider text-text-muted">
              {section.group}
            </div>
            <div className="space-y-0.5">
              {section.items.map((item) => {
                const active =
                  item.href === "/" ? pathname === "/" : pathname.startsWith(item.href);
                const Icon = item.icon;
                return (
                  <Link
                    key={item.href}
                    href={item.href}
                    className={`focus-ring flex items-center gap-2.5 rounded-md px-2.5 py-1.5 text-sm transition-colors ${
                      active
                        ? "bg-surface-active text-text"
                        : "text-text-secondary hover:bg-surface hover:text-text"
                    }`}
                  >
                    <Icon className="h-4 w-4 shrink-0" strokeWidth={1.75} />
                    {item.label}
                  </Link>
                );
              })}
            </div>
          </div>
        ))}
      </nav>

      <div className="border-t border-border px-4 py-3">
        <div className="flex items-center gap-2 text-xs text-text-secondary">
          <Database className="h-3.5 w-3.5 text-text-muted" />
          <span className="mono-id">{DATASET_METADATA.seed}</span>
          <span className="ml-auto rounded border border-border-strong px-1 py-0.5 text-2xs text-text-muted">
            {DATASET_METADATA.provenance}
          </span>
        </div>
        <div className="mt-2 flex items-center gap-1.5 text-2xs text-text-muted">
          <StatusDot tone={apiStatus === "UP" ? "success" : apiStatus === "DOWN" ? "danger" : "neutral"} />
          Offline intelligence {apiStatus === "UP" ? "ready" : apiStatus === "DOWN" ? "unavailable" : "unknown"}
        </div>
      </div>
    </aside>
  );
}
