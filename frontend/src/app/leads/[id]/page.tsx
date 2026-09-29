"use client";

import { use, useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, Cpu, Layers, ShieldCheck } from "lucide-react";
import {
  fetchLead,
  fetchEvidence,
  ApiError,
  DATASET_METADATA,
  type Lead,
  type EvidenceResponse,
} from "@/lib/api";
import { Panel, PanelHeader, ErrorState, Skeleton } from "@/components/ui/States";
import { Badge } from "@/components/ui/Badge";
import { HashValue } from "@/components/ui/HashValue";
import { formatRankShift, formatScore, formatTimestamp, flagLabel } from "@/lib/format";

function SignalField({ label, value, sub }: { label: string; value: string; sub?: string }) {
  return (
    <div>
      <div className="text-2xs uppercase tracking-wide text-text-muted">{label}</div>
      <div className="mt-1 font-mono-tabular text-base font-medium text-text">{value}</div>
      {sub && <div className="text-2xs text-text-muted">{sub}</div>}
    </div>
  );
}

export default function LeadDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const [lead, setLead] = useState<Lead | null>(null);
  const [evidence, setEvidence] = useState<EvidenceResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      const loadedLead = await fetchLead(id);
      setLead(loadedLead);
      try {
        setEvidence(await fetchEvidence(loadedLead.primary_txid));
      } catch {
        setEvidence(null);
      }
    } catch (cause) {
      setError(cause instanceof Error ? cause : new Error("Failed to load lead"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  if (loading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (error || !lead) {
    return (
      <div>
        <BackLink />
        <ErrorState
          title="Unable to load this lead"
          message={error?.message ?? "Lead not found."}
          detail={error instanceof ApiError ? `${error.code ?? "ERROR"} · status ${error.status}` : undefined}
          onRetry={() => void load()}
        />
      </div>
    );
  }

  const shift = formatRankShift(lead.rank_shift);

  return (
    <div className="space-y-6">
      <BackLink />

      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="flex items-center gap-2 text-2xs uppercase tracking-wide text-text-muted">
            Lead #{String(lead.fused_rank).padStart(2, "0")}
          </div>
          <h1 className="mt-1 flex items-center gap-2 text-xl font-semibold text-text">
            <span className="mono-id">{lead.entity_id}</span>
          </h1>
          <div className="mt-1 flex items-center gap-1.5 text-sm text-text-secondary">
            Primary transaction <HashValue value={lead.primary_txid} />
          </div>
        </div>
        <div className="flex flex-wrap gap-1.5">
          {lead.anomaly_flags.map((flag) => (
            <Badge key={flag} tone="warning">{flagLabel(flag)}</Badge>
          ))}
          {lead.anomaly_flags.length === 0 && <Badge tone="neutral">No flags</Badge>}
        </div>
      </div>

      {/* Signal summary */}
      <Panel>
        <PanelHeader
          title="Signal summary"
          subtitle="Scores are investigative ranking values — not calibrated probabilities"
        />
        <div className="grid grid-cols-2 gap-6 px-5 py-4 md:grid-cols-5">
          <SignalField label="Chain signal" value={formatScore(lead.chain_only_score)} sub={`rank #${lead.chain_only_rank}`} />
          <SignalField label="Network signal" value={formatScore(lead.network_score)} />
          <SignalField label="Network quality" value={formatScore(lead.network_evidence_quality)} />
          <SignalField label="Fused priority" value={formatScore(lead.fused_score)} sub={`rank #${lead.fused_rank}`} />
          <SignalField
            label="Rank shift"
            value={shift.label}
            sub="chain-only \u2192 fused ranking"
          />
        </div>
        <div className="border-t border-border px-5 py-2.5 text-2xs text-text-muted">
          <strong>Limitation:</strong> The current chain signal is computed by an Isolation Forest trained on all 11
          features (7 blockchain + 4 network). It is not a pure chain-only anomaly score. Rank shift therefore
          reflects composite re-ranking, not an isolated measurement of network-evidence contribution.
        </div>
      </Panel>

      {/* Why this lead is here */}
      <Panel>
        <PanelHeader title="Why this lead is here" subtitle="Backend-provided explanation — not inferred by the interface" />
        <div className="px-5 py-4">
          <div className="flex items-start gap-2.5 rounded-md border border-border bg-bg-raised px-4 py-3 text-sm text-text-secondary">
            <Cpu className="mt-0.5 h-4 w-4 shrink-0 text-accent-strong" />
            <p>{lead.explanation || "No explanation was returned for this lead."}</p>
          </div>
          <div className="mt-3 flex items-center gap-1.5 text-2xs text-text-muted">
            <Layers className="h-3.5 w-3.5" />
            Heuristic association strength: <span className="font-mono-tabular text-text-secondary">{formatScore(lead.heuristic_association_strength)}</span>
          </div>
        </div>
      </Panel>

      {/* Blockchain evidence */}
      <Panel>
        <PanelHeader title="Blockchain evidence" subtitle={`Transaction ${lead.primary_txid.slice(0, 16)}…`} />
        {evidence ? (
          <div className="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm md:grid-cols-2">
            <Field label="TXID"><HashValue value={evidence.transaction.txid} truncate={false} /></Field>
            <Field label="Timestamp">{formatTimestamp(evidence.transaction.timestamp)}</Field>
            <Field label="Fee"><span className="font-mono-tabular">{evidence.transaction.fee} BTC</span></Field>
            <Field label="Script type"><span className="mono-id">{evidence.transaction.script_type}</span></Field>
            <Field label={`Input addresses (${evidence.transaction.input_addresses.length})`}>
              <AddressList addresses={evidence.transaction.input_addresses} />
            </Field>
            <Field label={`Output addresses (${evidence.transaction.output_addresses.length})`}>
              <AddressList addresses={evidence.transaction.output_addresses} />
            </Field>
          </div>
        ) : (
          <div className="px-5 py-6 text-sm text-text-muted">
            Blockchain evidence for this transaction is unavailable.
          </div>
        )}
      </Panel>

      {/* Network evidence */}
      <Panel>
        <PanelHeader
          title="Network evidence"
          subtitle="Relay observations correlated to this transaction — not attributed ownership"
        />
        {evidence && evidence.network_observations.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full border-collapse text-sm">
              <thead>
                <tr className="border-b border-border text-left text-2xs uppercase tracking-wide text-text-muted">
                  <th className="px-5 py-2.5 font-medium">Observation</th>
                  <th className="px-5 py-2.5 font-medium">Observed at</th>
                  <th className="px-5 py-2.5 font-medium">Observed peer</th>
                  <th className="px-5 py-2.5 font-medium">Port</th>
                  <th className="px-5 py-2.5 font-medium">Geo / ASN</th>
                  <th className="px-5 py-2.5 font-medium">Propagation delay</th>
                </tr>
              </thead>
              <tbody>
                {evidence.network_observations.map((obs) => (
                  <tr key={obs.observation_id} className="table-row-hover border-b border-border last:border-b-0">
                    <td className="px-5 py-2.5 mono-id text-text-secondary">{obs.observation_id}</td>
                    <td className="px-5 py-2.5 text-text-secondary">{formatTimestamp(obs.observed_at)}</td>
                    <td className="px-5 py-2.5 mono-id text-text-secondary">{obs.first_heard_peer_ip}</td>
                    <td className="px-5 py-2.5 font-mono-tabular text-text-secondary">{obs.src_port}</td>
                    <td className="px-5 py-2.5 text-text-muted">
                      {obs.geo_country || obs.asn ? `${obs.geo_country ?? "—"} / ${obs.asn ?? "—"}` : "Not enriched"}
                    </td>
                    <td className="px-5 py-2.5 font-mono-tabular text-text-muted">
                      {obs.propagation_delay_ms !== null ? `${obs.propagation_delay_ms} ms` : "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="px-5 py-6 text-sm text-text-muted">
            No correlated network observations for this transaction.
          </div>
        )}
        {evidence?.disclaimer && (
          <div className="border-t border-border px-5 py-3 text-2xs text-text-muted">{evidence.disclaimer}</div>
        )}
      </Panel>

      {/* Provenance */}
      <Panel>
        <PanelHeader title="Static product metadata" subtitle="Provenance disclosure — demonstration benchmark" />
        <div className="grid grid-cols-2 gap-6 px-5 py-4 md:grid-cols-4">
          <SignalField label="Source" value={DATASET_METADATA.provenance} />
          <SignalField label="Benchmark seed" value={DATASET_METADATA.seed} />
          <SignalField label="Generator version" value={DATASET_METADATA.generatorVersion} />
          <div>
            <div className="text-2xs uppercase tracking-wide text-text-muted">Derived stages</div>
            <div className="mt-1.5 flex flex-wrap gap-1">
              {["CORRELATED", "ENTITY_RESOLUTION", "INTELLIGENCE", "FUSION"].map((stage) => (
                <Badge key={stage} tone="info" mono>{stage}</Badge>
              ))}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-1.5 border-t border-border px-5 py-3 text-2xs text-text-muted">
          <ShieldCheck className="h-3.5 w-3.5" />
          Demonstration evidence is synthetic, generated for controlled evaluation — not captured from a live network.
        </div>
      </Panel>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="text-2xs uppercase tracking-wide text-text-muted">{label}</div>
      <div className="mt-1 text-text-secondary">{children}</div>
    </div>
  );
}

function AddressList({ addresses }: { addresses: string[] }) {
  if (addresses.length === 0) return <span className="text-text-muted">None</span>;
  return (
    <div className="flex flex-col gap-1">
      {addresses.slice(0, 4).map((addr) => (
        <HashValue key={addr} value={addr} lead={10} tail={6} />
      ))}
      {addresses.length > 4 && (
        <span className="text-2xs text-text-muted">+{addresses.length - 4} more</span>
      )}
    </div>
  );
}

function BackLink() {
  return (
    <Link href="/leads" className="focus-ring inline-flex items-center gap-1.5 text-sm text-text-secondary hover:text-text">
      <ArrowLeft className="h-3.5 w-3.5" /> Leads
    </Link>
  );
}
