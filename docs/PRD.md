# TraceLayer — Round-2 Scope Boundary

**SIH Problem Statement:** SIH26146
**Client:** National Technical Research Organisation (NTRO)
**Document version:** 2.0
**Status:** BASELINED FOR ROUND-2 IMPLEMENTATION

---

## 1. What Round 2 Is

Round 2 delivers a **single, complete, verifiable, offline forensic evidence-correlation vertical slice** from synthetic raw data to a ranked investigative-lead UI.

### Core hypothesis being tested

> Does network-layer propagation evidence produce measurable rank shift when added to blockchain-only analysis?

- **H0:** Network evidence adds no change (rank_shift = 0, w_network ≈ 0).
- **H1:** Network evidence measurably re-orders investigative leads relative to chain-only analysis.

The prototype must produce empirical, non-fabricated evidence for or against H1.

### Vertical slice (exact pipeline)

```
data/raw/  (CSV + JSON)
    |
    v
Go Ingestion Engine
    (schema validation, normalization, rejection logging, GeoIP enrichment)
    |
    v
PostgreSQL  (system of record)
    |
    v
TXID Correlation
    (join network observations to transactions on txid)
    |
    v
Entity Resolution (Go DSU)
    (common-input-ownership heuristic)
    |
    v
Neo4j Evidence Graph
    (5-node topological projection)
    |
    v
Python Intelligence Worker
    (feature extraction -> Isolation Forest -> peeling/mixing -> logistic fusion)
    |
    v
Rank Computation (Go)
    (chain_only_rank, fused_rank, rank_shift)
    |
    v
Go API  (OpenAPI v1)
    |
    v
React/Cytoscape.js Frontend
    (dashboard, evidence card, link graph, comparison toggle)
    |
    v
Analyst Review
```

---

## 2. In-Scope — Round 2

| # | Capability | Owner |
|---|---|---|
| S-01 | Deterministic synthetic dataset generation (seed-42) | Sayali |
| S-02 | CSV/JSON ingestion of data/raw/ (transactions + network observations separately) | Dipak |
| S-03 | Combined-format ingestion of data/raw/bitcoin_traffic.csv (14 PS minimum fields) | Dipak |
| S-04 | Ingestion schema validation and rejection logging | Dipak |
| S-05 | PostgreSQL persistence (all normalized tables) | Dipak |
| S-06 | Deterministic duplicate-transaction handling (first-write-wins, logged) | Dipak |
| S-07 | TXID correlation (network observations <-> transactions on txid) | Dipak |
| S-08 | Transactions with zero network observations (valid state, Q(tx)=0.0) | Dipak |
| S-09 | Multiple network observations per TXID (1:N fully supported) | Dipak |
| S-10 | Orphan observation handling (obs.txid not in transactions) | Dipak |
| S-11 | Offline GeoIP/ASN enrichment of raw network observations | Aniruddha |
| S-12 | Network schema normalization | Aniruddha |
| S-13 | Entity resolution via Common-Input-Ownership DSU | Aakanksha |
| S-14 | Neo4j 5-node evidence graph projection | Aakanksha |
| S-15 | Feature extraction per entity (6 structural features) | Sayali |
| S-16 | Isolation Forest chain anomaly scoring | Sayali |
| S-17 | Peeling-chain structural detector | Sayali |
| S-18 | CoinJoin/mixing structural detector | Sayali |
| S-19 | Network evidence quality Q(tx) computation | Sayali |
| S-20 | Logistic evidence fusion -> investigative_priority_score | Sayali |
| S-21 | chain_only_rank, fused_rank, rank_shift computation | Dipak |
| S-22 | Ranked investigative lead assembly | Dipak |
| S-23 | Go REST API (13 endpoints, OpenAPI v1) | Dipak |
| S-24 | React + Cytoscape.js dashboard, evidence card, link graph, comparison toggle | Pushkar |
| S-25 | Zero mock/hardcoded data in frontend | Pushkar |
| S-26 | Docker Compose offline deployment | Dipak |
| S-27 | Provenance tagging on every persisted record | Dipak / Sayali |
| S-28 | Dataset manifest and checksum verification | Sayali |
| S-29 | Evaluation harness (offline CI only, reads data/eval/) | Sayali |
| S-30 | Requirements traceability matrix | Prachi |

---

## 3. Explicitly Out-of-Scope — Round 2

The following are FORBIDDEN from Round-2 implementation.

| Non-Goal | Rationale |
|---|---|
| Live Bitcoin mainnet monitoring | National-scale only |
| Real seized / intercepted data | No such data for this prototype |
| Guaranteed transaction-origin attribution | Peer IP = relay evidence only |
| Identifying real-world persons from addresses | Investigative leads only, not conclusions |
| Guaranteed wallet-ownership proof | Heuristic clustering only |
| Breaking CoinJoin | Detection only, not reversal |
| Autonomous enforcement or alerting | Human analyst review required |
| Multi-chain support | Bitcoin only |
| Kafka / RabbitMQ / NATS | Not needed for prototype scale |
| Redis / Memcached | Not needed |
| Kubernetes / Helm | Docker Compose sufficient |
| Spark / Airflow | Not needed |
| GNN / node2vec / LLM | Isolation Forest sufficient |
| Cloud-only services | Must run offline |
| Runtime internet dependencies | Fully offline |
| Calibrated probability outputs | Raw model scores only; not calibrated posteriors |
| Multi-analyst access control / TLS | Prototype scope |
| Bitcoin Core regtest as primary data source | Optional dev-adapter only |

