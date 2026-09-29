import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { fetchHealth, fetchLeads, DATASET_METADATA, type Lead } from "@/lib/api";
import { PageHeader } from "@/components/layout/PageHeader";
import { Panel, PanelHeader, EmptyState } from "@/components/ui/States";
import { Badge, StatusDot } from "@/components/ui/Badge";
import { HashValue } from "@/components/ui/HashValue";
import { formatRankShift, formatScore } from "@/lib/format";

async function loadOverviewData() {
  const [leadsResult, healthResult] = await Promise.allSettled([
    fetchLeads(1, 8, "fused_rank"),
    fetchHealth(),
  ]);

  return {
    leads: leadsResult.status === "fulfilled" ? leadsResult.value : null,
    leadsError: leadsResult.status === "rejected" ? leadsResult.reason : null,
    health: healthResult.status === "fulfilled" ? healthResult.value : null,
  };
}

function StatCell({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div className="flex-1 border-r border-border px-5 py-4 last:border-r-0">
      <div className="text-2xs uppercase tracking-wide text-text-muted">{label}</div>
      <div className="mt-1 font-mono-tabular text-xl font-semibold text-text">{value}</div>
      {sub && <div className="mt-0.5 text-2xs text-text-muted">{sub}</div>}
    </div>
  );
}

export default async function OverviewPage() {
  const { leads, health } = await loadOverviewData();

  const sampleEntities = leads ? new Set(leads.leads.map((l) => l.entity_id)).size : 0;
  const topScore = leads?.leads.reduce((m, l) => Math.max(m, l.fused_score), 0) ?? 0;
  const avgQuality =
    leads && leads.leads.length
      ? leads.leads.reduce((s, l) => s + l.network_evidence_quality, 0) / leads.leads.length
      : null;

  return (
    <div>
      <PageHeader
        title="Investigation workspace"
        subtitle={`Benchmark (${DATASET_METADATA.seed}) · ${DATASET_METADATA.provenance.toLowerCase()} · generator v${DATASET_METADATA.generatorVersion}`}
      />

      <Panel className="mb-6">
        <div className="flex flex-wrap">
          <StatCell label="Investigative leads" value={leads ? String(leads.total_leads) : "—"} />
          <StatCell
            label="Entities (top ranked)"
            value={leads ? String(sampleEntities) : "—"}
            sub="unique entities in current queue"
          />
          <StatCell
            label="Highest fused priority"
            value={leads ? formatScore(topScore) : "—"}
          />
          <StatCell
            label="Avg. network quality"
            value={avgQuality !== null ? formatScore(avgQuality) : "—"}
            sub="across top-ranked leads"
          />
          <StatCell
            label="Intelligence status"
            value={health ? "Available" : "Unavailable"}
          />
        </div>
      </Panel>

      <Panel className="mb-6">
        <PanelHeader
          title="Investigation queue"
          subtitle="Highest-priority fused leads requiring analyst review"
          action={
            <Link
              href="/leads"
              className="focus-ring flex items-center gap-1 text-xs font-medium text-accent-strong hover:underline"
            >
              View all leads <ArrowRight className="h-3 w-3" />
            </Link>
          }
        />
        {leads && leads.leads.length > 0 ? (
          <QueueTable leads={leads.leads} />
        ) : leads ? (
          <EmptyState
            title="No investigative leads yet"
            description="Ingest blockchain and network evidence, then run correlation to generate ranked leads."
            actionHref="/ingestion"
            actionLabel="Go to ingestion"
          />
        ) : (
          <EmptyState
            title="Leads unavailable"
            description="The TraceLayer API did not return lead data. Confirm the backend services are running."
            actionHref="/health"
            actionLabel="Check system health"
          />
        )}
      </Panel>

      <Panel>
        <PanelHeader title="Pipeline status" subtitle="Reported service state from the API health check" />
        {health ? (
          <div className="divide-y divide-border">
            <PipelineRow label="API" state="UP" />
            {Object.entries(health.services).map(([name, state]) => (
              <PipelineRow key={name} label={name} state={state} />
            ))}
          </div>
        ) : (
          <EmptyState
            title="TraceLayer API unavailable"
            description="Start the backend services and retry to see live pipeline status."
            actionHref="/health"
            actionLabel="Open system health"
          />
        )}
      </Panel>
    </div>
  );
}

function QueueTable({ leads }: { leads: Lead[] }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full border-collapse text-sm">
        <thead>
          <tr className="border-b border-border text-left text-2xs uppercase tracking-wide text-text-muted">
            <th className="px-5 py-2.5 font-medium">Rank</th>
            <th className="px-5 py-2.5 font-medium">Entity</th>
            <th className="px-5 py-2.5 font-medium">Primary TXID</th>
            <th className="px-5 py-2.5 font-medium">Chain</th>
            <th className="px-5 py-2.5 font-medium">Network</th>
            <th className="px-5 py-2.5 font-medium">Fused</th>
            <th className="px-5 py-2.5 font-medium">Δ Rank</th>
            <th className="px-5 py-2.5 font-medium">Flags</th>
          </tr>
        </thead>
        <tbody>
          {leads.map((lead) => {
            const shift = formatRankShift(lead.rank_shift);
            return (
              <tr key={lead.lead_id} className="table-row-hover border-b border-border last:border-b-0">
                <td className="px-5 py-2.5">
                  <Link href={`/leads/${encodeURIComponent(lead.lead_id)}`} className="focus-ring font-mono-tabular text-text hover:text-accent-strong">
                    #{lead.fused_rank}
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
                <td className="px-5 py-2.5">
                  <div className="flex flex-wrap gap-1">
                    {lead.anomaly_flags.slice(0, 2).map((flag) => (
                      <Badge key={flag} tone="warning">{flag.replace(/_/g, " ")}</Badge>
                    ))}
                    {lead.anomaly_flags.length > 2 && (
                      <Badge tone="neutral">+{lead.anomaly_flags.length - 2}</Badge>
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

function PipelineRow({ label, state }: { label: string; state: string }) {
  const normalized = state.toUpperCase();
  const tone = normalized === "UP" || normalized === "HEALTHY" ? "success" : normalized === "DOWN" ? "danger" : "neutral";
  return (
    <div className="flex items-center justify-between px-5 py-2.5 text-sm">
      <span className="capitalize text-text-secondary">{label}</span>
      <span className="flex items-center gap-1.5">
        <StatusDot tone={tone} />
        <span className="font-mono-tabular text-xs text-text">{state}</span>
      </span>
    </div>
  );
}
