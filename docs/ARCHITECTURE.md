# TraceLayer — Round-2 Architecture

**Status:** BASELINED

---

## 1. Component Map

```
+--------------------------------------------------+
|               ANALYST INTERFACE                  |
|  React 18 + TypeScript + Vite + Cytoscape.js     |
|  Port: 3000                                      |
|  Owner: Pushkar                                  |
|                                                  |
|  Dashboard | Entity Detail | Link Graph          |
|  Evidence Card | Comparison Toggle               |
+-------------------------+------------------------+
                          | REST/JSON (HTTP)
                          | Base: http://localhost:8080/api/v1
                          v
+--------------------------------------------------+
|              GO ORCHESTRATION API                |
|  cmd/api/main.go  |  Port: 8080                 |
|  Owner: Dipak                                    |
|                                                  |
|  +------------------------------------------+   |
|  | Ingestion Engine                         |   |
|  |   CSV/JSON streaming parser              |   |
|  |   Schema validation                      |   |
|  |   Rejection logger                       |   |
|  |   Duplicate handler                      |   |
|  +------------------------------------------+   |
|  | GeoIP Enrichment (Aniruddha boundary)    |   |
|  |   Offline MaxMind/GeoIP2Fast lookup      |   |
|  |   internal/enrichment/geoip/             |   |
|  +------------------------------------------+   |
|  | TXID Correlation Engine                  |   |
|  |   SQL join: network_obs <-> transactions |   |
|  +------------------------------------------+   |
|  | Entity Resolution Client                 |   |
|  |   In-memory DSU (Aakanksha interface)    |   |
|  |   Writes entity_addresses to PG          |   |
|  +------------------------------------------+   |
|  | Intelligence Client                      |   |
|  |   POST http://127.0.0.1:8000/score       |   |
|  |   Receives feature vectors               |   |
|  |   Persists detection_results             |   |
|  +------------------------------------------+   |
|  | Rank Shift Compute                       |   |
|  |   chain_only_rank, fused_rank, shift     |   |
|  +------------------------------------------+   |
+----------------+-----------------+---------------+
                 |                 |
         SQL (pgxpool)    REST/JSON loopback
         pgx/v5           http://127.0.0.1:8000
                 |                 |
+----------------v---+   +---------v--------------+
|   POSTGRESQL 16    |   | PYTHON INTELLIGENCE    |
|   Port: 5432       |   | WORKER (FastAPI)       |
|   Owner: Dipak     |   | Port: 8000 (loopback)  |
|                    |   | Owner: Sayali           |
| Tables:            |   |                        |
|  transactions      |   | POST /score            |
|  transaction_inputs|   |  - Feature extraction  |
|  transaction_outputs   |  - Isolation Forest    |
|  network_observ.   |   |  - Peeling/mixing det. |
|  entities          |   |  - Logistic fusion     |
|  entity_addresses  |   |  Returns: scores,      |
|  detection_results |   |  flags, top_features,  |
|  rejection_logs    |   |  weights_used          |
|  raw_bulk_traffic  |   |                        |
+----------+---------+   +------------------------+
           |
     Cypher (Bolt)
     bolt://localhost:7687
           |
+----------v-----------------------------------------+
|   NEO4J 5.x EVIDENCE GRAPH                         |
|   Port: 7474 (HTTP), 7687 (Bolt)                   |
|   Owner: Aakanksha                                  |
|                                                     |
|   Nodes: NetworkObservation, Transaction,           |
|          UTXO, Address, Entity                      |
|   Edges: OBSERVED, HAS_OUTPUT, SPENDS,              |
|          LOCKS_TO, BELONGS_TO                       |
|                                                     |
|   Read-only for API (GET /evidence/subgraph)        |
|   Written by Go DSU output                          |
+-----------------------------------------------------+

+--------------------------------------------------+
|   SYNTHETIC DATA (Sayali)                        |
|                                                  |
|   data/raw/         <- ONLY inputs to pipeline   |
|     bitcoin_traffic.csv  (14 PS fields, 369 rows)|
|     transactions.csv     (11 fields, 109 rows)   |
|     transactions.json                            |
|     network_observations.csv (10 fields, 369)    |
|     network_observations.json                    |
|                                                  |
|   data/eval/        <- EVALUATION ONLY           |
|     ground_truth.json   <- NEVER read by pipeline|
|     geoip_fixture.json  <- NEVER read by pipeline|
|                                                  |
|   data/metadata/                                 |
|     manifest.json                                |
|     schema.md                                    |
|     validation_report.json                       |
+--------------------------------------------------+
```

