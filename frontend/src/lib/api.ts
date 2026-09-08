export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";
const API_ROOT_URL = API_BASE_URL.replace(/\/api\/v1\/?$/, "");
const API_INTERNAL_URL = process.env.API_INTERNAL_URL || API_ROOT_URL;

export interface HealthResponse {
  status: string;
  timestamp: string;
  services: Record<string, string>;
}

export interface BatchSummary {
  ingested_count: number;
  duplicate_count: number;
  rejected_count: number;
  dataset_id: string;
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
}

export interface EvidenceTransaction {
  txid: string;
  timestamp: string;
  input_addresses: string[];
  output_addresses: string[];
  fee: string;
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

export interface CorrelateResponse {
  status: string;
  entities_clustered: number;
  leads_generated: number;
  observations_correlated: number;
}

interface ApiErrorBody {
  error?: { message?: string };
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    cache: "no-store",
  });

  if (!response.ok) {
    let message = `${response.status} ${response.statusText}`;
    try {
      const body = (await response.json()) as ApiErrorBody;
      message = body.error?.message || message;
    } catch {
      // Preserve the HTTP status when the server did not return JSON.
    }
    throw new Error(message);
  }

  return (await response.json()) as T;
}

export async function fetchHealth(): Promise<HealthResponse> {
  const response = await fetch(`${typeof window === "undefined" ? API_INTERNAL_URL : API_ROOT_URL}/health`, { cache: "no-store" });
  if (!response.ok) {
    throw new Error(`Health check failed: ${response.status}`);
  }
  return (await response.json()) as HealthResponse;
}

export function fetchLeads(page = 1, limit = 20): Promise<LeadListResponse> {
  return request<LeadListResponse>(`/leads?page=${page}&limit=${limit}`);
}

export function fetchLead(id: string): Promise<Lead> {
  return request<Lead>(`/leads/${encodeURIComponent(id)}`);
}

export function fetchEvidence(txid: string): Promise<EvidenceResponse> {
  return request<EvidenceResponse>(`/evidence/${encodeURIComponent(txid)}`);
}

export function uploadBlockchainCsv(file: File): Promise<BatchSummary> {
  return request<BatchSummary>("/ingest/blockchain", {
    method: "POST",
    headers: { "Content-Type": "text/csv" },
    body: file,
  });
}

export function uploadNetworkCsv(file: File): Promise<BatchSummary> {
  return request<BatchSummary>("/ingest/network", {
    method: "POST",
    headers: { "Content-Type": "text/csv" },
    body: file,
  });
}

export function triggerCorrelation(): Promise<CorrelateResponse> {
  return request<CorrelateResponse>("/correlate", { method: "POST" });
}
