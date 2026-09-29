"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { ArrowUpDown, Search } from "lucide-react";
import { fetchLeads, ApiError, type Lead, type LeadSortBy } from "@/lib/api";
import { PageHeader } from "@/components/layout/PageHeader";
import { Panel, EmptyState, ErrorState, Skeleton } from "@/components/ui/States";
import { Badge } from "@/components/ui/Badge";
import { HashValue } from "@/components/ui/HashValue";
import { formatRankShift, formatScore } from "@/lib/format";

const PAGE_SIZE = 50;

export default function LeadsPage() {
  const [leads, setLeads] = useState<Lead[] | null>(null);
  const [totalLeads, setTotalLeads] = useState(0);
  const [sortBy, setSortBy] = useState<LeadSortBy>("fused_rank");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<ApiError | Error | null>(null);

  const [query, setQuery] = useState("");
  const [minFused, setMinFused] = useState("");
  const [flaggedOnly, setFlaggedOnly] = useState(false);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const result = await fetchLeads(1, PAGE_SIZE, sortBy);
      setLeads(result.leads);
      setTotalLeads(result.total_leads);
    } catch (cause) {
      setError(cause instanceof Error ? cause : new Error("Failed to load leads"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sortBy]);

  const filtered = useMemo(() => {
    if (!leads) return [];
    return leads.filter((lead) => {
      if (query && !lead.entity_id.includes(query) && !lead.primary_txid.includes(query)) return false;
      if (minFused && lead.fused_score < parseFloat(minFused)) return false;
      if (flaggedOnly && lead.anomaly_flags.length === 0) return false;
      return true;
    });
  }, [leads, query, minFused, flaggedOnly]);

  return (
    <div>
      <PageHeader
        title="Investigative leads"
        subtitle="Ranked evidence combinations requiring analyst review."
      />

      <div className="mb-4 flex flex-wrap items-center gap-2">
        <div className="flex min-w-[220px] flex-1 items-center gap-2 rounded-md border border-border bg-surface px-3 py-1.5">
          <Search className="h-3.5 w-3.5 text-text-muted" />
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Filter by entity or TXID…"
            className="w-full bg-transparent text-sm text-text placeholder:text-text-muted focus:outline-none"
          />
        </div>
        <input
          value={minFused}
          onChange={(e) => setMinFused(e.target.value)}
          type="number"
          step="0.05"
          min={0}
          max={1}
          placeholder="Min fused score"
          className="w-36 rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-text placeholder:text-text-muted focus:outline-none"
        />
        <label className="flex items-center gap-1.5 rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-text-secondary">
          <input
            type="checkbox"
            checked={flaggedOnly}
            onChange={(e) => setFlaggedOnly(e.target.checked)}
            className="accent-accent"
          />
          Flagged only
        </label>
        <button
          onClick={() => setSortBy(sortBy === "fused_rank" ? "rank_shift" : "fused_rank")}
          className="focus-ring flex items-center gap-1.5 rounded-md border border-border bg-surface px-3 py-1.5 text-sm text-text-secondary hover:bg-surface-hover"
        >
          <ArrowUpDown className="h-3.5 w-3.5" />
          Sort: {sortBy === "fused_rank" ? "Fused rank" : "Rank shift"}
        </button>
        <span className="ml-auto text-xs text-text-muted">
          {leads ? `${filtered.length} shown of ${totalLeads} total` : ""}
        </span>
      </div>

      <Panel>
        {loading ? (
          <div className="space-y-2 p-5">
            {Array.from({ length: 8 }).map((_, i) => (
              <Skeleton key={i} className="h-8 w-full" />
            ))}
          </div>
        ) : error ? (
          <div className="p-5">
            <ErrorState
              title="Unable to load leads"
              message={error.message}
              detail={error instanceof ApiError ? `${error.code ?? "ERROR"} · status ${error.status}` : undefined}
              onRetry={() => void load()}
            />
          </div>
        ) : filtered.length === 0 ? (
          <EmptyState
            title={leads && leads.length > 0 ? "No leads match these filters" : "No investigative leads yet"}
            description={
              leads && leads.length > 0
                ? "Clear or widen your filters to see more of the ranked queue."
                : "Ingest evidence and run correlation to generate ranked leads."
            }
            actionHref={leads && leads.length > 0 ? undefined : "/ingestion"}
            actionLabel={leads && leads.length > 0 ? undefined : "Go to ingestion"}
          />
        ) : (
          <LeadsTable leads={filtered} />
        )}
      </Panel>
    </div>
  );
}

function LeadsTable({ leads }: { leads: Lead[] }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full border-collapse text-sm">
        <thead className="sticky top-0 bg-surface">
          <tr className="border-b border-border text-left text-2xs uppercase tracking-wide text-text-muted">
            <th className="px-5 py-2.5 font-medium">#</th>
            <th className="px-5 py-2.5 font-medium">Entity</th>
            <th className="px-5 py-2.5 font-medium">Primary TXID</th>
            <th className="px-5 py-2.5 font-medium">Chain</th>
            <th className="px-5 py-2.5 font-medium">Network</th>
            <th className="px-5 py-2.5 font-medium">Fused</th>
            <th className="px-5 py-2.5 font-medium">Δ Rank</th>
            <th className="px-5 py-2.5 font-medium">Quality</th>
            <th className="px-5 py-2.5 font-medium">Flags</th>
          </tr>
        </thead>
        <tbody>
          {leads.map((lead) => {
            const shift = formatRankShift(lead.rank_shift);
            return (
              <tr key={lead.lead_id} className="table-row-hover border-b border-border last:border-b-0">
                <td className="px-5 py-2.5">
                  <Link
                    href={`/leads/${encodeURIComponent(lead.lead_id)}`}
                    className="focus-ring font-mono-tabular text-text hover:text-accent-strong"
                  >
                    #{String(lead.fused_rank).padStart(2, "0")}
                  </Link>
                </td>
                <td className="px-5 py-2.5 mono-id text-text-secondary">{lead.entity_id}</td>
                <td className="px-5 py-2.5"><HashValue value={lead.primary_txid} /></td>
                <td className="px-5 py-2.5 font-mono-tabular text-text-secondary">{formatScore(lead.chain_only_score)}</td>
                <td className="px-5 py-2.5 font-mono-tabular text-text-secondary">{formatScore(lead.network_score)}</td>
                <td className="px-5 py-2.5 font-mono-tabular font-medium text-text">{formatScore(lead.fused_score)}</td>
                <td className="px-5 py-2.5">
                  <span
                    className={`font-mono-tabular text-xs ${
                      shift.tone === "up" ? "text-danger" : shift.tone === "down" ? "text-success" : "text-text-muted"
                    }`}
                  >
                    {shift.label}
                  </span>
                </td>
                <td className="px-5 py-2.5 font-mono-tabular text-text-secondary">
                  {formatScore(lead.network_evidence_quality)}
                </td>
                <td className="px-5 py-2.5">
                  <div className="flex flex-wrap gap-1">
                    {lead.anomaly_flags.length === 0 ? (
                      <span className="text-xs text-text-muted">—</span>
                    ) : (
                      lead.anomaly_flags.map((flag) => (
                        <Badge key={flag} tone="warning">{flag.replace(/_/g, " ")}</Badge>
                      ))
                    )}
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
