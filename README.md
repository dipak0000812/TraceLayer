# TraceLayer

**SIH Problem Statement:** SIH26146
**Client:** National Technical Research Organisation (NTRO)
**Theme:** Blockchain & Cybersecurity

> AI-Powered Monitoring & Analysis of Bitcoin Transaction Traffic

TraceLayer is an offline forensic evidence-correlation and investigation-support prototype that combines Bitcoin blockchain transaction evidence with Bitcoin P2P network propagation observations, correlates them through TXID, and produces ranked, explainable, uncertainty-aware investigative leads for human analyst review.

**Core hypothesis:** Does network-layer evidence add useful investigative information beyond blockchain-only analysis?

---

## What TraceLayer Is NOT

TraceLayer is a prototype that produces **investigative leads for human review**. It does NOT:
- Identify real-world persons from addresses
- Guarantee wallet ownership
- Prove transaction origin from IP observations
- Produce calibrated probabilities of criminal activity
- Make autonomous enforcement decisions

Peer IP observation = network relay evidence only. Address clustering = heuristic only. Scores = NOT calibrated probabilities.

---

## Round-2 Scope (What This Prototype Builds)

One complete verifiable vertical slice:

```
data/raw/  (synthetic CSV/JSON)
    |-> Go Ingestion + Validation + GeoIP Enrichment
    |-> PostgreSQL (system of record)
    |-> TXID Correlation
    |-> Entity Resolution (Common-Input DSU)
    |-> Neo4j Evidence Graph (5-node)
    |-> Python Intelligence Worker (Isolation Forest + peeling/mixing + fusion)
    |-> Rank Shift Computation (chain-only vs fused)
    |-> Go REST API (13 endpoints)
    |-> React/Cytoscape.js Frontend (dashboard, evidence card, link graph)
```

**Full scope:** `docs/ROUND2_SCOPE.md`
**Not in Round 2:** live mainnet, Kafka, Redis, Kubernetes, GNN/LLM, real data, cloud infrastructure

---

## Architecture (Quick Overview)

```
Frontend (React+Vite, :3000)
    |  REST/JSON
Go API (:8080)
    |                   |
PostgreSQL (:5432)   Python FastAPI (:8000, loopback)
    |
Neo4j (:7474/:7687)
```

**Full architecture:** `docs/ARCHITECTURE.md`

---

## Repository Structure

```
TraceLayer/
|-- api/
|   +-- openapi.yaml              # Normative API contract (13 endpoints)
|-- cmd/
|   +-- api/
|       +-- main.go               # Go API entry point
|-- docs/
|   |-- ROUND2_SCOPE.md           # What is in/out of scope
|   |-- DATA_CONTRACT.md          # Field-level data contracts
|   |-- API_CONTRACT.md           # Endpoint contracts + examples
|   |-- ARCHITECTURE.md           # System architecture + data flow
|   |-- INTEGRATION_CONTRACT.md   # Cross-branch integration guide
|   +-- OWNERSHIP.md              # Team ownership matrix
|-- frontend/                     # React+Vite+Cytoscape.js
|-- intelligence/                 # Python FastAPI worker
|-- internal/                     # Go packages (to be created)
|   |-- domain/                   # Shared Go types
|   |-- ingestion/                # CSV/JSON parsers
|   |-- correlation/              # TXID join logic
|   |-- entity/                   # DSU clustering
|   |-- graph/                    # Neo4j writer
|   |-- enrichment/geoip/         # Offline GeoIP
|   |-- ranking/                  # Rank shift computation
|   +-- intelligence/             # Python HTTP client
|-- db/
|   +-- schema.sql                # PostgreSQL DDL (to be created)
|-- config/
|   +-- fusion_baseline.json      # Fusion weights (to be created)
|-- go.mod                        # github.com/dipak0000812/TraceLayer, Go 1.22.2
|-- docker-compose.yml            # (to be created)
+-- .env.example                  # Environment template

data/                             # OUTSIDE TraceLayer/ directory
|-- raw/                          # ONLY inputs to pipeline
|   |-- bitcoin_traffic.csv       # Combined 14-field format (369 rows)
|   |-- transactions.csv          # Blockchain only (109 rows)
|   |-- transactions.json
|   |-- network_observations.csv  # Network only (369 rows)
|   +-- network_observations.json
|-- eval/                         # EVALUATION ONLY - never read by pipeline
|   |-- ground_truth.json
|   +-- geoip_fixture.json
+-- metadata/
    |-- manifest.json             # dataset_id=6bc084b677a63411, seed=42
    |-- schema.md
    +-- validation_report.json
```

---

## How to Run Locally

### Prerequisites
- Docker + Docker Compose
- No internet connection required after image pull

### Steps

```bash
# 1. Clone and enter TraceLayer directory
cd TraceLayer

# 2. Copy environment config
cp .env.example .env
# Edit .env if needed (defaults work for local dev)

# 3. Start all services
docker compose up --build -d

# 4. Wait for all services to be healthy
curl -s http://localhost:8080/api/v1/health | python -m json.tool

# 5. Ingest the seed-42 synthetic dataset
# Option A: Combined 14-field format
curl -X POST http://localhost:8080/api/v1/ingest/bulk \
  -H "Content-Type: text/csv" \
  --data-binary @../data/raw/bitcoin_traffic.csv

# Option B: Separated feeds
curl -X POST http://localhost:8080/api/v1/ingest/transactions \
  -H "Content-Type: text/csv" \
  --data-binary @../data/raw/transactions.csv

curl -X POST http://localhost:8080/api/v1/ingest/network \
  -H "Content-Type: text/csv" \
  --data-binary @../data/raw/network_observations.csv

# 6. Trigger detection run
RUN_ID=$(curl -s -X POST http://localhost:8080/api/v1/detection/run \
  -H "Content-Type: application/json" -d '{}' | python -c "import sys,json; print(json.load(sys.stdin)['run_id'])")

# 7. Wait for completion, then view results
curl -s "http://localhost:8080/api/v1/detection/results?run_id=${RUN_ID}" | python -m json.tool

# 8. View rank shift for an entity
curl -s http://localhost:8080/api/v1/evidence/compare/<entity_id> | python -m json.tool

# 9. Open frontend
# http://localhost:3000
```

