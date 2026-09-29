"use client";

import { ChangeEvent, useState } from "react";
import Link from "next/link";
import { ArrowRight, CheckCircle2, Database, Radio, Upload } from "lucide-react";
import { ingestBlockchainCsv, ingestNetworkCsv, ApiError, type IngestionResponse } from "@/lib/api";
import { PageHeader } from "@/components/layout/PageHeader";
import { Panel, PanelHeader, ErrorState } from "@/components/ui/States";
import { Badge } from "@/components/ui/Badge";

type Kind = "blockchain" | "network";

interface UploadState {
  fileName: string;
  fileSize: number;
  status: "uploading" | "done" | "failed";
  result?: IngestionResponse;
  error?: string;
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function IngestCard({
  kind,
  title,
  description,
  icon: Icon,
}: {
  kind: Kind;
  title: string;
  description: string;
  icon: typeof Database;
}) {
  const [state, setState] = useState<UploadState | null>(null);

  async function handleFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;

    setState({ fileName: file.name, fileSize: file.size, status: "uploading" });
    try {
      const result = kind === "blockchain" ? await ingestBlockchainCsv(file) : await ingestNetworkCsv(file);
      setState({ fileName: file.name, fileSize: file.size, status: "done", result });
    } catch (cause) {
      const message =
        cause instanceof ApiError ? cause.message : cause instanceof Error ? cause.message : "Ingestion failed";
      setState({ fileName: file.name, fileSize: file.size, status: "failed", error: message });
    }
  }

  return (
    <Panel>
      <PanelHeader
        title={title}
        subtitle={description}
        action={<Icon className="h-4 w-4 text-text-muted" />}
      />
      <div className="p-5">
        <label className="focus-ring flex cursor-pointer items-center justify-center gap-2 rounded-md border border-dashed border-border-strong bg-bg-raised px-4 py-6 text-sm text-text-secondary hover:border-accent/50 hover:text-text">
          <Upload className="h-4 w-4" />
          Select {kind} evidence CSV
          <input type="file" accept=".csv,text/csv" className="sr-only" onChange={(e) => void handleFile(e)} />
        </label>

        {state && (
          <div className="mt-4 rounded-md border border-border bg-bg-raised px-4 py-3 text-sm">
            <div className="flex items-center justify-between">
              <span className="mono-id text-text-secondary">{state.fileName}</span>
              <span className="text-2xs text-text-muted">{formatBytes(state.fileSize)}</span>
            </div>

            {state.status === "uploading" && (
              <div className="mt-2 flex items-center gap-2 text-2xs text-text-muted">
                <div className="h-1.5 w-1.5 animate-pulse rounded-full bg-accent" />
                Uploading and validating…
              </div>
            )}

            {state.status === "done" && state.result && (
              <div className="mt-3 grid grid-cols-3 gap-3 text-center">
                <div>
                  <div className="flex items-center justify-center gap-1 text-success">
                    <CheckCircle2 className="h-3.5 w-3.5" />
                    <span className="font-mono-tabular text-base font-semibold">{state.result.ingested_count}</span>
                  </div>
                  <div className="text-2xs text-text-muted">Accepted</div>
                </div>
                <div>
                  <div className="font-mono-tabular text-base font-semibold text-warning">{state.result.duplicate_count}</div>
                  <div className="text-2xs text-text-muted">Duplicate</div>
                </div>
                <div>
                  <div className="font-mono-tabular text-base font-semibold text-danger">{state.result.rejected_count}</div>
                  <div className="text-2xs text-text-muted">Rejected</div>
                </div>
              </div>
            )}
            {state.status === "done" && state.result && (
              <div className="mt-2 text-2xs text-text-muted">
                Dataset: <span className="mono-id">{state.result.dataset_id}</span>
              </div>
            )}

            {state.status === "failed" && (
              <div className="mt-3">
                <ErrorState title="Ingestion failed" message={state.error ?? "Unknown error"} />
              </div>
            )}
          </div>
        )}
      </div>
    </Panel>
  );
}

export default function IngestionPage() {
  return (
    <div>
      <PageHeader
        title="Evidence ingestion"
        subtitle="Load raw blockchain and network evidence before running correlation."
      />

      <div className="mb-6 grid grid-cols-1 gap-6 md:grid-cols-2">
        <IngestCard
          kind="blockchain"
          title="Blockchain evidence"
          description="Transaction records: TXID, inputs, outputs, fee, script type"
          icon={Database}
        />
        <IngestCard
          kind="network"
          title="Network observations"
          description="P2P propagation records: observed TXID, peer, port, timestamp"
          icon={Radio}
        />
      </div>

      <Panel>
        <div className="flex items-center justify-between px-5 py-4">
          <div className="flex items-center gap-2 text-sm text-text-secondary">
            <Badge tone="neutral">Next</Badge>
            Once evidence is ingested, run correlation to generate ranked leads.
          </div>
          <Link
            href="/correlation"
            className="focus-ring flex items-center gap-1.5 rounded-md bg-accent px-3.5 py-1.5 text-sm font-medium text-white hover:bg-accent-strong"
          >
            Go to correlation <ArrowRight className="h-3.5 w-3.5" />
          </Link>
        </div>
      </Panel>
    </div>
  );
}
