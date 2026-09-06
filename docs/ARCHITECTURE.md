# TraceLayer — System Architecture v1.0

**Document Status:** Approved Architectural and Contract Baseline  
**Scope:** Round-2 Integration Architecture  

---

## 1. System Topology & Component Layout

TraceLayer operates as an air-gapped, containerized multi-tier forensic prototype:

```
                               ┌──────────────────────────────────────────────┐
                               │           Analyst Dashboard (Svelte)         │
                               │          Port 3000 | Owner: Pushkar          │
                               └──────────────────────┬───────────────────────┘
                                                      │
                                                      │ REST JSON
                                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                       Go Backend Service (Port 8080)                                    │
│                                           Owner & Architect: Dipak                                      │
│                                                                                                         │
│  ┌───────────────────────┐   ┌───────────────────────┐   ┌───────────────────────────────────────────┐  │
│  │ Ingestion Engine      │   │ Entity Resolution     │   │ Neo4j Graph Writer                        │  │
│  │ (CSV/JSON Parser)     │   │ (internal/entity/dsu) │   │ (internal/graph/writer)                   │  │
│  │ Owner: Dipak          │   │ Owner: Aakanksha      │   │ Owner: Aakanksha                          │  │
│  └──────────┬────────────┘   └───────────┬───────────┘   └─────────────────────┬─────────────────────┘  │
│             │                            │                                     │                        │
│             │                            │                                     │                        │
│  ┌──────────▼────────────┐               │                                     │                        │
│  │ Offline GeoIP Enrich  │               │                                     │                        │
│  │ (internal/enrichment) │               │                                     │                        │
│  │ Owner: Aniruddha      │               │                                     │                        │
│  └───────────────────────┘               │                                     │                        │
└──────────────┬───────────────────────────┼─────────────────────────────────────┼────────────────────────┘
               │                           │                                     │
       SQL     │                           │ HTTP RPC (Port 8000)                │ Bolt Protocol (Port 7687)
               ▼                           ▼                                     ▼
┌─────────────────────────────┐ ┌─────────────────────────────────────┐ ┌─────────────────────────────────┐
│     PostgreSQL 16 DB        │ │     Python Intelligence Worker      │ │           Neo4j 5.x             │
│   Raw, Enriched, & Leads    │ │    (FastAPI | Owner: Sayali)        │ │       Topology Projection       │
│        Port 5432            │ │  Isolation Forest + Sigmoid Fusion  │ │           Port 7687             │
└─────────────────────────────┘ └─────────────────────────────────────┘ └─────────────────────────────────┘
```

---

## 2. Go Backend Clean Architecture (`internal/`)

The Go codebase follows Clean Architecture to ensure parallel feature development without interface conflicts:

```
cmd/
  api/
    main.go                   # Dipak: Process lifecycle, wiring, and HTTP server
internal/
  domain/                     # Dipak: Core structs (Transaction, NetworkObs, Lead)
  ingest/                     # Dipak: CSV/JSON streaming parsers and deduplication
  enrichment/
    geoip/                    # Aniruddha: Offline GeoIP/ASN static file reader
  entity/
    dsu.go                    # Aakanksha: Common-Input-Ownership clustering algorithm
  correlation/                # Dipak: Logical TXID joiner and orphan classifier
  graph/
    writer.go                 # Aakanksha: Neo4j Cypher batch projection
  intelligence/
    client.go                 # Dipak: HTTP client calling Python worker
  api/
    handlers.go               # Dipak: 12 REST endpoint handlers
    router.go                 # Dipak: Chi / standard library HTTP multiplexer
```

---

## 3. Data Flow & Processing Pipeline

1. **Step 1 — Ingestion (`POST /api/v1/ingest/bulk`):**
   - Dipak's ingestion parser reads `data/raw/transactions.csv` (109 records) and streams them into PostgreSQL `transactions`.
   - The parser reads `data/raw/network_observations.csv` (368 records) and calls Aniruddha's `geoip.Enrich(ip)` to attach country and ASN codes.
   - Records are inserted into PostgreSQL `network_observations` with `observed_txid` and default status `PENDING`. No database foreign key is enforced at this stage.