---

## 4. Acceptance Criteria

| ID | Criterion | Pass Condition |
|---|---|---|
| AC-01 | Offline boot | docker compose up --build starts all services without internet; /health -> HTTP 200 all sub-services healthy |
| AC-02 | Bulk ingestion | POST /api/v1/ingest/bulk with bitcoin_traffic.csv processes all 369 rows deterministically |
| AC-03 | Separated ingestion | POST /api/v1/ingest/transactions and POST /api/v1/ingest/network each work independently |
| AC-04 | Duplicate handling | Duplicate txid: first accepted, subsequent rejected and logged to rejection_log |
| AC-05 | TXID correlation | GET /api/v1/transactions/{txid} returns transaction with network_observations array |
| AC-06 | Zero-observation transactions | Transactions with no network obs are ingested and scored (Q=0.0, no crash) |
| AC-07 | Rank shift | GET /api/v1/evidence/compare/{id} returns non-zero rank_shift for at least one entity |
| AC-08 | Evidence card math | compare response contains a(E), n(E), r(E), weights, logit, sigmoid; values from DB |
| AC-09 | Neo4j graph | GET /api/v1/evidence/subgraph/{id} returns Cytoscape.js JSON with all 5 node types |
| AC-10 | Pattern detection | peeling_chain_flag and mixing_flag correctly set on seeded scenarios |
| AC-11 | No eval leakage | Detection pipeline has zero imports/reads from data/eval/ |
| AC-12 | Provenance | Every PostgreSQL record has data_provenance = 'SYNTHETIC' |
| AC-13 | Reproducibility | Generator --seed 42 always produces dataset_id = 6bc084b677a63411 |
| AC-14 | Zero fabrication | All UI values derive from API responses; zero hardcoded statistics |

---

## 5. Assumptions

| # | Assumption |
|---|---|
| A-01 | The generator is the authoritative source of the seed-42 dataset. Its schema overrides any PRD field claim. |
| A-02 | data/raw/bitcoin_traffic.csv (369 data rows) is the combined ingest feed with all 14 PS minimum fields including geo_country and asn. |
| A-03 | data/raw/transactions.csv (109 rows) and data/raw/network_observations.csv (369 rows) are the normalized separate feeds without geo_country/asn. |
| A-04 | Amounts in raw files are in BTC (float, 8 decimal places). DB stores satoshis (multiply by 1e8, round to integer). |
| A-05 | All timestamps in raw files are ISO-8601 UTC with Z suffix. |
| A-06 | Synthetic addresses prefixed sbc1 are not real Bitcoin addresses; real Bitcoin checksum validation must NOT be applied. |
| A-07 | The Python intelligence worker is stateless and holds no database connection. |
| A-08 | Neo4j is a read-optimized projection only. PostgreSQL is the authoritative system of record. |
| A-09 | Performance thresholds are BENCHMARK-PENDING; Round-2 requires deterministic completion, not measured latency targets. |
| A-10 | geo_country and asn in bitcoin_traffic.csv are generator-embedded fields for that combined format only. The separate network_observations.csv is truly raw (no geo_country/asn). |

---

## 6. Unresolved Decisions

| ID | Decision | Blocking | Owner |
|---|---|---|---|
| UR-01 | Duplicate-handling policy: first-write-wins vs last-write-wins vs reject-all | YES | Dipak |
| UR-02 | Orphan observation policy: ingest with null FK vs reject to rejection_log | YES | Dipak |
| UR-03 | Canonical demo ingest path: bitcoin_traffic.csv or separate files? | YES | Dipak |
| UR-04 | DSU entity label format: UUID (PRD) vs sequential E0000 (generator) | YES | Aakanksha / Dipak |
| UR-05 | Isolation Forest: trained at run-time on ingested data, or pre-trained artifact in repo? | YES | Sayali |
| UR-06 | observing_node_id: not in generator output; default MONITOR_NODE_1 assumed; confirm uniqueness impact | YES | Dipak |
| UR-07 | GeoIP database file: exact path and license for embedded offline GeoIP2Fast database | NO | Aniruddha |
| UR-08 | hops parameter max for /evidence/subgraph/{id} before Neo4j timeout | NO | Aakanksha |

---

## 7. Future / Production-Only (Do Not Design In)

- Live streaming telemetry consumer (extension seam: domain.IngestionParser interface)
- Distributed graph DB / partitioned union-find (domain.EntityResolver interface)
- Async inference worker pool via gRPC/Kafka (domain.IntelligenceEngine interface)
- Partitioned PostgreSQL with read-replicas
- Neo4j Enterprise Fabric
- Kubernetes Helm / GitOps
- Multi-analyst RBAC / TLS
- GNN / node2vec / deep learning
- Automated model selection
- Calibrated probability posteriors
