# TraceLayer

TraceLayer is an offline-oriented forensic evidence-correlation prototype for the SIH 2026 project SIH26146. It combines synthetic blockchain transactions with independently ingested network observations, correlates them by TXID, resolves deterministic entities, and produces ranked forensic leads.

## Current Round-2 scope

The current implementation includes:

- Go domain validation and ingestion.
- PostgreSQL persistence.
- Transaction and network observation ingestion.
- TXID correlation.
- Deterministic common-input entity resolution.
- Evidence queries.
- Fusion, ranking, rank shift, and forensic leads.
- A separate stateless Python intelligence worker on the intelligence branch.
- A Next.js frontend on the frontend branch that still requires API reconciliation.

Graph storage, evidence subgraphs, runtime enrichment services, and external runtime services are excluded from the current Round-2 runtime.

## Architecture

```text
seed-42 raw files -> Go ingestion -> PostgreSQL
                                  |
                     TXID correlation and entities
                                  |
                       evidence and feature inputs
                                  |
                    optional Python intelligence worker
                                  |
                          fusion and ranking
                                  |
                         forensic leads and REST API
```

The Go API owns persistence and orchestration. The Python worker receives feature inputs over HTTP and does not connect directly to PostgreSQL.

## Seed-42 dataset

Runtime data is under:

```text
data/synthetic/datasets/seed-42/data/raw/
```

The manifest identifies dataset `6bc084b677a63411`, generator version `1.0.0`, random seed `42`, 109 transaction records, and 368 network observation records.

The files under `data/synthetic/datasets/seed-42/data/eval/` are evaluation-only and must not be consumed by runtime ingestion or intelligence scoring.

## Go API

The current public routes are:

```text
GET  /health
POST /api/v1/ingest/blockchain
POST /api/v1/ingest/network
POST /api/v1/correlate
GET  /api/v1/evidence/{txid}
GET  /api/v1/leads
GET  /api/v1/leads/{id}
```

See `docs/API_CONTRACT.md` and `api/openapi.yaml` for request and response details.

## Intelligence worker

The separate Python worker exposes:

```text
GET  /health
POST /intelligence/score
```

It is stateless, uses a checked-in Isolation Forest model artifact, and receives transaction/network data from Go. It does not connect to PostgreSQL. The current Go boundary sends BTC amounts as floating-point JSON fields to the Python schema, while the Go domain itself uses exact satoshi-based values.

## Local development

The checked-out backend requires a reachable PostgreSQL instance and a `DATABASE_URL` environment variable. The API listens on `:8080` by default and can be started with:

```text
go run ./cmd/api
```

The seed-42 files can then be submitted through the two ingestion endpoints before calling `/api/v1/correlate`.

The Python worker is currently on `origin/feature/intelligence`, not integrated into this checked-out branch. The frontend is currently on `origin/feature/frontend` and calls routes that are not present in the current Go router.

## Tests

Run the Go tests with:

```text
go test ./...
```

The current Go test suite passes in the working tree. This does not verify a containerized deployment, Python integration, frontend integration, or air-gapped execution.

## Offline goal and limitations

The design is intended for local/offline operation: runtime services should use local PostgreSQL, local model artifacts, and local synthetic data. Offline operation has not yet been verified end to end. Build-time dependencies are not currently proven to be available from an offline cache.

Known limitations include the lack of one shared database transaction across all `/api/v1/correlate` stages, heuristic fallback when the intelligence worker is unavailable, floating-point amounts at the Go/Python boundary, and frontend API drift.
