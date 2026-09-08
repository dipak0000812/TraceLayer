# TraceLayer Round-2 Scope

**Status:** Current scope and implementation boundary

## 1. Purpose

TraceLayer is an offline-oriented forensic prototype that tests whether network observations add useful investigative evidence to blockchain transaction analysis.

## 2. Current in-scope system

Round 2 currently includes:

1. Deterministic seed-42 synthetic transaction input.
2. Independent network observation input.
3. PostgreSQL persistence.
4. TXID correlation.
5. Deterministic common-input entity resolution.
6. Evidence queries.
7. Optional stateless Python intelligence scoring.
8. Fusion and ranking.
9. Rank-shift calculation.
10. Forensic lead persistence.
11. Go REST endpoints for health, ingestion, correlation, evidence, and leads.

## 3. Runtime data

The runtime dataset is:

```text
data/synthetic/datasets/seed-42/data/raw/
```

The manifest identifies dataset `6bc084b677a63411`, generator version `1.0.0`, seed `42`, 109 transaction records, and 368 network observation records.

The following are evaluation-only:

```text
data/synthetic/datasets/seed-42/data/eval/ground_truth.json
data/synthetic/datasets/seed-42/data/eval/geoip_fixture.json
```

Runtime services must not consume those files as operational inputs.

## 4. Components

### Go API

Owns ingestion, validation, persistence, correlation, entity resolution, evidence queries, intelligence calls, ranking, and leads.

### PostgreSQL

Stores normalized transactions, network observations, entities, and forensic leads.

### Python worker

The separate worker exposes `GET /health` and `POST /intelligence/score`. It is stateless, does not connect to PostgreSQL, receives data from Go, and uses a checked-in model artifact.

### Frontend

A Next.js frontend exists on a separate branch. It is not currently aligned with the Go API and is not treated as the authoritative backend contract.

## 5. Current API surface

```text
GET  /health
POST /api/v1/ingest/blockchain
POST /api/v1/ingest/network
POST /api/v1/correlate
GET  /api/v1/evidence/{txid}
GET  /api/v1/leads
GET  /api/v1/leads/{id}
```

The internal worker endpoint is separate:

```text
POST /intelligence/score
```

## 6. What the current demo proves

The current Go tests and seed-42 pipeline code exercise domain validation, persistence behavior, ingestion, correlation, entity resolution, ranking, leads, and API handlers. `go test ./...` passes in the current working tree.

This does not prove a containerized or air-gapped deployment. Phase 14 remains future work.

## 7. Explicit exclusions

The current Round-2 system does not require:

- Graph storage or graph projection.
- Evidence graph data.
- Evidence subgraph endpoints.
- Runtime enrichment services.
- External network services.
- Runtime model or data downloads.

These may be reconsidered only as deferred future work, not as current acceptance criteria.

## 8. Known limitations

- `/api/v1/correlate` does not yet provide one shared database transaction across all stages.
- Worker-unavailable ranking can fall back to heuristics.
- The Go exact-amount model crosses into Python floating-point fields.
- The frontend branch calls obsolete backend routes.
- Offline build and runtime validation has not yet been performed.