2. **Step 2 — Correlation & Entity Resolution (`POST /api/v1/correlate`):**
   - Correlation logic performs a logical join on `network_observations.observed_txid = transactions.txid`. Matching rows become `CORRELATED`; unlinked rows become `ORPHAN`.
   - Aakanksha's `dsu.Cluster()` scans all transaction inputs and assigns addresses to deterministic `entity_id` clusters.
   - Aakanksha's `graph.Writer` writes nodes (`Entity`, `Address`, `Transaction`, `UTXO`, `NetworkObservation`) and relationships to Neo4j.
3. **Step 3 — Forensic Intelligence Scoring:**
   - Dipak's `intelligence.Client` batches correlated transaction bundles to Python worker `POST /intelligence/score`.
   - Sayali's service computes:
     - $S_{\text{chain}}$ via Isolation Forest.
     - Peeling-chain and CoinJoin penalty $S_{\text{mixing}}$.
     - Network evidence quality $Q(\text{tx})$ with $N_{\text{threshold}} = 3$ and $\text{MAX\_SPREAD} = 120.0\text{s}$.
     - Network anomaly score $S_{\text{net}}$.
     - Fused priority score $\text{fused\_score} = \sigma(w_1 S_{\text{chain}} + w_2 S_{\text{net}} Q(\text{tx}) - w_3 S_{\text{mixing}} + b)$.
4. **Step 4 — Rank Shift & Lead Persistence:**
   - Python returns scores to Go.
   - Go computes `chain_only_rank`, `fused_rank`, and `rank_shift = chain_only_rank - fused_rank`.
   - Leads are written to PostgreSQL `forensic_leads`.
5. **Step 5 — Analyst Presentation:**
   - Pushkar's dashboard queries `GET /api/v1/leads` and `GET /api/v1/evidence/compare/{id}` to display lead triage and hypothesis metrics.

---

## 4. Machine Learning & Fusion Architecture

### 4.1 Feature Extraction
- **Blockchain Feature Vector:** Amount (BTC), fee ratio, input count, output count, output entropy.
- **Network Feature Vector:** Observation count $N$, dispersion spread $\Delta t_{\text{spread}}$, distinct ASNs, distinct country codes.

### 4.2 Separation of Weight Fitting from Probability Calibration
- The sigmoid fusion model produces a **discriminative ranking score / heuristic priority metric** in $[0, 1]$.
- Weight optimization ($w_1, w_2, w_3, b$) optimizes ranking metrics (such as AUPRC).
- Fused scores are **not** calibrated probabilities. Probability calibration (e.g., Platt scaling, isotonic regression) is treated as a separate post-processing stage evaluated via Brier score and Expected Calibration Error (ECE).

---

## 5. Graph Projection Architecture (Neo4j)

### Node Representation & Semantics
- `(:Entity)`: Clustered entity with UUID.
- `(:Address)`: Synthetic Bitcoin address (`sbc1...`).
- `(:Transaction)`: Bitcoin transaction (`txid`).
- `(:UTXO)`: Derived positional output node (`txid:vout_index`), representing transactional value outputs.  
  *(Note: Does not represent true cross-transaction cryptographic spend graph due to absence of `prev_txid`/`vout` in synthetic generator.)*
- `(:NetworkObservation)`: Vantage observation node with IP, country, and ASN.

### Cypher Subgraph Traversal
```cypher
MATCH (e:Entity {entity_id: $entity_id})-[:CONTROLS]->(a:Address)-[:INPUT_OF]->(t:Transaction)
OPTIONAL MATCH (t)-[:OBSERVED_IN]->(n:NetworkObservation)
OPTIONAL MATCH (t)-[:OUTPUT_TO]->(out_a:Address)
RETURN e, a, t, n, out_a LIMIT 100
```
