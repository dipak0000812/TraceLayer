"use client";

import { useState } from "react";
import Link from "next/link";
import { ArrowRight, Play } from "lucide-react";
import { runCorrelation, ApiError, type CorrelateResponse } from "@/lib/api";
import { PageHeader } from "@/components/layout/PageHeader";
import { Panel, PanelHeader, ErrorState } from "@/components/ui/States";
import { StatusDot } from "@/components/ui/Badge";

const STAGES = ["Ingestion", "TXID correlation", "Entity resolution", "Intelligence", "Fusion", "Ranking"];

export default function CorrelationPage() {
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<CorrelateResponse | null>(null);
  const [error, setError] = useState<Error | null>(null);

  async function handleRun() {
    setRunning(true);
    setError(null);
    setResult(null);
    try {
      const response = await runCorrelation();
      setResult(response);
    } catch (cause) {
      setError(cause instanceof Error ? cause : new Error("Correlation failed"));
    } finally {
      setRunning(false);
    }
  }

  return (
    <div>
      <PageHeader
        title="Correlation & ranking"
        subtitle="Correlate ingested evidence by TXID, resolve entities, score, and rank investigative leads."
      />

      <Panel className="mb-6">
        <PanelHeader
          title="Pipeline stages"
          subtitle="Sequence the backend executes on each run — reported as a single completed result, not per-stage status"
        />
        <div className="flex flex-wrap items-center gap-2 px-5 py-4 text-sm">
          {STAGES.map((stage, i) => (
            <div key={stage} className="flex items-center gap-2">
              <span className="rounded-md border border-border bg-bg-raised px-3 py-1.5 text-text-secondary">
                {stage}
              </span>
              {i < STAGES.length - 1 && <ArrowRight className="h-3.5 w-3.5 text-text-muted" />}
            </div>
          ))}
        </div>
      </Panel>

      <Panel>
        <PanelHeader
          title="Run correlation"
          action={
            <button
              onClick={() => void handleRun()}
              disabled={running}
              className="focus-ring flex items-center gap-1.5 rounded-md bg-accent px-3.5 py-1.5 text-sm font-medium text-white hover:bg-accent-strong disabled:opacity-50"
            >
              <Play className="h-3.5 w-3.5" />
              {running ? "Running…" : "Run correlation & ranking"}
            </button>
          }
        />
        <div className="px-5 py-5">
          {running && (
            <div className="flex items-center gap-2 text-sm text-text-secondary">
              <StatusDot tone="info" />
              Processing ingested evidence — this call blocks until the pipeline completes.
            </div>
          )}

          {!running && error && (
            <ErrorState
              title="Correlation failed"
              message={error.message}
              detail={error instanceof ApiError ? `${error.code ?? "ERROR"} · status ${error.status}` : undefined}
              onRetry={() => void handleRun()}
            />
          )}

          {!running && !error && result && (
            <div>
              <div className="flex items-center gap-2 text-sm">
                <StatusDot tone={result.status.toUpperCase() === "SUCCESS" || result.status.toUpperCase() === "OK" || result.status.toUpperCase() === "COMPLETED" ? "success" : "neutral"} />
                <span className="font-medium text-text">{result.status}</span>
              </div>
              <div className="mt-4 grid grid-cols-3 gap-4">
                <ResultStat
                  label="Newly correlated this pass"
                  value={result.observations_correlated}
                  sub="Zero on re-run — already-correlated observations are not re-processed"
                />
                <ResultStat label="Entities clustered" value={result.entities_clustered} />
                <ResultStat label="Leads generated" value={result.leads_generated} />
              </div>
              <div className="mt-5 flex justify-end">
                <Link
                  href="/leads"
                  className="focus-ring flex items-center gap-1.5 text-sm font-medium text-accent-strong hover:underline"
                >
                  Review generated leads <ArrowRight className="h-3.5 w-3.5" />
                </Link>
              </div>
            </div>
          )}

          {!running && !error && !result && (
            <p className="text-sm text-text-muted">
              No correlation run has been triggered yet in this session.
            </p>
          )}
        </div>
      </Panel>
    </div>
  );
}

function ResultStat({ label, value, sub }: { label: string; value: number; sub?: string }) {
  return (
    <div className="rounded-md border border-border bg-bg-raised px-4 py-3 text-center">
      <div className="font-mono-tabular text-xl font-semibold text-text">{value}</div>
      <div className="mt-0.5 text-2xs text-text-muted">{label}</div>
      {sub && <div className="mt-1 text-2xs text-text-muted opacity-70">{sub}</div>}
    </div>
  );
}
