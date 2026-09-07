# TraceLayer API Contract

**Status:** Current implementation contract

This document describes the Go API in the checked-out backend working tree and the separate Python worker contract present on `origin/feature/intelligence`. Implementation is authoritative over older documentation.

## 1. API conventions

- Go API base path: `/api/v1`.
- Health is exposed at `/health`, outside the versioned API path.
- Requests and responses use JSON unless stated otherwise.
- The current server enables CORS for the prototype.

Current Go routes:

```text
GET  /health
POST /api/v1/ingest/blockchain
POST /api/v1/ingest/network
POST /api/v1/correlate
GET  /api/v1/evidence/{txid}
GET  /api/v1/leads
GET  /api/v1/leads/{id}
```

There is no public Go route for bulk ingestion, ingestion status, detection runs, detection results, evidence comparison, or evidence subgraphs.

## 2. Health

### `GET /health`

The handler checks PostgreSQL and returns a JSON health document. A successful response means the API can answer the health request; it does not mean that a Python worker is configured.

Current response shape:

```json
{
  "status": "HEALTHY",
  "timestamp": "2026-09-07T00:00:00Z",
  "services": {
    "postgres": "UP",
    "intelligence_worker": "NOT_IMPLEMENTED",
    "neo4j": "NOT_IMPLEMENTED"
  }
}
```

The `neo4j` key is a legacy diagnostic emitted by the current handler and does not represent a current dependency. The worker key is also diagnostic; the current Go health handler does not actively probe the separate Python service. The route returns `200` when PostgreSQL is healthy and `503` when it is unavailable.

## 3. Blockchain ingestion

### `POST /api/v1/ingest/blockchain`

The current handler accepts a CSV request body. JSON request bodies are not supported by this route.

The CSV header is:

```text
txid,timestamp,input_addresses,output_addresses,input_amounts,output_amounts,fee,script_type,provenance,dataset_id,generator_version
```

The raw seed-42 CSV uses JSON-encoded address and amount arrays inside CSV fields. The Go ingestion parser converts them into domain values. Amount strings are parsed using the Go domain's exact satoshi representation.

The response reports the ingestion result and counts emitted by the current handler. A successful request returns `200`; malformed CSV or invalid records return a client error; persistence failures return a server error.

## 4. Network ingestion

### `POST /api/v1/ingest/network`

The current handler accepts a CSV request body. JSON request bodies are not supported by this route.

The CSV header is:

```text
observation_id,timestamp,src_ip,dst_ip,src_port,dst_port,txid,provenance,dataset_id,generator_version
```

Network ingestion is independent of transaction ingestion. An observation is retained even when its TXID does not yet have a matching transaction. Correlation is performed later.

## 5. Correlation and ranking

### `POST /api/v1/correlate`

The current handler does not require a request body. It runs:

1. TXID correlation.
2. Deterministic entity resolution.
3. Forensic lead ranking and persistence.

Example response:

```json
{
  "status": "SUCCESS",
  "entities_clustered": 20,
  "leads_generated": 20,
  "observations_correlated": 350
}
```

The counts are data-dependent. A failure in any stage returns a server error. The current implementation does not wrap all three stages in one shared PostgreSQL transaction, so partial persistence is possible if a later stage fails.

If `INTELLIGENCE_URL` is configured, ranking calls the Python worker. If it is not configured or the worker cannot be reached, the current ranking implementation can use its heuristic fallback. This is current behavior, not an explicit `FAILED`/`INCOMPLETE` run-state contract.

## 6. Evidence

### `GET /api/v1/evidence/{txid}`

Returns the evidence bundle for one transaction ID, including the transaction and correlated network observations available to the Go evidence layer. Observation objects may contain nullable `geo_country`, `asn`, and `propagation_delay_ms` fields because they exist in the current storage and response structs; no current runtime enrichment service is required to populate them.

- `200`: evidence found.
- `400`: malformed TXID path value.
- `404`: no evidence for the TXID.
- `500`: storage failure.

No graph or subgraph representation is part of the current contract.

## 7. Leads

### `GET /api/v1/leads`

Returns persisted forensic leads ordered by `fused_rank` by default. The current handler supports `page` and `limit` query parameters, defaulting to 1 and 20, and accepts `sort_by=rank_shift` to order by rank shift. The response contains `total_leads`, `page`, `limit`, and `leads`.

### `GET /api/v1/leads/{id}`

Returns one forensic lead by its lead identifier.

A lead contains the persisted lead ID, entity ID, primary TXID, chain-only score, network score, fused score, chain-only rank, fused rank, rank shift, and explanation fields available from the current ranking/storage implementation.

## 8. Validation and errors

The Go domain validates TXIDs as 64 lowercase hexadecimal characters, transaction amount relationships, fee relationships, script type, network observation identifiers, ports, and lead score/rank invariants.

Errors use this JSON envelope from `internal/api/errors.go`:

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "description",
    "field": "txid"
  }
}
```

The `field` value may be null. Current codes are `VALIDATION_FAILED`, `NOT_FOUND`, `CONFLICT`, and `INTERNAL_ERROR`.

## 9. Amount encoding

The Go domain uses exact satoshi-based monetary values and parses decimal BTC strings without routing them through `float64`.

The Python worker schema on `origin/feature/intelligence` accepts `amount_btc` and `fee_btc` as JSON floating-point numbers. The Go intelligence client currently defines those fields as `float64`. Exact decimal preservation inside Python is therefore not guaranteed by this boundary.

## 10. Provenance and correlation

Current provenance values are defined by the Go domain and PostgreSQL constraints. Seed-42 runtime records use `SYNTHETIC`. Network observations are decoupled from transactions and may remain unmatched until `/api/v1/correlate` runs.

## 11. Go to Python boundary

The separate worker exposes:

```text
GET  /health
POST /intelligence/score
```

The Go client sends a batch of transactions containing TXID, BTC amount fields, input/output counts, script type, and network observations. The worker returns chain, network, mixing, quality, fused, flags, and explanation fields.

The worker is stateless, has no PostgreSQL connection, and loads a checked-in Isolation Forest model artifact. It is an internal service boundary, not a public frontend route.

## 12. Removed or deferred routes

The following are not current routes and must not be added to accommodate stale clients:

- `/api/v1/ingest/status`
- `/api/v1/ingest/bulk`
- `/api/v1/evidence/compare/{id}`
- `/api/v1/evidence/subgraph/{id}`
- `/api/v1/detection/run`
- `/api/v1/detection/results`

The frontend branch currently calls several of these paths and therefore remains integration-incompatible until separately updated.
