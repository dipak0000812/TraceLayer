# TraceLayer — Round 2 Implementation Scope & Boundary

**Document Status:** Approved Architectural and Contract Baseline  
**Project:** TraceLayer (SIH 2026, PS ID: SIH26146)  
**Evaluation Milestone:** Round 2 (National Prototype Selection)  
**Requirements Authority:** National PRD v2  
**Concrete Data Reality:** Generator v1.0.0 & Seed-42 Dataset (`data/raw/`, `data/metadata/`)

---

## 1. Executive Summary & Purpose

TraceLayer is an offline, air-gapped forensic intelligence prototype designed to test the core investigative hypothesis:
> *"Does network-layer evidence add useful investigative information beyond blockchain-only analysis?"*

This document defines the **frozen implementation boundary** for the Round-2 evaluation. It specifies what must be built, what is explicitly deferred, the acceptance criteria, and the empirical evaluation framework.

---

## 2. In-Scope vs. Out-of-Scope (Round-2 Boundary)

### In-Scope (Deliverables for Round 2)
1. **Decoupled Air-Gapped Ingestion:**
   - Ingestion of synthetic blockchain transactions from `data/raw/transactions.csv` (109 records, 108 distinct TXIDs, 3 intentional duplicate rows) or `.json`.
   - Ingestion of synthetic network observations from `data/raw/network_observations.csv` (368 records) or `.json`.
   - Ingestion rejection logging, duplicate detection accounting, and dataset provenance verification against `data/metadata/manifest.json`.
2. **Offline Network Enrichment:**
   - Offline GeoIP and ASN derivation by Aniruddha's component without external HTTP calls.
   - Explicit exclusion of evaluation fixtures (`data/eval/geoip_fixture.json`, `data/eval/ground_truth.json`) from runtime pipeline execution.
3. **Correlation Engine:**
   - Decoupled ingestion into PostgreSQL with logical TXID join (`network_observations.observed_txid = transactions.txid`).
   - Categorization into `CORRELATED` and `ORPHAN` network records.
4. **Entity Resolution:**
   - In-memory Common-Input-Ownership Disjoint Set Union (DSU) algorithm in Go (`internal/entity/dsu.go`).
   - Projection of clustered entities into PostgreSQL and Neo4j.
5. **Graph Topology Projection:**
   - Neo4j graph population with five node types: `Entity`, `Address`, `Transaction`, `UTXO` (derived positional representations), and `NetworkObservation`.
   - Subgraph extraction endpoint supporting max depth $k \le 2$ and node cap $\le 100$.
6. **Dual-Layer Forensic Intelligence & Fusion:**
   - Python FastAPI intelligence worker exposing `POST /intelligence/score`.
   - Blockchain anomaly scoring via Isolation Forest ($S_{\text{chain}}$).
   - Heuristic pattern detection for peeling chains and CoinJoin-like mixing ($S_{\text{mixing}}$).
   - Network evidence quality weighting ($Q(\text{tx})$) and network anomaly scoring ($S_{\text{net}}$).
   - Sigmoid three-term fusion aggregator computing `fused_score` alongside `chain_only_score`.
7. **Lead Prioritization & Rank-Shift Triage:**
   - Persistence of `chain_only_rank`, `fused_rank`, and `rank_shift = chain_only_rank - fused_rank`.
   - REST API serving paginated leads, evidence bundles, and rank comparisons.
8. **Forensic Analyst UI:**
   - Single-page dashboard rendering the lead triage table, detail cards with heuristic association metrics, Cytoscape.js link graph, and dual-hypothesis comparison view.

### Explicitly Out-of-Scope (Strictly Deferred Past Round 2)
- ❌ **`bitcoin_traffic.csv` Ingestion:** Unmanifested pre-joined file is deprecated and excluded.
- ❌ **Live Bitcoin P2P Sniffing / Socket Capture:** No live socket listening on port 8333; no libpcap dependencies.
- ❌ **Live Blockchain RPC / Node Sync:** No connection to `bitcoind` or external public blockchain indexers.
- ❌ **Online Dynamic GeoIP APIs:** No live API requests to MaxMind or external IP geolocation providers.
- ❌ **Deep Learning / GNN Architectures:** No Graph Neural Networks, node2vec, or autoencoders.
- ❌ **Multi-Tenancy & Complex RBAC:** No multi-tenant permissions or JWT user auth; air-gapped local workstation session.
- ❌ **True Cryptographic UTXO Spend Graph:** Input data lacks `prev_txid` and `vout` references; genuine cross-transaction spend graph is deferred until generator enhancement.

---

## 3. Mathematical & Algorithmic Formulations

### 3.1 Network Evidence Quality Metric $Q(\text{tx})$

For a transaction $\text{tx}$ associated with $N$ network observations ($N = |\text{obs}(\text{tx})|$):

If $N = 0$:
$$Q(\text{tx}) = 0.0$$

If $N > 0$:
Let:
- $N_{\text{threshold}} = 3$ (minimum observations for full observation confidence)
- $\text{MAX\_SPREAD} = 120.0\text{ seconds}$ ($120{,}000\text{ ms}$, maximum acceptable propagation spread)
- $\Delta t_{\text{spread}} = \max(t_{\text{obs}}) - \min(t_{\text{obs}})$ across all observations for $\text{tx}$

