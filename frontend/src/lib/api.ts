/**
 * TraceLayer API client.
 *
 * Every type and function here corresponds 1:1 to api/openapi.yaml and the
 * Go handlers in internal/api. Nothing here is speculative — if a field or
 * endpoint isn't in the contract, it doesn't appear in this file.
 */

export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";
const API_ROOT_URL = API_BASE_URL.replace(/\/api\/v1\/?$/, "");
const API_INTERNAL_URL = process.env.API_INTERNAL_URL || API_ROOT_URL;

// ---------------------------------------------------------------------------
// Types (mirrors components.schemas in api/openapi.yaml)
// ---------------------------------------------------------------------------

export interface HealthResponse {
  status: string;
  timestamp: string;
  services: Record<string, string>;
}

export interface IngestionResponse {
  ingested_count: number;
  duplicate_count: number;
  rejected_count: number;
  dataset_id: string;
}

export interface CorrelateResponse {
  status: string;
  entities_clustered: number;
  leads_generated: number;
  observations_correlated: number;
}

export interface Lead {
  lead_id: string;
  entity_id: string;
  primary_txid: string;
  chain_only_score: number;
  network_score: number;
  fused_score: number;
  chain_only_rank: number;
  fused_rank: number;
  rank_shift: number;
  network_evidence_quality: number;
  heuristic_association_strength: number;
  anomaly_flags: string[];
  explanation: string;
}

export interface LeadListResponse {
  total_leads: number;
  page: number;
  limit: number;
  leads: Lead[];
}export interface EvidenceTransaction {
  txid: string;
  timestamp: string;
  input_addresses: string[];
  output_addresses: string[];
  fee: string | number;
  script_type: string;
}

export interface EvidenceObservation {
  observation_id: string;
  observed_at: string;
  first_heard_peer_ip: string;
  src_port: number;
  geo_country: string | null;
  asn: string | null;
  propagation_delay_ms: number | null;
}

export interface EvidenceResponse {
  transaction: EvidenceTransaction;
  network_observations: EvidenceObservation[];
  quality_metric: number;
  disclaimer: string;
}

export type LeadSortBy = "fused_rank" | "rank_shift";

interface ApiErrorBody {
  error?: { code?: string; message?: string; field?: string | null };
}

export class ApiError extends Error {
  status: number;
  code?: string;
  constructor(message: string, status: number, code?: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

function getApiBaseUrl(): string {
  if (typeof window === "undefined") {
    const internalRoot = (process.env.API_INTERNAL_URL || API_ROOT_URL).replace(/\/api\/v1\/?$/, "");
    return `${internalRoot}/api/v1`;
  }
  return API_BASE_URL;
}

function getApiRootUrl(): string {
  if (typeof window === "undefined") {
    return (process.env.API_INTERNAL_URL || API_ROOT_URL).replace(/\/api\/v1\/?$/, "");
  }
  return API_ROOT_URL;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;
  const baseUrl = getApiBaseUrl();
  try {
    response = await fetch(`${baseUrl}${path}`, {
      ...init,
      cache: "no-store",
    });
  } catch {
    throw new ApiError(
      "Could not reach the TraceLayer API. Confirm the backend services are running.",
      0,
      "NETWORK_UNREACHABLE",
    );
  }

  if (!response.ok) {
    let message = `Request failed (${response.status})`;
    let code: string | undefined;
    try {
      const body = (await response.json()) as ApiErrorBody;
      message = body.error?.message || message;
      code = body.error?.code;
    } catch {
      // Server did not return a JSON error body; keep the HTTP status message.
    }
    throw new ApiError(message, response.status, code);
  }

  return (await response.json()) as T;
}

// ---------------------------------------------------------------------------
// Endpoints
// ---------------------------------------------------------------------------

export async function fetchHealth(): Promise<HealthResponse> {
  const root = getApiRootUrl();
  let response: Response;
  try {
    response = await fetch(`${root}/health`, { cache: "no-store" });
  } catch {
    throw new ApiError("TraceLayer API is unreachable.", 0, "NETWORK_UNREACHABLE");
  }
  if (!response.ok) {
    throw new ApiError(`Health check failed (${response.status})`, response.status);
  }
  return (await response.json()) as HealthResponse;
}

export function fetchLeads(
  page = 1,
  limit = 20,
  sortBy: LeadSortBy = "fused_rank",
): Promise<LeadListResponse> {
  return request<LeadListResponse>(
    `/leads?page=${page}&limit=${limit}&sort_by=${sortBy}`,
  );
}

export function fetchLead(id: string): Promise<Lead> {
  return request<Lead>(`/leads/${encodeURIComponent(id)}`);
}

export function fetchEvidence(txid: string): Promise<EvidenceResponse> {
  return request<EvidenceResponse>(`/evidence/${encodeURIComponent(txid)}`);
}

export function ingestBlockchainCsv(file: File): Promise<IngestionResponse> {
  return request<IngestionResponse>("/ingest/blockchain", {
    method: "POST",
    headers: { "Content-Type": "text/csv" },
    body: file,
  });
}

export function ingestNetworkCsv(file: File): Promise<IngestionResponse> {
  return request<IngestionResponse>("/ingest/network", {
    method: "POST",
    headers: { "Content-Type": "text/csv" },
    body: file,
  });
}

export function runCorrelation(): Promise<CorrelateResponse> {
  return request<CorrelateResponse>("/correlate", { method: "POST" });
}

// ---------------------------------------------------------------------------
// Static, documented product metadata (not a live API value).
// Benchmark scope runs against a single synthetic dataset; this is disclosed
// in the UI as provenance, not presented as a measured metric.
// ---------------------------------------------------------------------------
export const DATASET_METADATA = {
  seed: "seed-42",
  provenance: "SYNTHETIC" as const,
  generatorVersion: "1.0.0",
};

export const TXID_PATTERN = /^[0-9a-f]{64}$/;

// ---------------------------------------------------------------------------
// Backward-compatibility exports
// ---------------------------------------------------------------------------
export type BatchSummary = IngestionResponse;
export const uploadBlockchainCsv = ingestBlockchainCsv;
export const uploadNetworkCsv = ingestNetworkCsv;
export const triggerCorrelation = runCorrelation;
