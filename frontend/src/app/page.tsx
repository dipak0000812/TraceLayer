"use client";

import { useState, useEffect } from "react";
import { fetchIngestStatus, fetchLeads, triggerBulkIngest, triggerCorrelation } from "@/lib/api";
import { Play, Database, AlertCircle, ArrowUpRight, ArrowDownRight, Minus } from "lucide-react";
import Link from "next/link";

export default function Dashboard() {
  const [status, setStatus] = useState<any>(null);
  const [leads, setLeads] = useState<any>(null);
  const [loadingIngest, setLoadingIngest] = useState(false);
  const [loadingCorrelate, setLoadingCorrelate] = useState(false);
  const [error, setError] = useState("");

  const loadData = async () => {
    try {
      const [statusData, leadsData] = await Promise.all([
        fetchIngestStatus().catch(() => null),
        fetchLeads().catch(() => null)
      ]);
      if (statusData) setStatus(statusData);
      if (leadsData) setLeads(leadsData);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleIngest = async () => {
    setLoadingIngest(true);
    setError("");
    try {
      await triggerBulkIngest();
      await loadData();
    } catch (err: any) {
      setError("Ingestion failed: " + err.message);
    }
    setLoadingIngest(false);
  };

  const handleCorrelate = async () => {
    setLoadingCorrelate(true);
    setError("");
    try {
      await triggerCorrelation();
      await loadData();
    } catch (err: any) {
      setError("Correlation failed: " + err.message);
    }
    setLoadingCorrelate(false);
  };

  return (
    <div className="space-y-6 max-w-6xl mx-auto pb-12">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Operations Dashboard</h2>
        <p className="text-muted-foreground mt-1">Manage data ingestion and view forensic leads.</p>
      </div>

      {error && (
        <div className="bg-destructive/10 border border-destructive text-destructive px-4 py-3 rounded-md flex items-center">
          <AlertCircle className="w-5 h-5 mr-2" />
          {error}
        </div>
      )}

      {/* Control Panel */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <div className="bg-card border border-border rounded-lg p-6 flex flex-col justify-between">
          <div>
            <h3 className="font-semibold text-lg flex items-center mb-2">
              <Database className="w-5 h-5 mr-2 text-primary" />
              Data Pipeline
            </h3>
            <p className="text-sm text-muted-foreground mb-4">
              Transactions Ingested: <span className="text-foreground font-mono">{status?.transactions_total ?? '-'}</span><br/>
              Network Observations: <span className="text-foreground font-mono">{status?.network_observations_total ?? '-'}</span><br/>
              Correlated: <span className="text-foreground font-mono text-primary">{status?.correlated_observations ?? '-'}</span>
            </p>
          </div>
          <button 
            onClick={handleIngest} 
            disabled={loadingIngest}
            className="bg-accent hover:bg-accent/80 border border-border text-foreground py-2 px-4 rounded-md text-sm font-medium transition-colors disabled:opacity-50"
          >
            {loadingIngest ? "Ingesting..." : "Trigger Bulk Ingestion (seed-42)"}
          </button>
        </div>

        <div className="bg-card border border-border rounded-lg p-6 flex flex-col justify-between">
          <div>
            <h3 className="font-semibold text-lg flex items-center mb-2">
              <Play className="w-5 h-5 mr-2 text-warning" />
              Intelligence Engine
            </h3>
            <p className="text-sm text-muted-foreground mb-4">
              Run entity resolution (DSU) and Python scoring.<br/>
              Total Leads Available: <span className="text-foreground font-mono">{leads?.total_leads ?? '-'}</span>
            </p>
          </div>
          <button 
            onClick={handleCorrelate} 
            disabled={loadingCorrelate}
            className="bg-primary hover:bg-primary/90 text-primary-foreground py-2 px-4 rounded-md text-sm font-medium transition-colors disabled:opacity-50"
          >
            {loadingCorrelate ? "Processing..." : "Run Detection & Ranking"}
          </button>
        </div>
      </div>

      {/* Leads Table */}
      <div className="bg-card border border-border rounded-lg overflow-hidden mt-8">
        <div className="px-6 py-4 border-b border-border bg-accent/50 flex justify-between items-center">
          <h3 className="font-semibold">Ranked Investigative Leads</h3>
          <span className="text-xs text-muted-foreground">Sorted by Fused Priority</span>
        </div>
        
        <div className="overflow-x-auto">
          <table className="w-full text-sm text-left">
            <thead className="text-xs text-muted-foreground uppercase bg-muted/30 border-b border-border">
              <tr>
                <th className="px-6 py-3">Rank (Fused)</th>
                <th className="px-6 py-3">Rank Shift</th>
                <th className="px-6 py-3">Entity ID</th>
                <th className="px-6 py-3">Chain Score</th>
                <th className="px-6 py-3">Fused Score</th>
                <th className="px-6 py-3">Flags</th>
                <th className="px-6 py-3">Action</th>
              </tr>
            </thead>
            <tbody>
              {leads?.leads?.length > 0 ? (
                leads.leads.map((lead: any) => (
                  <tr key={lead.entity_id} className="border-b border-border hover:bg-muted/30 transition-colors">
                    <td className="px-6 py-4 font-mono font-bold text-foreground">#{lead.fused_rank}</td>
                    <td className="px-6 py-4">
                      {lead.rank_shift > 0 ? (
                        <span className="flex items-center text-destructive font-medium bg-destructive/10 px-2 py-1 rounded-full w-fit">
                          <ArrowUpRight className="w-3 h-3 mr-1" /> +{lead.rank_shift}
                        </span>
                      ) : lead.rank_shift < 0 ? (
                        <span className="flex items-center text-primary font-medium bg-primary/10 px-2 py-1 rounded-full w-fit">
                          <ArrowDownRight className="w-3 h-3 mr-1" /> {lead.rank_shift}
                        </span>
                      ) : (
                        <span className="flex items-center text-muted-foreground">
                          <Minus className="w-3 h-3 mr-1" /> 0
                        </span>
                      )}
                    </td>
                    <td className="px-6 py-4 font-mono text-xs">{lead.entity_id}</td>
                    <td className="px-6 py-4 font-mono text-muted-foreground">{lead.chain_only_score?.toFixed(2)}</td>
                    <td className="px-6 py-4 font-mono font-medium text-foreground">{lead.fused_score?.toFixed(2)}</td>
                    <td className="px-6 py-4">
                      <div className="flex gap-1 flex-wrap">
                        {lead.anomaly_flags?.map((flag: string) => (
                          <span key={flag} className="text-[10px] uppercase bg-warning/20 text-warning px-1.5 py-0.5 rounded border border-warning/30">
                            {flag.replace(/_/g, ' ')}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="px-6 py-4">
                      <Link href={`/entity/${lead.entity_id}`} className="text-primary hover:underline font-medium text-sm">
                        Analyze
                      </Link>
                    </td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td colSpan={7} className="px-6 py-8 text-center text-muted-foreground">
                    No leads available. Run detection pipeline to generate leads.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