$$f_{\text{count}}(N) = \min\left(1.0, \frac{N}{N_{\text{threshold}}}\right)$$

$$f_{\text{spread}}(\Delta t_{\text{spread}}) = \max\left(0.0, 1.0 - \frac{\Delta t_{\text{spread}}}{\text{MAX\_SPREAD}}\right)$$

$$Q(\text{tx}) = f_{\text{count}}(N) \times f_{\text{spread}}(\Delta t_{\text{spread}})$$

Properties:
- $Q(\text{tx}) \in [0.0, 1.0]$.
- $Q(\text{tx}) = 0.0$ when $N = 0$ (zero observations).
- $Q(\text{tx}) \to 1.0$ when $N \ge 3$ and all peer observations arrive in close temporal proximity ($\Delta t_{\text{spread}} \approx 0$).

### 3.2 Logistic Sigmoid Fusion Formulation

The fused lead priority score is computed by the three-term logistic aggregator:
$$\text{fused\_score} = \sigma\left(w_1 \cdot S_{\text{chain}} + w_2 \cdot S_{\text{net}} \cdot Q(\text{tx}) - w_3 \cdot S_{\text{mixing}} + b\right)$$

Where:
- $\sigma(z) = \frac{1}{1 + e^{-z}}$
- $S_{\text{chain}} \in [0, 1]$: Isolation Forest anomaly score from blockchain features (amount, fee ratio, output fan-out).
- $S_{\text{net}} \in [0, 1]$: Network propagation anomaly score (relay count, dispersion jitter, peer entropy).
- $S_{\text{mixing}} \in [0, 1]$: Mixing pattern penalty (peeling chain length, CoinJoin structure).
- Baseline initialization: $w_1 = 2.0, w_2 = 1.5, w_3 = 1.0, b = -1.0$ (configurable in `config/fusion_baseline.json`).

> [!IMPORTANT]
> **Separation of Fusion Fitting and Probability Calibration:**  
> The resulting `fused_score` and `chain_only_score` are discriminative ranking scores and heuristic priority metrics for analyst triage, **not** calibrated frequentist or Bayesian probabilities. They must never be described as calibrated probabilities unless explicit empirical calibration (e.g., Platt scaling or isotonic regression evaluated via Brier score and Expected Calibration Error) has been formally verified against evaluation ground truth.

---

## 4. Empirical Hypothesis Framework

The prototype evaluates the core hypothesis under a formal scientific framework:

- **Null Hypothesis ($H_0$):**  
  Adding network-layer evidence produces no measurable improvement or reordering in investigative lead triage relative to the blockchain-only baseline.
- **Alternative Hypothesis ($H_1$):**  
  Adding network-layer evidence produces measurable changes in investigative ranking and/or evaluation metrics relative to the blockchain-only baseline.

### Evaluation Metrics
1. **Rank Shift Distribution:** $\text{rank\_shift} = \text{chain\_only\_rank} - \text{fused\_rank}$ across all evaluated entities.
2. **Classification & Ranking Metrics:** Precision@K, Recall@K, Area Under the Precision-Recall Curve (AUPRC), and F1 score evaluated against `data/eval/ground_truth.json`.
3. **Model Diagnostics:** Learned weight magnitude $w_2$, feature importances, and observation quality distributions $Q(\text{tx})$.

---

## 5. Acceptance Criteria (Round-2 Verification Gates)

- **AC-01 — Reproducible Environment:** The system launches via `docker-compose up` running PostgreSQL, Neo4j, Go API, Python Intelligence, and UI. `GET /health` returns HTTP 200 with all dependencies healthy.
- **AC-02 — Bulk Ingestion:** The system ingests the seed-42 canonical raw dataset comprising exactly **109 transaction data records** and **368 network observation data records**, outputting deterministic counts and rejection accounting.
- **AC-03 — Independent Ingestion:** Network observations can be ingested before or without blockchain transactions; orphan observations are preserved with `correlation_status = 'ORPHAN'`.
- **AC-04 — Offline Enrichment:** Network observations are enriched with country and ASN codes without making external HTTP requests or reading `data/eval/`.
- **AC-05 — Entity Clustering:** The Go DSU algorithm clusters common-input addresses deterministically into persistent `entity_id` sets.
- **AC-06 — Zero-Observation Validity:** Transactions with zero network observations are processed successfully with $Q(\text{tx}) = 0.0$ and valid baseline scores.
- **AC-07 — Rank-Shift Computation:** For every evaluated entity, the system computes and persists `chain_only_rank`, `fused_rank`, and `rank_shift = chain_only_rank - fused_rank`.
- **AC-08 — Graph Topology Query:** `GET /api/v1/evidence/subgraph/{id}` returns valid Cytoscape-formatted JSON for the entity neighborhood ($k \le 2$).
- **AC-09 — Explicit Worker Failure:** If the Python intelligence worker is unavailable, the pipeline marks the run as `FAILED` or `INCOMPLETE` with an explicit error, avoiding silent heuristic substitution.
- **AC-10 — Frozen 13-Endpoint Surface:** The system exposes exactly 12 external Go REST endpoints and 1 internal Python endpoint.
