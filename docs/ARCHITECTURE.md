# TraceLayer Architecture

**Status:** Current Round-2 implementation architecture

TraceLayer is an offline-oriented forensic evidence-correlation prototype. The current implementation is a Go and PostgreSQL backend with a separate optional Python intelligence worker. A frontend exists on a separate branch but is not currently aligned with the Go API.

## 1. Current flow

```text
seed-42 raw data
        |
        v
Go ingestion
        |
        v
PostgreSQL
        |
        +------------------+
        |                  |
        v                  v
transactions       network observations
        |                  |
        +------ TXID ------+
                 |
                 v
             correlation
                 |
                 v
        deterministic entities
                 |
                 v
              evidence
                 |
                 v
       optional Python intelligence
                 |
                 v
              fusion
                 |
                 v
              ranking
                 |
                 v
            rank shift
                 |
                 v
          forensic leads
                 |
                 v
              Go REST API
```

## 2. Go backend

The Go application owns:

- Domain validation.
- CSV/JSON ingestion parsing.
- PostgreSQL persistence.
- Transaction and network observation correlation.
- Deterministic entity resolution.
- Evidence queries.
- The HTTP API.
- The client boundary for the Python worker.
- Fusion, ranking, rank shift, and forensic lead persistence.

## 3. PostgreSQL

PostgreSQL is the system of record for normalized transactions, network observations, entities, and forensic leads. Network ingestion is independent of transaction ingestion so unmatched observations can be retained until correlation.

The initialization schema is under `init/postgres/001_schema.sql`.

## 4. Python intelligence worker

The worker on `origin/feature/intelligence` is:

- Stateless.
- HTTP-accessed by Go.
- Independent of PostgreSQL.
- Responsible for feature extraction, Isolation Forest scoring, pattern detection, fusion inputs, and explanations.
- Backed by a checked-in model artifact.

The worker exposes `GET /health` and `POST /intelligence/score`. It receives transaction and correlated network data from Go and returns transaction-level scores and explanations.

The worker is not yet integrated into a verified runtime on the checked-out backend branch. Its Dockerfile also requires a build-context correction before container work begins.

## 5. Frontend boundary

The frontend branch contains a Next.js application. It currently calls ingestion status, bulk ingestion, evidence comparison, and evidence subgraph paths that are not present in the Go router. The frontend is therefore an integration issue, not an authority for the current backend contract.

The frontend must be updated separately to consume the routes documented in the current API contract. This documentation change does not modify frontend code.

## 6. Runtime and evaluation data

Runtime uses only:

```text
data/synthetic/datasets/seed-42/data/raw/
```

Evaluation artifacts under `data/synthetic/datasets/seed-42/data/eval/` are not runtime intelligence inputs. They are used only for tests or offline evaluation.

## 7. Offline boundary

The intended runtime boundary is local PostgreSQL, the Go API, and an optional local Python worker. No current scoring path requires a public API, remote configuration, telemetry endpoint, model download, or external database.

Build-time dependency installation is a separate concern. The repository does not yet prove that all Go, Python, Node, and base-image artifacts are available without network access.

## 8. Failure behavior and known limitations

- The current `/api/v1/correlate` handler runs correlation, entity resolution, and ranking sequentially without one shared transaction boundary.
- When the worker is not configured or unavailable, current ranking behavior can fall back to heuristics.
- Go uses exact satoshi values, while the current Python boundary accepts floating-point BTC fields.
- The frontend branch is not currently compatible with the Go route set.
- Phase-14 offline execution has not been verified.

## 9. Deferred or excluded scope

The following are explicitly excluded from the current Round-2 runtime:

- Graph projection.
- Evidence graph projection.
- Evidence subgraph endpoints.
- Runtime enrichment services.

The current health response may still expose a legacy diagnostic label with a `NOT_IMPLEMENTED` value. That label is not a service, dependency, acceptance criterion, or API requirement. The excluded capabilities are future/deferred possibilities only.
