"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Search } from "lucide-react";
import { TXID_PATTERN } from "@/lib/api";

const ACTIONS = [
  { label: "Overview", href: "/" },
  { label: "Investigative leads", href: "/leads" },
  { label: "Evidence explorer", href: "/evidence" },
  { label: "Ingestion", href: "/ingestion" },
  { label: "Correlation", href: "/correlation" },
  { label: "System health", href: "/health" },
];

export function CommandPalette() {
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const router = useRouter();

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      const isMeta = event.metaKey || event.ctrlKey;
      const targetTag = (event.target as HTMLElement | null)?.tagName;
      const inField = targetTag === "INPUT" || targetTag === "TEXTAREA";

      if (isMeta && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setOpen(true);
      } else if (event.key === "/" && !inField) {
        event.preventDefault();
        setOpen(true);
      } else if (event.key === "Escape") {
        setOpen(false);
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  useEffect(() => {
    if (open) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setQuery("");
      requestAnimationFrame(() => inputRef.current?.focus());
    }
  }, [open]);

  if (!open) return null;

  const trimmed = query.trim();
  const isTxid = TXID_PATTERN.test(trimmed);
  const filtered = ACTIONS.filter((a) =>
    a.label.toLowerCase().includes(trimmed.toLowerCase()),
  );

  function go(href: string) {
    setOpen(false);
    router.push(href);
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-black/60 pt-[15vh]"
      onClick={() => setOpen(false)}
    >
      <div
        className="w-full max-w-lg overflow-hidden rounded-lg border border-border-strong bg-bg-raised shadow-2xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-2 border-b border-border px-3.5 py-2.5">
          <Search className="h-4 w-4 text-text-muted" />
          <input
            ref={inputRef}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter" && isTxid) go(`/evidence?txid=${trimmed}`);
              else if (e.key === "Enter" && filtered[0]) go(filtered[0].href);
            }}
            placeholder="Search TXID, or jump to a page…"
            className="w-full bg-transparent text-sm text-text placeholder:text-text-muted focus:outline-none"
          />
          <kbd className="rounded border border-border-strong px-1 py-0.5 text-2xs text-text-muted">
            Esc
          </kbd>
        </div>
        <div className="max-h-72 overflow-y-auto py-1.5">
          {isTxid && (
            <button
              onClick={() => go(`/evidence?txid=${trimmed}`)}
              className="focus-ring flex w-full items-center gap-2 px-3.5 py-2 text-left text-sm text-text hover:bg-surface-hover"
            >
              Open evidence for <span className="mono-id text-text-secondary">{trimmed}</span>
            </button>
          )}
          {filtered.map((action) => (
            <button
              key={action.href}
              onClick={() => go(action.href)}
              className="focus-ring flex w-full items-center px-3.5 py-2 text-left text-sm text-text-secondary hover:bg-surface-hover hover:text-text"
            >
              {action.label}
            </button>
          ))}
          {!isTxid && filtered.length === 0 && (
            <div className="px-3.5 py-4 text-sm text-text-muted">No matches.</div>
          )}
        </div>
      </div>
    </div>
  );
}
