# TraceLayer — Data Contract v1.0

**Document Status:** Approved Architectural and Contract Baseline  
**Authority:** Generator v1.0.0 & Seed-42 Dataset (`data/metadata/schema.md`)  
**Scope:** Round-2 Implementation Boundary  

---

## 1. Authoritative Datasets & Counts

The canonical inputs for Round 2 are strictly restricted to the following files:

| File Path | Record Count | Distinct Keys | Notes |
|---|---|---|---|
| `data/raw/transactions.csv` | **109 data records** | 108 distinct TXIDs | 3 intentional duplicate transaction rows for dedup testing |
| `data/raw/transactions.json` | **109 records** | 108 distinct TXIDs | JSON array encoding of the transaction records |
| `data/raw/network_observations.csv` | **368 data records** | 368 observation IDs | Unenriched P2P vantage peer observations |
| `data/raw/network_observations.json` | **368 records** | 368 observation IDs | JSON array encoding of network observations |

> [!CAUTION]
> **Deprecated Artifact:** `data/raw/bitcoin_traffic.csv` is an unmanifested, pre-joined convenience file that contains pre-derived GeoIP fields. It is **not** an authoritative input and must **never** be ingested by the Go pipeline or referenced in acceptance tests.

### Isolated Evaluation Data (Strict Anti-Leakage Boundary)
- `data/eval/ground_truth.json`: Scenario classifications and true entity labels.
- `data/eval/geoip_fixture.json`: Ground truth GeoIP/ASN mapping used solely to grade offline enrichment accuracy.
- **Rule:** The runtime Go ingestion pipeline and Python intelligence worker must **never** read from `data/eval/`. Access is restricted to unit and acceptance evaluation harnesses.

---

## 2. Raw Ingestion Schemas

### 2.1 Raw Blockchain Transactions (`data/raw/transactions.csv`)

| Field Name | Type | Nullable | Constraints & Description |
|---|---|---|---|
| `txid` | `VARCHAR(64)` | No | Hex-encoded 256-bit synthetic transaction identifier (e.g., `sbc1...`) |
| `timestamp` | `TIMESTAMPTZ` | No | ISO-8601 UTC timestamp of block inclusion |
| `input_addresses` | `TEXT[]` / `JSON` | No | JSON-encoded array of synthetic input addresses (prefixed `sbc1`) |
| `output_addresses` | `TEXT[]` / `JSON` | No | JSON-encoded array of synthetic output addresses |
| `input_amounts` | `NUMERIC(16,8)[]` | No | Array of amounts in BTC (8 decimal places) |
| `output_amounts` | `NUMERIC(16,8)[]` | No | Array of amounts in BTC (8 decimal places) |
| `fee` | `NUMERIC(16,8)` | No | BTC fee; must equal $\sum(\text{inputs}) - \sum(\text{outputs}) \ge 0$ |
| `script_type` | `VARCHAR(16)` | No | Enum: `P2PKH`, `P2WPKH`, `P2SH`, `P2TR` |
| `provenance` | `VARCHAR(32)` | No | Always `SYNTHETIC` |
| `dataset_id` | `VARCHAR(32)` | No | Matches `metadata/manifest.json` (`6bc084b677a63411`) |
| `generator_version` | `VARCHAR(16)` | No | Generator release version (e.g., `1.0.0`) |

### 2.2 Raw Network Observations (`data/raw/network_observations.csv`)

This schema represents raw network packets captured before GeoIP enrichment. It intentionally omits `geo_country` and `asn`.

| Field Name | Type | Nullable | Constraints & Description |
|---|---|---|---|
| `observation_id` | `VARCHAR(64)` | No | Primary key, synthetic UUID/hash |
| `timestamp` | `TIMESTAMPTZ` | No | ISO-8601 UTC timestamp when peer announced the transaction |
| `src_ip` | `VARCHAR(45)` | No | Relay peer IPv4/IPv6 address (RFC 5737 / RFC 1918) |
| `dst_ip` | `VARCHAR(45)` | No | Vantage listener IPv4/IPv6 address |
| `src_port` | `INTEGER` | No | Port range: 1024–65535 |
| `dst_port` | `INTEGER` | No | Port range: 1024–65535 |
| `txid` | `VARCHAR(64)` | No | Observed transaction hash (joins to `transactions.txid`) |
| `provenance` | `VARCHAR(32)` | No | Always `SYNTHETIC` |
| `dataset_id` | `VARCHAR(32)` | No | Matches `metadata/manifest.json` |
| `generator_version` | `VARCHAR(16)` | No | Generator release version (e.g., `1.0.0`) |

---

## 3. Database Persistence & Ingestion Decoupling

Ingestion of transactions and network observations is fully decoupled. The database table stores `observed_txid` without an immediate foreign key constraint so that network observations can be ingested before, during, or after transactions.

```sql
-- PostgreSQL DDL (db/schema.sql)

CREATE TABLE IF NOT EXISTS transactions (
    txid VARCHAR(64) PRIMARY KEY,
    block_time TIMESTAMPTZ NOT NULL,
    input_addresses JSONB NOT NULL,
    output_addresses JSONB NOT NULL,
    input_amounts JSONB NOT NULL,
    output_amounts JSONB NOT NULL,
    fee NUMERIC(16,8) NOT NULL,
    script_type VARCHAR(16) NOT NULL,
    provenance VARCHAR(32) NOT NULL DEFAULT 'SYNTHETIC',
    dataset_id VARCHAR(32) NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS network_observations (
    observation_id VARCHAR(64) PRIMARY KEY,
    observed_txid VARCHAR(64) NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    src_ip VARCHAR(45) NOT NULL,
    dst_ip VARCHAR(45) NOT NULL,
    src_port INT NOT NULL,
    dst_port INT NOT NULL,
    geo_country VARCHAR(2),             -- Enriched offline
    asn VARCHAR(16),                    -- Enriched offline
    correlation_status VARCHAR(16) NOT NULL DEFAULT 'PENDING', -- PENDING, CORRELATED, ORPHAN
    provenance VARCHAR(32) NOT NULL DEFAULT 'SYNTHETIC',
    dataset_id VARCHAR(32) NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_net_obs_txid ON network_observations(observed_txid);
CREATE INDEX IF NOT EXISTS idx_net_obs_status ON network_observations(correlation_status);
```