---

## 2. Data Flow

### 2.1 Ingestion Flow

```
data/raw/bitcoin_traffic.csv  (or transactions.csv + network_observations.csv)
    |
    v  POST /api/v1/ingest/bulk (or /ingest/transactions, /ingest/network)
    |
    v  Go: parse CSV/JSON rows
    |     validate each row (txid format, timestamp, IPs, amounts, fee balance)
    |     handle duplicates (first-write-wins, reject duplicates to rejection_log)
    |     parse JSON-encoded array fields (input_addresses, output_addresses, etc.)
    |
    v  Go: offline GeoIP enrichment (Aniruddha)
    |     src_ip -> geo_country, asn, asn_org via embedded GeoIP2Fast
    |     set geoip_status
    |
    v  Go: PostgreSQL writes (pgxpool)
    |     raw_bulk_traffic (audit)
    |     transactions (upsert blocked by UNIQUE constraint)
    |     transaction_inputs / transaction_outputs (exploded)
    |     network_observations
    |     rejection_logs (for any failed rows)
    |
    v  Return BatchSummary to client
```

### 2.2 Correlation + Entity Resolution Flow

```
PostgreSQL: transactions + network_observations
    |
    v  Go: TXID correlation
    |     SELECT ... JOIN ON observed_txid = txid
    |     Compute: observation_count, unique_peer_count,
    |              timing_spread, observation_coverage, peer_consistency
    |     Transactions with 0 observations: valid, Q(tx) = 0.0
    |
    v  Go DSU (Aakanksha interface boundary)
    |     Input: SELECT txid, input_addresses FROM transactions
    |     For each tx with |inputs| > 1: union all input addresses
    |     Output: entity clusters -> write to entities + entity_addresses
    |
    v  Go -> Neo4j (Aakanksha writes)
         Bolt driver writes 5-node graph projection
         NetworkObservation, Transaction, UTXO, Address, Entity nodes
         OBSERVED, HAS_OUTPUT, SPENDS, LOCKS_TO, BELONGS_TO edges
```

### 2.3 Detection Flow

```
POST /api/v1/detection/run
    |
    v  Go: extract entity feature vectors from PostgreSQL
    |     tx_count, total_volume_btc, avg_inputs_per_tx, avg_outputs_per_tx,
    |     address_count, spending_velocity (per entity)
    |
    v  Go -> Python Intelligence Worker (Sayali boundary)
    |     POST http://127.0.0.1:8000/score
    |     Request: [{entity_id, features: {...}}]
    |     Response: [{entity_id, chain_anomaly_score, network_context_score,
    |                 linkage_evidence_strength, investigative_priority_score,
    |                 peeling_chain_flag, mixing_flag, top_features, weights_used}]
    |
    v  Go: rank computation
    |     chain_only_rank = rank by chain_anomaly_score (1=most anomalous)
    |     fused_rank = rank by investigative_priority_score (1=highest priority)
    |     rank_shift = chain_only_rank - fused_rank
    |
    v  Go: persist detection_results to PostgreSQL
    |     update run status to COMPLETED
    |
    v  Client polls GET /detection/results?run_id=...
```

---

## 3. Ownership Boundaries

