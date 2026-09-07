export const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export async function fetchHealth() {
  try {
    const res = await fetch(`http://localhost:8080/health`, { cache: 'no-store' });
    if (!res.ok) throw new Error('Health check failed');
    return res.json();
  } catch (error) {
    console.error("Health check error:", error);
    return { status: "DOWN", services: { postgres: "DOWN", neo4j: "DOWN", intelligence_worker: "DOWN" } };
  }
}

export async function fetchIngestStatus() {
  const res = await fetch(`${API_BASE_URL}/ingest/status`, { cache: 'no-store' });
  if (!res.ok) throw new Error('Failed to fetch ingest status');
  return res.json();
}

export async function fetchLeads() {
  const res = await fetch(`${API_BASE_URL}/leads`, { cache: 'no-store' });
  if (!res.ok) throw new Error('Failed to fetch leads');
  return res.json();
}

export async function fetchEvidenceCompare(id: string) {
  const res = await fetch(`${API_BASE_URL}/evidence/compare/${id}`, { cache: 'no-store' });
  if (!res.ok) throw new Error('Failed to fetch evidence compare');
  return res.json();
}

export async function fetchSubgraph(id: string) {
  const res = await fetch(`${API_BASE_URL}/evidence/subgraph/${id}`, { cache: 'no-store' });
  if (!res.ok) throw new Error('Failed to fetch subgraph');
  return res.json();
}

export async function triggerBulkIngest() {
  const res = await fetch(`${API_BASE_URL}/ingest/bulk`, { method: 'POST' });
  if (!res.ok) throw new Error('Failed to trigger ingest');
  return res.json();
}

export async function triggerCorrelation() {
  const res = await fetch(`${API_BASE_URL}/correlate`, { method: 'POST' });
  if (!res.ok) throw new Error('Failed to trigger correlation');
  return res.json();
}