### Ingestion Deduplication Rules
1. **Transactions:** Deduplicated by `txid`. If `txid` already exists, record is logged as duplicate and ignored (idempotent upsert).
2. **Network Observations:** Deduplicated by `observation_id`.

---

## 4. Derived & Enriched Models

### 4.1 Enriched Network Observation (Offline Component Output)
Produced by Aniruddha's `internal/enrichment/geoip/` module:
- `geo_country`: ISO 3166-1 alpha-2 country code (e.g., `US`, `DE`, `IN`).
- `asn`: Autonomous System Number string (e.g., `AS15169`).
- `propagation_delay_ms`: Signed millisecond delta between `observed_at` and `block_time` ($\text{observed\_at} - \text{block\_time}$). May be negative if transaction was observed in mempool before block confirmation.

### 4.2 Entity Cluster (Common-Input DSU Output)
Produced by Aakanksha's `internal/entity/dsu.go` module:
- `entity_id`: Deterministic UUID representing the clustered entity.
- `member_addresses`: Array of Bitcoin addresses linked via common-input spending heuristics.
- `cluster_size`: Total distinct addresses in cluster.

### 4.3 Neo4j Evidence Graph Schema
Node labels and relationships in Neo4j:
- Nodes:
  - `(:Entity {entity_id: STRING, size: INT})`
  - `(:Address {address: STRING})`
  - `(:Transaction {txid: STRING, fee: FLOAT, block_time: STRING})`
  - `(:UTXO {utxo_id: STRING, amount: FLOAT, index: INT})`  
    *(Derived positional representation: `txid:vout_index`)*
  - `(:NetworkObservation {observation_id: STRING, observed_at: STRING, src_ip: STRING, country: STRING, asn: STRING})`
- Relationships:
  - `(:Entity)-[:CONTROLS]->(:Address)`
  - `(:Address)-[:INPUT_OF]->(:Transaction)`
  - `(:Transaction)-[:OUTPUT_TO]->(:Address)`
  - `(:Transaction)-[:SPENDS]->(:UTXO)`
  - `(:Transaction)-[:OBSERVED_IN]->(:NetworkObservation)`

---

## 5. Intelligence Scoring & Forensic Lead Contracts

### 5.1 Network Evidence Quality Metric $Q(\text{tx})$

For a given transaction $\text{tx}$ with $N = |\text{obs}(\text{tx})|$:

$$Q(\text{tx}) = \begin{cases} 
0.0, & N = 0 \\
\min\left(1.0, \frac{N}{3}\right) \times \max\left(0.0, 1.0 - \frac{\Delta t_{\text{spread}}}{120.0}\right), & N > 0 
\end{cases}$$

Where:
- $N_{\text{threshold}} = 3$
- $\text{MAX\_SPREAD} = 120.0\text{ seconds}$
- $\Delta t_{\text{spread}} = \max(t_{\text{obs}}) - \min(t_{\text{obs}})$ (in seconds)
- When $N = 0$, $Q(\text{tx})$ is strictly $0.0$.

### 5.2 Logistic Sigmoid Fusion Formula
$$\text{fused\_score} = \sigma\left(w_1 \cdot S_{\text{chain}} + w_2 \cdot S_{\text{net}} \cdot Q(\text{tx}) - w_3 \cdot S_{\text{mixing}} + b\right)$$

- $S_{\text{chain}} \in [0, 1]$: Isolation Forest anomaly score from blockchain features.
- $S_{\text{net}} \in [0, 1]$: Network propagation anomaly score.
- $S_{\text{mixing}} \in [0, 1]$: Peeling-chain and CoinJoin penalty score.
- Baseline defaults: $w_1 = 2.0, w_2 = 1.5, w_3 = 1.0, b = -1.0$.

### 5.3 Forensic Lead Schema (`forensic_leads`)

| Field Name | Type | Nullable | Description |
|---|---|---|---|
| `lead_id` | `VARCHAR(64)` | No | Primary key |
| `entity_id` | `VARCHAR(64)` | No | Clustered entity identifier |
| `primary_txid` | `VARCHAR(64)` | No | Representative transaction under review |
| `chain_only_score` | `FLOAT` | No | Baseline score $[0, 1]$ using only blockchain features |
| `network_score` | `FLOAT` | No | Standalone network anomaly score $[0, 1]$ |
| `fused_score` | `FLOAT` | No | Logistic fusion score $[0, 1]$ |
| `chain_only_rank` | `INTEGER` | No | Priority rank without network data (1 = highest risk) |
| `fused_rank` | `INTEGER` | No | Priority rank with network fusion |
| `rank_shift` | `INTEGER` | No | $\text{chain\_only\_rank} - \text{fused\_rank}$ |
| `network_evidence_quality` | `FLOAT` | No | $Q(\text{tx}) \in [0, 1]$ |
| `heuristic_association_strength` | `FLOAT` | No | Normalized association metric $[0, 1]$ (not a calibrated probability) |
| `anomaly_flags` | `TEXT[]` | No | Flags (e.g., `RAPID_DISPERSION`, `PEELING_CHAIN`, `HIGH_FEE_RATIO`) |
| `explanation` | `TEXT` | No | Analyst-facing natural language reasoning string |