| Component | Owner | Primary files |
|---|---|---|
| Go API / orchestration | Dipak | cmd/api/, internal/ingestion/, internal/correlation/, internal/ranking/ |
| PostgreSQL schema and migrations | Dipak | db/schema.sql, db/migrations/ |
| Docker Compose and infrastructure | Dipak | docker-compose.yml, .env.example |
| OpenAPI specification | Dipak | api/openapi.yaml |
| GeoIP enrichment | Aniruddha | internal/enrichment/geoip/ |
| Network schema normalization | Aniruddha | internal/network/ |
| Entity resolution (DSU) | Aakanksha | internal/entity/ |
| Neo4j graph projection | Aakanksha | internal/graph/ |
| Synthetic generator | Sayali | intelligence/generator.py |
| Feature extraction | Sayali | intelligence/features.py |
| Isolation Forest | Sayali | intelligence/models/isolation_forest.py |
| Peeling/mixing detectors | Sayali | intelligence/detectors/ |
| Network evidence quality Q(tx) | Sayali | intelligence/network_quality.py |
| Logistic fusion | Sayali | intelligence/fusion.py |
| FastAPI intelligence worker | Sayali | intelligence/main.py |
| React/Cytoscape.js frontend | Pushkar | frontend/ |
| Requirements traceability | Prachi | docs/TRACEABILITY.md |
| Experiment/demo support | Prachi | tests/e2e/ |

---

## 4. Interface Boundaries

### 4.1 Go <-> Python (HTTP loopback)

- Go sends: `POST http://127.0.0.1:8000/score`
- Request body: `[{entity_id, features: {tx_count, total_volume_btc, avg_inputs_per_tx, avg_outputs_per_tx, address_count, spending_velocity}}]`
- Response body: `[{entity_id, chain_anomaly_score, network_context_score, linkage_evidence_strength, investigative_priority_score, peeling_chain_flag, mixing_flag, top_features, weights_used}]`
- Health: `GET http://127.0.0.1:8000/health` -> `{"status": "ok"}`
- Protocol: HTTP/1.1, JSON, loopback only (127.0.0.1)
- **Go does not call Python for any other purpose.**

### 4.2 Go <-> Neo4j (Bolt)

- Go writes graph projection after DSU clustering
- Go reads subgraphs for `GET /evidence/subgraph/{id}`
- Protocol: Bolt protocol, neo4j Go driver
- **Frontend does not connect to Neo4j directly.**

### 4.3 Go <-> PostgreSQL (pgxpool)

- All reads and writes go through Go
- **Python does not connect to PostgreSQL.**
- **Neo4j does not connect to PostgreSQL.**
- **Frontend does not connect to PostgreSQL.**

### 4.4 Frontend <-> Go API

- REST/JSON only
- All data derives from Go API responses
- **Frontend has zero hardcoded metrics or scores.**

---

## 5. Data Isolation (Anti-Leakage)

```
data/raw/  <-- ONLY these files are read by the pipeline
    |
    +-- bitcoin_traffic.csv   <- /ingest/bulk
    +-- transactions.csv      <- /ingest/transactions
    +-- transactions.json     <- /ingest/transactions
    +-- network_observations.csv  <- /ingest/network
    +-- network_observations.json <- /ingest/network

data/eval/  <-- FORBIDDEN from pipeline
    |
    +-- ground_truth.json    <- evaluation harness ONLY
    +-- geoip_fixture.json   <- evaluation harness ONLY
```

**Enforcement rules:**
1. Go binary: no `os.Open` or `ioutil.ReadFile` paths containing `data/eval`
2. Python worker: no file reads whatsoever (stateless compute pod)
3. Frontend: no fetch/XHR to any file path under `data/eval`
4. Only `tests/eval/` evaluation harness may read `data/eval/`

---

## 6. Deployment

```
docker compose up --build
```

Services:
- `tracelayer-api`: Go API (port 8080)
- `tracelayer-intelligence`: Python FastAPI (port 8000, internal only)
- `tracelayer-postgres`: PostgreSQL 16 (port 5432, internal)
- `tracelayer-neo4j`: Neo4j 5.x Community (port 7474, 7687, internal)
- `tracelayer-frontend`: React/Vite (port 3000)

All service images pre-cached. Zero outbound network traffic at runtime.

Config loaded from `.env` (copy from `.env.example`).
Fusion parameters loaded from `config/fusion_baseline.json`:
```json
{"w_chain": 2.5, "w_network": 1.8, "w_linkage": 1.0, "b": -2.0}
```
