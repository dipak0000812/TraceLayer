"use client";

import { use, useEffect, useState } from "react";
import { AlertTriangle, ArrowLeft, Cpu } from "lucide-react";
import Link from "next/link";
import { EvidenceResponse, fetchEvidence, fetchLead, Lead } from "@/lib/api";

export default function EntityAnalysisPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const [lead, setLead] = useState<Lead | null>(null);
  const [evidence, setEvidence] = useState<EvidenceResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    async function load() {
      try {
        const loadedLead = await fetchLead(id);
        setLead(loadedLead);
        setEvidence(await fetchEvidence(loadedLead.primary_txid));
      } catch (cause) {
        setError(cause instanceof Error ? cause.message : "Failed to load lead evidence");
      } finally {
        setLoading(false);
      }
    }
    void load();
  }, [id]);

  if (loading) return <div className="p-8 text-muted-foreground animate-pulse">Loading lead evidence...</div>;

  return (
    <div className="max-w-6xl mx-auto pb-8 space-y-6">
      <Link href="/" className="inline-flex items-center text-primary hover:underline text-sm font-medium"><ArrowLeft className="w-4 h-4 mr-1" />Dashboard</Link>
      {error || !lead || !evidence ? (
        <div className="bg-destructive/10 border border-destructive text-destructive p-4 rounded-md flex items-center"><AlertTriangle className="w-5 h-5 mr-2" />{error || "Lead evidence is unavailable."}</div>
      ) : (
        <>
          <div>
            <h2 className="text-2xl font-bold tracking-tight">Lead {lead.lead_id}</h2>
            <p className="text-muted-foreground mt-1">Entity {lead.entity_id} · primary transaction {lead.primary_txid}</p>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="bg-card border border-border rounded-lg p-6"><div className="text-xs text-muted-foreground mb-2 uppercase">Fused Rank</div><div className="text-5xl font-bold font-mono">#{lead.fused_rank}</div><div className="text-sm text-muted-foreground mt-2">Rank shift: {lead.rank_shift}</div></div>
            <div className="bg-card border border-border rounded-lg p-6"><div className="text-xs text-muted-foreground mb-2 uppercase">Scores</div><div className="font-mono space-y-1"><div>Chain: {lead.chain_only_score.toFixed(3)}</div><div>Network: {lead.network_score.toFixed(3)}</div><div>Fused: {lead.fused_score.toFixed(3)}</div></div></div>
            <div className="bg-card border border-border rounded-lg p-6"><div className="text-xs text-muted-foreground mb-2 uppercase">Network Quality</div><div className="text-3xl font-mono">{evidence.quality_metric.toFixed(3)}</div><div className="text-sm text-muted-foreground mt-2">{evidence.network_observations.length} observations</div></div>
          </div>
          <div className="bg-card border border-border rounded-lg p-6"><h3 className="font-semibold flex items-center mb-3"><Cpu className="w-4 h-4 mr-2" />Lead explanation</h3><p className="text-sm text-muted-foreground">{lead.explanation}</p><div className="flex gap-2 flex-wrap mt-4">{lead.anomaly_flags.map((flag) => <span key={flag} className="text-xs bg-warning/20 text-warning px-2 py-1 rounded border border-warning/30">{flag.replace(/_/g, " ")}</span>)}</div></div>
          <div className="bg-card border border-border rounded-lg overflow-hidden"><div className="px-6 py-4 border-b border-border"><h3 className="font-semibold">Transaction evidence</h3></div><div className="p-6 grid grid-cols-1 md:grid-cols-2 gap-4 text-sm"><div><span className="text-muted-foreground">TXID:</span> <span className="font-mono break-all">{evidence.transaction.txid}</span></div><div><span className="text-muted-foreground">Timestamp:</span> {evidence.transaction.timestamp}</div><div><span className="text-muted-foreground">Fee:</span> <span className="font-mono">{evidence.transaction.fee} BTC</span></div><div><span className="text-muted-foreground">Script:</span> {evidence.transaction.script_type}</div><div><span className="text-muted-foreground">Inputs:</span> {evidence.transaction.input_addresses.length}</div><div><span className="text-muted-foreground">Outputs:</span> {evidence.transaction.output_addresses.length}</div></div></div>
          <div className="bg-card border border-border rounded-lg overflow-hidden"><div className="px-6 py-4 border-b border-border"><h3 className="font-semibold">Network observations</h3></div>{evidence.network_observations.length ? <div className="overflow-x-auto"><table className="w-full text-sm text-left"><thead className="text-xs text-muted-foreground uppercase bg-muted/30"><tr><th className="px-6 py-3">Observation</th><th className="px-6 py-3">Observed</th><th className="px-6 py-3">Peer IP</th><th className="px-6 py-3">Port</th></tr></thead><tbody>{evidence.network_observations.map((observation) => <tr key={observation.observation_id} className="border-t border-border"><td className="px-6 py-3 font-mono">{observation.observation_id}</td><td className="px-6 py-3">{observation.observed_at}</td><td className="px-6 py-3 font-mono">{observation.first_heard_peer_ip}</td><td className="px-6 py-3">{observation.src_port}</td></tr>)}</tbody></table></div> : <div className="px-6 py-8 text-center text-muted-foreground">No correlated network observations.</div>}<p className="px-6 py-4 text-xs text-muted-foreground border-t border-border">{evidence.disclaimer}</p></div>
        </>
      )}
    </div>
  );
}
