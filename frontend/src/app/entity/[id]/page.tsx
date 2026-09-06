"use client";

import { use, useEffect, useState } from "react";
import { fetchEvidenceCompare, fetchSubgraph } from "@/lib/api";
import NetworkGraph from "@/components/NetworkGraph";
import { ArrowLeft, ArrowUpRight, ArrowDownRight, Layers, Box, Cpu, AlertTriangle } from "lucide-react";
import Link from "next/link";

export default function EntityAnalysisPage({ params }: { params: Promise<{ id: string }> }) {
  const resolvedParams = use(params);
  const entityId = resolvedParams.id;

  const [compareData, setCompareData] = useState<any>(null);
  const [subgraphData, setSubgraphData] = useState<any>(null);
  const [loading, setLoading] = useState(true);
  const [viewMode, setViewMode] = useState<"FUSED" | "CHAIN_ONLY">("FUSED");

  useEffect(() => {
    async function load() {
      try {
        const [compare, subgraph] = await Promise.all([
          fetchEvidenceCompare(entityId).catch(() => null),
          fetchSubgraph(entityId).catch(() => null)
        ]);
        if (compare) setCompareData(compare);
        if (subgraph) setSubgraphData(subgraph);
      } catch (e) {
        console.error(e);
      }
      setLoading(false);
    }
    load();
  }, [entityId]);

  if (loading) {
    return <div className="p-8 text-muted-foreground animate-pulse">Loading forensic data for {entityId}...</div>;
  }

  if (!compareData) {
    return (
      <div className="p-8">
        <Link href="/" className="inline-flex items-center text-primary hover:underline mb-6">
          <ArrowLeft className="w-4 h-4 mr-2" /> Back to Dashboard
        </Link>
        <div className="bg-destructive/10 border border-destructive text-destructive p-4 rounded-md flex items-center">
          <AlertTriangle className="w-5 h-5 mr-2" />
          Failed to load analysis for Entity {entityId}. The entity might not exist or the API is unreachable.
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-7xl mx-auto h-full flex flex-col pb-6 space-y-6">
      
      {/* Header */}
      <div>
        <Link href="/" className="inline-flex items-center text-primary hover:underline mb-4 text-sm font-medium">
          <ArrowLeft className="w-4 h-4 mr-1" /> Dashboard
        </Link>
        <div className="flex items-start justify-between">
          <div>
            <h2 className="text-2xl font-bold tracking-tight">Entity Analysis: {entityId}</h2>
            <p className="text-muted-foreground mt-1 text-sm">{compareData.hypothesis_evaluation}</p>
          </div>
          <div className="flex items-center space-x-2 bg-card border border-border rounded-lg p-1">
            <button 
              onClick={() => setViewMode("CHAIN_ONLY")}
              className={`px-4 py-1.5 text-sm font-medium rounded-md transition-colors ${viewMode === "CHAIN_ONLY" ? "bg-accent text-foreground" : "text-muted-foreground hover:text-foreground"}`}
            >
              <Box className="w-4 h-4 inline-block mr-2 mb-0.5" />
              Blockchain Only
            </button>
            <button 
              onClick={() => setViewMode("FUSED")}
              className={`px-4 py-1.5 text-sm font-medium rounded-md transition-colors ${viewMode === "FUSED" ? "bg-primary text-primary-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"}`}
            >
              <Layers className="w-4 h-4 inline-block mr-2 mb-0.5" />
              Fused Evidence
            </button>
          </div>
        </div>
      </div>

      {/* Evidence Card */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        
        <div className={`col-span-1 rounded-lg border p-6 flex flex-col justify-center relative overflow-hidden transition-colors ${viewMode === 'FUSED' ? 'bg-primary/5 border-primary/20' : 'bg-card border-border'}`}>
          <h3 className="text-sm font-medium text-muted-foreground mb-4 uppercase tracking-wider">Investigative Priority</h3>
          <div className="text-5xl font-bold font-mono tracking-tighter mb-2">
            #{viewMode === 'FUSED' ? compareData.fused_rank : compareData.chain_only_rank}
          </div>
          <div className="flex items-center space-x-2">
            <span className="text-sm text-muted-foreground">Score:</span>
            <span className="font-mono bg-background px-2 py-0.5 rounded text-sm border border-border">
              {viewMode === 'FUSED' ? compareData.fused_score?.toFixed(2) : compareData.chain_only_score?.toFixed(2)}
            </span>
          </div>

          {viewMode === 'FUSED' && compareData.rank_shift !== 0 && (
            <div className={`mt-6 inline-flex items-center px-3 py-1.5 rounded-full text-sm font-medium border w-fit ${compareData.rank_shift > 0 ? 'bg-destructive/10 text-destructive border-destructive/20' : 'bg-primary/10 text-primary border-primary/20'}`}>
              {compareData.rank_shift > 0 ? (
                <><ArrowUpRight className="w-4 h-4 mr-1" /> Shifted up {compareData.rank_shift} ranks</>
              ) : (
                <><ArrowDownRight className="w-4 h-4 mr-1" /> Shifted down {Math.abs(compareData.rank_shift)} ranks</>
              )}
            </div>
          )}
        </div>

        <div className="col-span-1 md:col-span-2 bg-card border border-border rounded-lg p-6">
          <h3 className="text-sm font-medium text-muted-foreground mb-4 uppercase tracking-wider flex items-center">
            <Cpu className="w-4 h-4 mr-2" /> Model Details
          </h3>
          <div className="grid grid-cols-2 gap-y-4 gap-x-8">
            <div>
              <div className="text-xs text-muted-foreground mb-1">Chain Anomaly Score</div>
              <div className="font-mono text-lg">{compareData.chain_only_score?.toFixed(2) ?? '-'}</div>
            </div>
            <div>
              <div className="text-xs text-muted-foreground mb-1">Network Quality Q(tx)</div>
              <div className="font-mono text-lg">{compareData.network_contribution ? "High" : "Low"}</div>
            </div>
            <div className="col-span-2 mt-2 pt-4 border-t border-border">
              <div className="text-xs text-muted-foreground mb-2">Automated Hypothesis Generation</div>
              <p className="text-sm italic text-foreground/80 border-l-2 border-primary pl-4 py-1">
                "{compareData.hypothesis_evaluation}"
              </p>
            </div>
          </div>
        </div>

      </div>

      {/* Network Graph */}
      <div className="flex-1 min-h-[400px] flex flex-col bg-card border border-border rounded-lg overflow-hidden relative">
        <div className="px-6 py-4 border-b border-border bg-accent/50 absolute top-0 w-full z-10 flex justify-between items-center pointer-events-none">
          <h3 className="font-semibold pointer-events-auto">5-Node Evidence Graph</h3>
          <div className="flex space-x-3 pointer-events-auto bg-background/80 backdrop-blur px-3 py-1.5 rounded-md border border-border text-xs">
            <span className="flex items-center"><span className="w-3 h-3 rounded bg-[#f59e0b] mr-1.5 inline-block"></span> Entity</span>
            <span className="flex items-center"><span className="w-3 h-3 rounded bg-[#3b82f6] mr-1.5 inline-block"></span> Address</span>
            <span className="flex items-center"><span className="w-3 h-3 rounded bg-[#8b5cf6] mr-1.5 inline-block"></span> TX</span>
            <span className="flex items-center"><span className="w-3 h-3 rounded bg-[#10b981] mr-1.5 inline-block"></span> UTXO</span>
            <span className="flex items-center"><span className="w-3 h-3 rounded bg-[#ef4444] mr-1.5 inline-block"></span> Network</span>
          </div>
        </div>
        
        <div className="flex-1 w-full h-full pt-16">
          <NetworkGraph elements={subgraphData?.elements || { nodes: [], edges: [] }} />
        </div>
      </div>

    </div>
  );
}
