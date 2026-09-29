"use client";

import { useEffect, useState } from "react";
import { RefreshCw } from "lucide-react";
import { fetchHealth, ApiError, DATASET_METADATA, type HealthResponse } from "@/lib/api";
import { PageHeader } from "@/components/layout/PageHeader";
import { Panel, PanelHeader, ErrorState } from "@/components/ui/States";
import { StatusDot } from "@/components/ui/Badge";
import { formatTimestamp } from "@/lib/format";

function toneFor(state: string): "success" | "danger" | "neutral" {
  const s = state.toUpperCase();
  if (s === "UP" || s === "HEALTHY") return "success";
  if (s === "DOWN" || s === "UNREACHABLE" || s === "MODEL_NOT_LOADED") return "danger";
  return "neutral"; // NOT_CONFIGURED, NOT_IMPLEMENTED (legacy), UNKNOWN
}

export default function HealthPage() {
  const [health, setHealth] = useState<HealthResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  async function load() {
    setLoading(true);
    setError(null);
    try {
      setHealth(await fetchHealth());
    } catch (cause) {
      setError(cause instanceof Error ? cause : new Error("Health check failed"));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    void load();
  }, []);

  return (
    <div>
      <PageHeader
        title="System health"
        subtitle="Live status reported by the TraceLayer API."
        action={
          <button
            onClick={() => void load()}
            className="focus-ring flex items-center gap-1.5 rounded-md border border-border bg-surface px-3 py-1.5 text-xs font-medium text-text-secondary hover:bg-surface-hover"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
            Refresh
          </button>
        }
      />

      <Panel className="mb-6">
        <PanelHeader title="Services" />
        {error ? (
          <div className="p-5">
            <ErrorState
              title="TraceLayer API unavailable"
              message={error.message}
              detail={error instanceof ApiError ? `${error.code ?? "ERROR"} · status ${error.status}` : undefined}
              onRetry={() => void load()}
            />
          </div>
        ) : health ? (
          <div className="divide-y divide-border">
            <div className="flex items-center justify-between px-5 py-3 text-sm">
              <span className="text-text-secondary">API</span>
              <span className="flex items-center gap-1.5">
                <StatusDot tone="success" />
                <span className="font-mono-tabular text-xs text-text">{health.status}</span>
              </span>
            </div>
            {Object.entries(health.services).map(([name, state]) => (
              <div key={name} className="flex items-center justify-between px-5 py-3 text-sm">
                <span className="capitalize text-text-secondary">{name.replace(/_/g, " ")}</span>
                <span className="flex items-center gap-1.5">
                  <StatusDot tone={toneFor(state)} />
                  <span className="font-mono-tabular text-xs text-text">{state}</span>
                </span>
              </div>
            ))}
            <div className="flex items-center justify-between px-5 py-3 text-sm">
              <span className="text-text-secondary">Last checked</span>
              <span className="font-mono-tabular text-xs text-text-muted">{formatTimestamp(health.timestamp)}</span>
            </div>
          </div>
        ) : (
          <div className="px-5 py-6 text-sm text-text-muted">Loading…</div>
        )}
      </Panel>

      <Panel className="mb-6">
        <PanelHeader title="Intelligence worker states" subtitle="What each intelligence_worker value means" />
        <div className="divide-y divide-border text-sm">
          {([
            ["UP", "success", "Worker reachable and Isolation Forest model is loaded."],
            ["MODEL_NOT_LOADED", "danger", "Worker reachable but model artifact is not loaded — scoring unavailable."],
            ["UNREACHABLE", "danger", "Worker did not respond within 3 s — scoring unavailable, heuristic fallback used."],
            ["NOT_CONFIGURED", "neutral", "INTELLIGENCE_URL is unset — worker is not wired; heuristic fallback used."],
          ] as [string, "success" | "danger" | "neutral", string][]).map(([label, tone, desc]) => (
            <div key={label} className="flex items-start gap-3 px-5 py-2.5">
              <StatusDot tone={tone} />
              <div>
                <span className="font-mono-tabular text-xs text-text">{label}</span>
                <span className="ml-2 text-2xs text-text-muted">{desc}</span>
              </div>
            </div>
          ))}
        </div>
      </Panel>
      <Panel>
        <PanelHeader title="Static product metadata" subtitle="Demonstration scope — not live database measurements" />
        <div className="grid grid-cols-3 gap-4 px-5 py-4 text-sm">
          <div>
            <div className="text-2xs uppercase tracking-wide text-text-muted">Benchmark scope</div>
            <div className="mt-1 mono-id text-text">{DATASET_METADATA.seed}</div>
          </div>
          <div>
            <div className="text-2xs uppercase tracking-wide text-text-muted">Provenance</div>
            <div className="mt-1 mono-id text-text">{DATASET_METADATA.provenance}</div>
          </div>
          <div>
            <div className="text-2xs uppercase tracking-wide text-text-muted">Generator version</div>
            <div className="mt-1 mono-id text-text">{DATASET_METADATA.generatorVersion}</div>
          </div>
        </div>
      </Panel>
    </div>
  );
}