---

## How to Generate the Seed-42 Dataset

The seed-42 dataset is already committed to `data/`. To regenerate it:

```bash
cd intelligence
pip install -r requirements.txt
python generator.py --seed 42 --output-dir ../data
```

This must produce `dataset_id = 6bc084b677a63411` in `data/metadata/manifest.json`.
Checksums are verified against manifest. If they differ, the dataset was not generated correctly.

**Do not commit a regenerated dataset to main without Sayali's sign-off.**

---

## Where Raw/Evaluation Data Live

| Location | Purpose | Pipeline access |
|---|---|---|
| data/raw/ | Primary ingestion inputs | YES — read by POST /ingest/* |
| data/eval/ground_truth.json | Anomaly labels for offline evaluation | NO — evaluation harness only |
| data/eval/geoip_fixture.json | True GeoIP answers for evaluating enrichment | NO — evaluation harness only |
| data/metadata/manifest.json | Dataset checksums, record counts, seed | Read at ingestion for dataset_id validation |

**The pipeline must NEVER read data/eval/**. Doing so constitutes label leakage.

---

## How Contracts Work

There are four contract documents, in authority order:

1. **data/metadata/schema.md** — authoritative field-level description (from generator)
2. **docs/DATA_CONTRACT.md** — full field contract table for all entities
3. **docs/API_CONTRACT.md** — endpoint request/response contracts with examples
4. **api/openapi.yaml** — machine-parseable normative spec

When in conflict, `schema.md` wins for raw field definitions.
When in conflict between `DATA_CONTRACT.md` and `openapi.yaml`, raise an issue — this is a bug.

---

## Team Ownership

| Person | Primary responsibility |
|---|---|
| Dipak | Go API, ingestion, PostgreSQL, TXID correlation, fusion orchestration, ranking, OpenAPI, Docker |
| Aniruddha | Network schema/ingestion, offline GeoIP/ASN enrichment |
| Aakanksha | Entity resolution (DSU), Neo4j evidence graph |
| Sayali | Synthetic generator, features, anomaly detection, fusion experiments, evaluation |
| Pushkar | Frontend (React + Cytoscape.js) |
| Prachi | Requirements traceability, documentation, demo support |

**Full ownership matrix:** `docs/OWNERSHIP.md`

---

## How Branches and PRs Work

| Branch | Owner | Contents |
|---|---|---|
| main | All | Integration-tested, working code only |
| feat/ingestion | Dipak | Go API + ingestion + PostgreSQL |
| feat/network-enrichment | Aniruddha | GeoIP enrichment package |
| feat/entity-graph | Aakanksha | DSU + Neo4j |
| feat/intelligence | Sayali | Python FastAPI worker |
| feat/frontend | Pushkar | React/Vite frontend |
| feat/docs-traceability | Prachi | Documentation updates |

**PR rules:**
- PRs require at least one reviewer
- PRs that cross branch boundaries (e.g., changing a shared interface) require both owners to approve
- No PR to main without passing `docker compose up --build` and `/health` returning 200
- No PR with reads from `data/eval/` in pipeline code

**Merge order:** ingestion -> network-enrichment -> entity-graph -> intelligence -> frontend
(parallel development is fine; integration testing follows this order)

---

## Current Limitations (Round 2)

1. **Synthetic data only.** No real Bitcoin network or seized data.
2. **No live monitoring.** Ingestion is batch (file upload), not streaming.
3. **Single-node PostgreSQL.** No replication or partitioning.
4. **Loopback Python worker.** Not production-scalable; adequate for prototype.
5. **Uncalibrated scores.** investigative_priority_score is NOT a probability.
6. **Heuristic entity resolution.** Common-Input-Ownership is a known heuristic with false positives.
7. **Test IP ranges only.** Synthetic IPs are RFC 5737 / RFC 1918; GeoIP results are from fixture, not real lookup.
8. **Performance untested.** Latency benchmarks are BENCHMARK-PENDING.
9. **Single analyst.** No multi-user access control.
10. **No calibration evaluation.** Precision/recall evaluation against ground truth is offline only.

---

## Key Documents

| Document | Purpose |
|---|---|
| [docs/ROUND2_SCOPE.md](docs/ROUND2_SCOPE.md) | What is in/out of Round 2; acceptance criteria |
| [docs/DATA_CONTRACT.md](docs/DATA_CONTRACT.md) | Field-level contracts for all data objects |
| [docs/API_CONTRACT.md](docs/API_CONTRACT.md) | API endpoint contracts with examples |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System architecture and data flow |
| [docs/INTEGRATION_CONTRACT.md](docs/INTEGRATION_CONTRACT.md) | Cross-branch integration guide |
| [docs/OWNERSHIP.md](docs/OWNERSHIP.md) | Team ownership matrix |
| [api/openapi.yaml](api/openapi.yaml) | Normative OpenAPI 3.0 specification |
| [data/metadata/schema.md](../data/metadata/schema.md) | Generator-authoritative field schema |
| [data/metadata/manifest.json](../data/metadata/manifest.json) | Dataset checksums and record counts |
| [docs/PRD_ROUND2.md](../docs/PRD_ROUND2.md) | Full PRD (reference; DATA_CONTRACT overrides on field detail) |
