"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Search } from "lucide-react";
import { fetchEvidence, ApiError, TXID_PATTERN, type EvidenceResponse } from "@/lib/api";
import { PageHeader } from "@/components/layout/PageHeader";
import { Panel, PanelHeader, ErrorState, EmptyState, Skeleton } from "@/components/ui/States";
import { HashValue } from "@/components/ui/HashValue";
import { formatTimestamp } from "@/lib/format";

function EvidenceExplorerInner() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const initialTxid = searchParams.get("txid") ?? "";

  const [input, setInput] = useState(initialTxid);
  const [evidence, setEvidence] = useState<EvidenceResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [history, setHistory] = useState<string[]>([]);

  async function lookup(txid: string) {
    if (!txid) return;
    setLoading(true);
    setError(null);
    setEvidence(null);
    try {
      const result = await fetchEvidence(txid);
      setEvidence(result);
      setHistory((prev) => [txid, ...prev.filter((t) => t !== txid)].slice(0, 8));
      router.replace(`/evidence?txid=${txid}`, { scroll: false });
    } catch (cause) {
      setError(cause instanceof Error ? cause : new Error("Failed to load evidence"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    if (initialTxid) void lookup(initialTxid);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const isValid = TXID_PATTERN.test(input.trim());

  return (
    <div>
      <PageHeader
        title="Evidence explorer"
        subtitle="Look up correlated blockchain and network evidence by transaction ID."
      />

      <div className="mb-6 flex items-center gap-2">
        <div className="flex flex-1 items-center gap-2 rounded-md border border-border bg-surface px-3 py-2">
          <Search className="h-4 w-4 text-text-muted" />
          <input
            value={input}
            onChange={(e) => setInput(e.target.value.trim())}
            onKeyDown={(e) => e.key === "Enter" && isValid && void lookup(input)}
            placeholder="64-character transaction ID (TXID)…"
            className="mono-id w-full bg-transparent text-sm text-text placeholder:text-text-muted placeholder:font-sans focus:outline-none"
          />
        </div>
        <button
          onClick={() => void lookup(input)}
          disabled={!isValid || loading}
          className="focus-ring rounded-md bg-accent px-4 py-2 text-sm font-medium text-white disabled:opacity-40"
        >
          Look up
        </button>
      </div>

      {history.length > 0 && (
        <div className="mb-6 flex flex-wrap items-center gap-1.5 text-xs">
          <span className="text-text-muted">Recent:</span>
          {history.map((txid) => (
            <button
              key={txid}
              onClick={() => {
                setInput(txid);
                void lookup(txid);
              }}
              className="focus-ring mono-id rounded border border-border bg-surface px-2 py-0.5 text-text-secondary hover:bg-surface-hover"
            >
              {txid.slice(0, 10)}…
            </button>
          ))}
        </div>
      )}

      {loading && (
        <Panel>
          <div className="space-y-2 p-5">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-6 w-full" />
            ))}
          </div>
        </Panel>
      )}

      {!loading && error && (
        <Panel>
          <div className="p-5">
            <ErrorState
              title="Evidence not found"
              message={error.message}
              detail={error instanceof ApiError ? `${error.code ?? "ERROR"} · status ${error.status}` : undefined}
              onRetry={() => void lookup(input)}
            />
          </div>
        </Panel>
      )}

      {!loading && !error && !evidence && (
        <Panel>
          <EmptyState
            title="No evidence loaded"
            description="Search a transaction ID to view its correlated blockchain and network evidence. A full browsable evidence index isn't exposed by the current API — lookups are by TXID."
          />
        </Panel>
      )}

      {!loading && evidence && (
        <div className="space-y-6">
          <Panel>
            <PanelHeader title="Transaction" subtitle={evidence.transaction.txid} />
            <div className="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm md:grid-cols-2">
              <div>
                <div className="text-2xs uppercase tracking-wide text-text-muted">TXID</div>
                <HashValue value={evidence.transaction.txid} truncate={false} className="mt-1" />
              </div>
              <div>
                <div className="text-2xs uppercase tracking-wide text-text-muted">Timestamp</div>
                <div className="mt-1 text-text-secondary">{formatTimestamp(evidence.transaction.timestamp)}</div>
              </div>
              <div>
                <div className="text-2xs uppercase tracking-wide text-text-muted">Fee</div>
                <div className="mt-1 font-mono-tabular text-text-secondary">{evidence.transaction.fee} BTC</div>
              </div>
              <div>
                <div className="text-2xs uppercase tracking-wide text-text-muted">Script type</div>
                <div className="mt-1 mono-id text-text-secondary">{evidence.transaction.script_type}</div>
              </div>
            </div>
          </Panel>

          <Panel>
            <PanelHeader
              title={`Network observations (${evidence.network_observations.length})`}
              subtitle="Relay observations — not attributed ownership"
            />
            {evidence.network_observations.length > 0 ? (
              <div className="overflow-x-auto">
                <table className="w-full border-collapse text-sm">
                  <thead>
                    <tr className="border-b border-border text-left text-2xs uppercase tracking-wide text-text-muted">
                      <th className="px-5 py-2.5 font-medium">Observed at</th>
                      <th className="px-5 py-2.5 font-medium">Observed peer</th>
                      <th className="px-5 py-2.5 font-medium">Port</th>
                    </tr>
                  </thead>
                  <tbody>
                    {evidence.network_observations.map((obs) => (
                      <tr key={obs.observation_id} className="table-row-hover border-b border-border last:border-b-0">
                        <td className="px-5 py-2.5 text-text-secondary">{formatTimestamp(obs.observed_at)}</td>
                        <td className="px-5 py-2.5 mono-id text-text-secondary">{obs.first_heard_peer_ip}</td>
                        <td className="px-5 py-2.5 font-mono-tabular text-text-secondary">{obs.src_port}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <div className="px-5 py-6 text-sm text-text-muted">No network observations correlated to this transaction.</div>
            )}
            <div className="border-t border-border px-5 py-3 text-2xs text-text-muted">{evidence.disclaimer}</div>
          </Panel>
        </div>
      )}
    </div>
  );
}

export default function EvidenceExplorerPage() {
  return (
    <Suspense fallback={null}>
      <EvidenceExplorerInner />
    </Suspense>
  );
}
