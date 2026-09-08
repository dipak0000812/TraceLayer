"use client";

import { ChangeEvent, useEffect, useState } from "react";
import { AlertCircle, ArrowDownRight, ArrowUpRight, Database, Minus, Play, Upload } from "lucide-react";
import Link from "next/link";
import {
  BatchSummary,
  fetchLeads,
  Lead,
  LeadListResponse,
  triggerCorrelation,
  uploadBlockchainCsv,
  uploadNetworkCsv,
} from "@/lib/api";

function formatUploadResult(result: BatchSummary): string {
  return `${result.ingested_count} ingested, ${result.duplicate_count} duplicates, ${result.rejected_count} rejected`;
}

export default function Dashboard() {
  const [leads, setLeads] = useState<LeadListResponse | null>(null);
  const [loadingLeads, setLoadingLeads] = useState(true);
  const [loadingIngest, setLoadingIngest] = useState<"blockchain" | "network" | null>(null);
  const [loadingCorrelate, setLoadingCorrelate] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  async function loadLeads() {
    try {
      setLeads(await fetchLeads());
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Failed to load leads");
    } finally {
      setLoadingLeads(false);
    }
  }

  useEffect(() => {
    let cancelled = false;
    void fetchLeads()
      .then((result) => {
        if (!cancelled) setLeads(result);
      })
      .catch((cause: unknown) => {
        if (!cancelled) setError(cause instanceof Error ? cause.message : "Failed to load leads");
      })
      .finally(() => {
        if (!cancelled) setLoadingLeads(false);
      });

    return () => {
      cancelled = true;
    };
  }, []);

  async function handleUpload(
    event: ChangeEvent<HTMLInputElement>,
    kind: "blockchain" | "network",
  ) {
    const file = event.target.files?.[0];
    if (!file) return;

    setLoadingIngest(kind);
    setError("");
    setNotice("");
    try {
      const result = kind === "blockchain"
        ? await uploadBlockchainCsv(file)
        : await uploadNetworkCsv(file);
      setNotice(`${kind === "blockchain" ? "Blockchain" : "Network"} CSV: ${formatUploadResult(result)}.`);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "CSV ingestion failed");
    } finally {
      setLoadingIngest(null);
      event.target.value = "";
    }
  }

  async function handleCorrelate() {
    setLoadingCorrelate(true);
    setLoadingLeads(true);
    setError("");
    setNotice("");
    try {
      const result = await triggerCorrelation();
      setNotice(`${result.status}: ${result.entities_clustered} entities, ${result.leads_generated} leads, ${result.observations_correlated} observations correlated.`);
      await loadLeads();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Correlation failed");
    } finally {
      setLoadingCorrelate(false);
    }
  }

  return (
    <div className="space-y-6 max-w-6xl mx-auto pb-12">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Operations Dashboard</h2>
        <p className="text-muted-foreground mt-1">Ingest raw observations, run correlation, and review forensic leads.</p>
      </div>

      {error && (
        <div className="bg-destructive/10 border border-destructive text-destructive px-4 py-3 rounded-md flex items-center">
          <AlertCircle className="w-5 h-5 mr-2" />
          {error}
        </div>
      )}
      {notice && <div className="bg-primary/10 border border-primary/30 text-primary px-4 py-3 rounded-md">{notice}</div>}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-card border border-border rounded-lg p-6">
          <h3 className="font-semibold text-lg flex items-center mb-2">
            <Database className="w-5 h-5 mr-2 text-primary" />
            Data Pipeline
          </h3>
          <p className="text-sm text-muted-foreground mb-4">Upload the raw seed-42 CSV or JSON files through the current ingestion endpoints.</p>
          <div className="space-y-3">
            <label className="flex items-center justify-between gap-3 border border-border rounded-md px-3 py-2 text-sm cursor-pointer hover:bg-accent">
              <span className="flex items-center gap-2"><Upload className="w-4 h-4" /> Blockchain data</span>
              <span>{loadingIngest === "blockchain" ? "Uploading..." : "Choose file"}</span>
              <input className="sr-only" type="file" accept=".csv,.json,text/csv,application/json" disabled={loadingIngest !== null} onChange={(event) => void handleUpload(event, "blockchain")} />
            </label>
            <label className="flex items-center justify-between gap-3 border border-border rounded-md px-3 py-2 text-sm cursor-pointer hover:bg-accent">
              <span className="flex items-center gap-2"><Upload className="w-4 h-4" /> Network data</span>
              <span>{loadingIngest === "network" ? "Uploading..." : "Choose file"}</span>
              <input className="sr-only" type="file" accept=".csv,.json,text/csv,application/json" disabled={loadingIngest !== null} onChange={(event) => void handleUpload(event, "network")} />
            </label>
          </div>
        </div>

        <div className="bg-card border border-border rounded-lg p-6 flex flex-col justify-between">
          <div>
            <h3 className="font-semibold text-lg flex items-center mb-2">
              <Play className="w-5 h-5 mr-2 text-warning" />
              Correlation and Ranking
            </h3>
            <p className="text-sm text-muted-foreground mb-4">Run TXID correlation, entity resolution, intelligence scoring, and lead ranking.</p>
          </div>
          <button onClick={() => void handleCorrelate()} disabled={loadingCorrelate} className="bg-primary hover:bg-primary/90 text-primary-foreground py-2 px-4 rounded-md text-sm font-medium transition-colors disabled:opacity-50">
            {loadingCorrelate ? "Processing..." : "Run Correlation and Ranking"}
          </button>
        </div>
      </div>

      <div className="bg-card border border-border rounded-lg overflow-hidden mt-8">
        <div className="px-6 py-4 border-b border-border bg-accent/50 flex justify-between items-center">
          <h3 className="font-semibold">Ranked Investigative Leads</h3>
          <span className="text-xs text-muted-foreground">{leads ? `${leads.total_leads} total` : "Loading"}</span>
        </div>
        {loadingLeads ? <div className="px-6 py-8 text-center text-muted-foreground">Loading leads...</div> : leads?.leads.length ? (
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead className="text-xs text-muted-foreground uppercase bg-muted/30 border-b border-border">
                <tr><th className="px-6 py-3">Rank</th><th className="px-6 py-3">Rank Shift</th><th className="px-6 py-3">Entity ID</th><th className="px-6 py-3">Chain Score</th><th className="px-6 py-3">Fused Score</th><th className="px-6 py-3">Flags</th><th className="px-6 py-3">Action</th></tr>
              </thead>
              <tbody>{leads.leads.map((lead: Lead) => (
                <tr key={lead.lead_id} className="border-b border-border hover:bg-muted/30 transition-colors">
                  <td className="px-6 py-4 font-mono font-bold">#{lead.fused_rank}</td>
                  <td className="px-6 py-4">{lead.rank_shift > 0 ? <span className="flex items-center text-destructive"><ArrowUpRight className="w-3 h-3 mr-1" />+{lead.rank_shift}</span> : lead.rank_shift < 0 ? <span className="flex items-center text-primary"><ArrowDownRight className="w-3 h-3 mr-1" />{lead.rank_shift}</span> : <span className="flex items-center text-muted-foreground"><Minus className="w-3 h-3 mr-1" />0</span>}</td>
                  <td className="px-6 py-4 font-mono text-xs">{lead.entity_id}</td>
                  <td className="px-6 py-4 font-mono">{lead.chain_only_score.toFixed(2)}</td>
                  <td className="px-6 py-4 font-mono font-medium">{lead.fused_score.toFixed(2)}</td>
                  <td className="px-6 py-4"><div className="flex gap-1 flex-wrap">{lead.anomaly_flags.map((flag) => <span key={flag} className="text-[10px] uppercase bg-warning/20 text-warning px-1.5 py-0.5 rounded border border-warning/30">{flag.replace(/_/g, " ")}</span>)}</div></td>
                  <td className="px-6 py-4"><Link href={`/entity/${encodeURIComponent(lead.lead_id)}`} className="text-primary hover:underline font-medium text-sm">Inspect</Link></td>
                </tr>
              ))}</tbody>
            </table>
          </div>
        ) : <div className="px-6 py-8 text-center text-muted-foreground">No leads available. Ingest data and run correlation.</div>}
      </div>
    </div>
  );
}
