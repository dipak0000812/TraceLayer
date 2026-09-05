TRACELAYER PRD v2 — NATIONAL-LEVEL ENGINEERING BASELINE
========================================================
Team Mempool · SIH PS 26146 · NTRO · Blockchain & Cybersecurity
Version 2.0 · 2026-09-04 · Supersedes v1

> This document is the single engineering source of truth for Team Mempool.
> Every architectural decision, schema, experiment, and acceptance criterion
> derives from this document. Claims labeled **[PENDING]**, **[HYPOTHESIS]**,
> or **[VERIFY]** must NOT be presented as established facts until resolved.
> Invented numbers, unsupported citations, and fabricated results are
> prohibited in any derivative material (slides, verbal answers, demos).

---

## Table of Contents

1. Executive Summary
2. Problem Definition
3. System Goals
4. Non-Goals (Absolute)
5. Core Hypothesis
6. National-Level Success Criteria
7. User / Analyst Workflow
8. System Architecture
9. Data Architecture
10. Network Evidence Model
11. Blockchain / UTXO Model
12. Entity Resolution
13. Evidence Correlation
14. Feature Engineering
15. Anomaly Detection
16. Evidence Fusion Model
17. Calibration
18. Uncertainty Model
19. CoinJoin / Mixing Handling
20. Custodial / Service Patterns
21. Evidence Graph
22. API Contracts
23. Frontend Requirements
24. Synthetic Data / Propagation Generator
25. Dataset Provenance
26. Experiment Design
27. Evaluation Metrics
28. Leakage Prevention
29. External Validation
30. Security / Offline Deployment
31. Failure Modes
32. Logging / Observability
33. Open-Source Foundations
34. TraceLayer-Built Components
35. Team Ownership
36. Implementation Phases
37. Scope: MUST / SHOULD / OPTIONAL
38. Acceptance Criteria
39. Demo Plan
40. National-Level Judge Attack Matrix
41. Limitations (Explicit)
42. Future Production Evolution
43. Research References
44. Open Questions / Verification Gates
45. Final Definition of Done

Appendix A — Requirement ID Index
PRD v2 Change Log

---

## 1. Executive Summary

| Field | Value |
|---|---|
| Product | TraceLayer — Network + Blockchain Evidence Intelligence |
| Team | Mempool |
| SIH PS | SIH26146 |
| PS Title | AI-Powered Monitoring & Analysis of Bitcoin Transaction Traffic |
| Client | National Technical Research Organisation (NTRO) |
| Theme | Blockchain & Cybersecurity |
| PRD Version | 2.0 |

TraceLayer is a forensic evidence-correlation and investigation-support system
that ingests Bitcoin blockchain transaction data and Bitcoin peer-to-peer network
propagation observations, correlates them through transaction identifiers,
analyses both evidence layers, and produces ranked, explainable, uncertainty-aware
investigative leads for human analyst review.

**Core positioning:** Two evidence layers. One testable hypothesis.

**Engineering principle:** Evidence first. AI second.

**National-level judge target memory:**
"The team that fused network and blockchain evidence into one calibrated, explainable
evidence graph — and experimentally measured where the fusion helps and where it breaks."

TraceLayer is **not** a blockchain, wallet, mining system, smart-contract platform,
surveillance infrastructure, or real-time network monitor. It is a forensic analysis
tool designed for offline, air-gapped, investigator-controlled deployment.

---

## 2. Problem Definition

Bitcoin transactions are simultaneously observable at two evidence layers:

**Blockchain layer:** The public ledger records inputs, outputs, amounts, UTXOs,
and the graph of value movement. Chain-side analysis can infer probable entity
clusters via ownership heuristics.

**Network layer:** During transaction propagation across the peer-to-peer network,
relay observations can capture which peers reported seeing a transaction, when, and
with what timing characteristics. These observations are not recorded on-chain.

**The structural gap:** Existing investigative workflows treat these layers separately.
An analyst must manually correlate network-side observations with blockchain-side
entity evidence. There is no principled, explainable method for asking whether
network-layer observations contain additional investigative signal beyond the chain.

**The investigative question this system addresses:**
Does combining network-propagation observations with blockchain entity evidence
improve the quality or priority of investigative leads compared with chain-only analysis?

This question has a testable hypothesis, which TraceLayer is designed to evaluate.

---

## 3. System Goals

| Goal | Description |
|---|---|
| G-01 | Ingest bulk Bitcoin network propagation observations and blockchain transaction data |
| G-02 | Validate, normalize, and store both evidence layers with schema integrity |
| G-03 | Correlate network observations to blockchain transactions via TXID |
| G-04 | Cluster Bitcoin addresses into probable entity groups using established heuristics |
| G-05 | Quantify network evidence quality per transaction independently from suspiciousness |
| G-06 | Detect chain-side anomalous patterns using graph-structural features |
| G-07 | Fuse network and chain evidence into a ranked, explainable investigative output |
| G-08 | Represent evidence uncertainty explicitly throughout the system |
| G-09 | Detect structural mixing signatures and reduce entity-linkage reliability accordingly |
| G-10 | Produce an evidence graph traceable from observation to entity |
| G-11 | Allow an analyst to compare chain-only vs chain+network investigative ranking |
| G-12 | Operate entirely offline with no runtime external dependencies |
| G-13 | Run reproducible experiments to empirically test the core hypothesis |

---

## 4. Non-Goals (Absolute)

These must not appear in any implementation, claim, demo, slide, or verbal answer.

| Non-Goal | Reason |
|---|---|
| Transaction origin IP identification | Bitcoin diffusion-relay defeats single-vantage origin inference; even multi-vantage is probabilistic |
| Real-world identity attribution from addresses | Address clusters are common-controller inferences, not identity proofs |
| Defeating, breaking, or deanonymizing CoinJoin | We detect structural patterns; we do not un-mix transactions |
| Real-time / live Bitcoin network monitoring | PS requires bulk ingestion; no live capture infrastructure exists |
| Multi-chain / cross-chain analysis | PS is Bitcoin-only |
| Smart contracts / Solidity / EVM | Architecturally irrelevant to Bitcoin forensic analysis |
| LLMs / deep learning | No demonstrated requirement; explainability requirement disfavors black boxes |
| Automated enforcement, accusation, or prosecution support | System supports human investigation; it does not make determinations |
| Replacing analyst judgment | TraceLayer generates ranked leads; the analyst decides |
| Production-scale Bitcoin network infrastructure | Prototype scope; synthetic propagation model used |
| Kafka, Redis, Kubernetes, Spark, Airflow, TimescaleDB | No measured performance gap at prototype scale justifies these |
| gRPC, Node.js backend | No requirement justifies these additions |

---

## 5. Core Hypothesis

**H0 (null):**
Network-layer propagation observations provide no measurable improvement to
investigative ranking quality when added to chain-only blockchain analysis.

**H1 (alternative):**
Network-layer propagation observations, when correlated with blockchain entity
evidence through transaction identity, measurably improve investigative ranking
quality under controlled experimental conditions.

TraceLayer is the experimental apparatus built to test H0 vs H1.
The value of the system depends on producing a valid experimental result — not on
assuming H1 in advance.

**Critical methodological note:**
Network observation quality (how many peers saw a transaction, how consistent their
reports are, how tight the timing was) is a measure of evidence reliability, NOT a
direct measure of transaction or entity suspiciousness. A widely-observed transaction
is not inherently more suspicious than a sparsely-observed one. The experiment must
establish what investigative signal, if any, network evidence provides — and under
what conditions. This distinction is fundamental to every part of the system design.

---

## 6. National-Level Success Criteria

| Criterion | Threshold |
|---|---|
| Core evidence pipeline functional | Network obs → TXID → Transaction → UTXO → Address → Entity; end-to-end |
| Chain-only vs fused comparison live | Judge can observe a ranking difference on a seeded scenario; toggle calls real API |
| Evidence card decomposable | Analyst can see exactly which evidence terms produced a result |
| CoinJoin adversarial case demonstrable | System detects structural pattern and reduces linkage reliability |
| Experiment A result exists | Three arms (naive, chain-only, fused) have real precision/recall/F1 from actual runs |
| Calibration evaluated | Calibration curve and calibration error metric computed on held-out data |
| System runs offline | Air-gapped machine with all services healthy via docker compose up |
| No unsupported claim made | Demo, slides, and verbal answers contain zero fabricated metrics |
| Failure cases handled | Malformed input, missing network obs, mixing detected — all handled without crash |
| Human-in-loop boundary maintained | System never produces an identity attribution or automated accusation |

---

## 7. User / Analyst Workflow

### 7.1 Primary Persona

**Role:** NTRO Intelligence Analyst

**Technical level:** Understands blockchain forensics conceptually; not a data scientist.

**Pain points:**
- Two evidence streams (network metadata, blockchain data) analysed separately
- No unified, explainable investigative ranking
- No principled way to know whether network evidence changes what they should investigate first
- Dependency on foreign commercial tools for chain analysis

**What TraceLayer gives them:**
- A single ranked alert list derived from both evidence layers
- An explainable score decomposed into auditable evidence components
- An evidence graph linking observation to entity
- Explicit uncertainty flags when evidence is weak, conflicting, or structurally degraded
- The ability to compare what chain-only analysis would have prioritised vs the fused view

### 7.2 Workflow Steps

```
1. INGEST
   Submit bulk CSV/JSON/XML containing network observations and blockchain transactions.
   System validates, normalizes, deduplicates, and enriches.
   Malformed records rejected and logged; batch continues.

2. CORRELATION (automatic)
   TXID joins network observations to blockchain transactions.
   Blockchain transactions resolved to UTXOs → addresses → probable entities.

3. EVIDENCE EXTRACTION (automatic)
   Per-transaction network evidence quality computed.
   Chain-side graph features extracted per entity.
   Structural mixing signatures detected.
   Custodial consolidation patterns flagged.

4. ANOMALY DETECTION (automatic)
   Graph-structural anomaly score computed per entity (IF or LOF).
   Score normalized to [0,1]. Top contributing features identified.

5. EVIDENCE FUSION (automatic)
   Evidence fusion model combines:
     - Chain-side anomaly score
     - Network evidence context (see Section 16 — does NOT assume network obs = suspicious)
     - Mixing/linkage reliability signal
   Produces ranked investigative priority with explainable decomposition.

6. REVIEW ALERT LIST
   Analyst sees ranked entity list; confidence/priority score; per-entity evidence summary.

7. INSPECT EVIDENCE CARD
   For any entity: score decomposed; contributing evidence terms; uncertainty flags;
   top anomaly features; network evidence quality; linkage reliability; limitations.

8. NAVIGATE EVIDENCE GRAPH
   NetworkObservation → Transaction → UTXO → Address → Entity
   Every edge carries provenance: which rule or model produced it, when, confidence.

9. COMPARE CHAIN-ONLY vs FUSED
   For any entity: chain-only rank vs fused rank; rank shift; explanation of what changed.

10. INVESTIGATE (HUMAN DECISION)
    A. Elevate lead for further investigation
    B. Dismiss — insufficient evidence
    C. Flag for follow-up

    The system makes no determinations. The analyst decides.
```

### 7.3 Human-in-Loop Boundary

TraceLayer produces ranked, explainable leads. The analyst makes all investigative
decisions. The system never produces identity attributions, enforcement recommendations,
or probabilistic conclusions about real-world persons.

---

## 8. System Architecture

### 8.1 Component Map

```
┌─────────────────────────────────────────────────────────────┐
│                      ANALYST INTERFACE                       │
│   React + Cytoscape.js                                      │
│   Dashboard | Entity Detail | Evidence Graph | Comparison   │
└─────────────────────┬───────────────────────────────────────┘
                      │ REST / JSON
┌─────────────────────▼───────────────────────────────────────┐
│                   GO ORCHESTRATION API                       │
│   Ingest │ Validate │ Correlate │ Orchestrate │ Serve       │
└───────┬──────────────────────────────────┬──────────────────┘
        │ SQL                              │ REST / JSON
┌───────▼──────────┐           ┌───────────▼──────────────────┐
│   PostgreSQL     │           │   PYTHON INTELLIGENCE SERVICE │
│   Raw evidence   │           │   Feature extraction          │
│   Events / logs  │           │   Anomaly detection (IF/LOF)  │
│   Experiments    │           │   Evidence fusion model       │
│   Ground truth   │           │   Calibration                 │
└──────────────────┘           │   Experiments                 │
                               └──────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│                     NEO4J EVIDENCE GRAPH                     │
│   Nodes: NetworkObs │ Transaction │ UTXO │ Address │ Entity  │
│   Edges with provenance, confidence, evidence_type          │
└─────────────────────────────────────────────────────────────┘
┌───────────────┐  ┌──────────────────┐  ┌───────────────────┐
│ Bitcoin Core  │  │  GeoIP2Fast      │  │ Synthetic Data    │
│ (regtest)     │  │  (local file,    │  │ Generator         │
│               │  │  offline)        │  │                   │
└───────────────┘  └──────────────────┘  └───────────────────┘
```

### 8.2 Data Flow

```
Bulk input (CSV / JSON / XML)
     │
     ▼
Go Ingestion Service
 ├─ Schema validation
 ├─ Deduplication
 ├─ Rejection logging
 └─ GeoIP2Fast enrichment (offline)
     │
     ├──────────────────────────────────────────┐
     ▼                                          ▼
PostgreSQL                                  Neo4j
NetworkObservation                    NetworkObservation node
Transaction / Input / Output / UTXO   Transaction node
                                      UTXO node → Address node
     │                                          │
     └──────────────────┬───────────────────────┘
                        │
                        ▼
               TXID Correlation
               (Go orchestration)
                        │
                        ▼
               Entity Resolution
               (Common-input + change-address heuristics)
               Entity nodes in Neo4j
                        │
                        ▼
               Python Intelligence Service
               ├─ Network evidence quality per TX
               ├─ Entity network context aggregation
               ├─ Graph feature extraction per entity
               ├─ Anomaly scoring (IF / LOF)
               ├─ Mixing structural detection
               └─ Evidence fusion model
                        │
                        ▼
               FusionEvidence + DetectionResult
               (PostgreSQL + Neo4j edges)
                        │
                        ▼
               REST API → React Frontend
```

### 8.3 Architecture Decisions (Locked)

| Decision | Rationale | Status |
|---|---|---|
| Go for API/ingestion/orchestration | Team strength; goroutines; strong typing; single binary | Locked |
| Python for intelligence/ML | scikit-learn ecosystem; feature extraction; experiments | Locked |
| PostgreSQL for structured storage | Raw events, logs, experiments — relational queries | Locked |
| Neo4j for evidence graph | Native graph traversal for multi-hop evidence queries | Locked |
| REST/JSON inter-service communication | No measured performance gap requiring gRPC at this scale | Locked |
| Docker + Docker Compose for deployment | Reproducible offline deployment; show-don't-tell | Locked |
| Bitcoin Core regtest | Real Bitcoin semantics on controlled private chain | Locked |
| GeoIP2Fast offline | Zero runtime network calls; local .dat.gz file | Locked |
| No TimescaleDB, Kafka, Redis, Kubernetes | No demonstrated requirement at prototype scale | Locked |
| node2vec: conditional only | Include only if ablation shows >2 F1-point improvement | Conditional |
| FastAPI as thin Python wrapper only | Intelligence service is not a primary backend | Locked |

---

## 9. Data Architecture

All schemas are canonical. The ML team must NOT independently redefine them.
Schema changes require API contract review by Dipak before implementation.

### 9.1 NetworkObservation

| Field | Type | Required | Description | Validation |
|---|---|---|---|---|
| id | UUID | Yes | Primary key | UUID v4 |
| observed_txid | VARCHAR(64) | Yes | Bitcoin TXID observed | Hex, 64 chars |
| observing_node_id | VARCHAR(64) | Yes | Identifier of the monitoring node | Non-empty |
| peer_ip | VARCHAR(45) | Yes | IP of the reporting peer (NOT claimed origin) | Valid IPv4/IPv6 |
| peer_port | INTEGER | Yes | Port of the reporting peer | 1–65535 |
| first_seen_utc | TIMESTAMPTZ | Yes | When this node first received this TX | RFC3339 UTC |
| message_type | VARCHAR(16) | Yes | Bitcoin P2P message type | Enum: tx, inv |
| asn | VARCHAR(16) | No | ASN after GeoIP enrichment | Post-enrichment |
| asn_org | VARCHAR(256) | No | ASN organization | Post-enrichment |
| geo_country | VARCHAR(2) | No | ISO 3166-1 alpha-2 | 2-char or null |
| geoip_status | VARCHAR(32) | No | resolved / private_range / not_found / error | Enum |
| data_provenance | VARCHAR(32) | Yes | REAL_NODE_CAPTURE / SYNTHETIC / REAL_PUBLIC | Enum |
| ingested_at | TIMESTAMPTZ | Yes | When ingested | Server time |
| batch_id | UUID | Yes | Ingestion batch reference | UUID v4 |

**Dedup key:** (observed_txid, observing_node_id, peer_ip, first_seen_utc)
**Terminology note:** peer_ip is the IP of the peer that reported propagation. It is NOT
labelled "source_ip" or "origin_ip" — those semantics cannot be established from
relay observations.

### 9.2 TransactionNetworkEvidence (per-TXID aggregation)

| Field | Type | Required | Description |
|---|---|---|---|
| id | UUID | Yes | Primary key |
| txid | VARCHAR(64) | Yes | Transaction this evidence applies to |
| observer_count | INTEGER | Yes | Number of distinct observing nodes |
| unique_peer_count | INTEGER | Yes | Number of distinct peer IPs reported |
| earliest_seen_utc | TIMESTAMPTZ | Yes | Earliest first_seen_utc across all observers |
| latest_seen_utc | TIMESTAMPTZ | Yes | Latest first_seen_utc across all observers |
| timing_spread_seconds | FLOAT | Yes | latest - earliest in seconds |
| peer_consistency | FLOAT | Yes | See Section 10 for definition [0,1] |
| observation_coverage | FLOAT | Yes | clip(observer_count / N_REF, 0, 1) |
| evidence_quality | FLOAT | Yes | Composite evidence quality [0,1] — NOT suspiciousness |
| data_provenance | VARCHAR(32) | Yes | Provenance of underlying observations |
| computed_at | TIMESTAMPTZ | Yes | When aggregated |

**Critical naming:** evidence_quality is a measure of observation reliability.
It is NOT a suspiciousness score and must not be treated as one.

### 9.3 Transaction

| Field | Type | Required | Description |
|---|---|---|---|
| txid | VARCHAR(64) | Yes | Primary key |
| block_height | INTEGER | No | Block height if confirmed |
| block_hash | VARCHAR(64) | No | Block hash |
| confirmed_at | TIMESTAMPTZ | No | Block timestamp |
| fee_satoshis | BIGINT | No | Fee in satoshis |
| total_input_value | BIGINT | No | Sum of input values in satoshis |
| total_output_value | BIGINT | No | Sum of output values in satoshis |
| input_count | INTEGER | Yes | Number of inputs |
| output_count | INTEGER | Yes | Number of outputs |
| is_coinbase | BOOLEAN | Yes | True if coinbase transaction |
| mixing_flag | BOOLEAN | No | Structural mixing signature detected |
| mixing_type | VARCHAR(32) | No | whirlpool_candidate / wasabi_v1_candidate / generic / null |
| mixing_confidence_level | VARCHAR(16) | No | high / medium / low / null — see Section 19 |
| data_provenance | VARCHAR(32) | Yes | REAL_NODE_CAPTURE / SYNTHETIC / REAL_PUBLIC |
| ingested_at | TIMESTAMPTZ | Yes | Ingestion timestamp |

**Note on mixing_type:** Values are labeled _candidate to reflect that these are
structural pattern matches, not confirmed CoinJoin instances.

### 9.4 TransactionInput / TransactionOutput

TransactionInput: id (UUID), txid, input_index, prev_txid (null for coinbase),
prev_vout (null for coinbase), address (VARCHAR 128, nullable), value_satoshis (BIGINT, nullable).

TransactionOutput: id (UUID), txid, vout, address (VARCHAR 128, nullable),
value_satoshis (BIGINT), is_change_candidate (BOOLEAN, heuristic),
is_spent (BOOLEAN), spent_by_txid (VARCHAR 64, nullable).

### 9.5 UTXO

id (UUID), txid, vout, address (VARCHAR 128), value_satoshis (BIGINT),
is_spent (BOOLEAN), spent_by_txid (VARCHAR 64, nullable).

### 9.6 Address

address (VARCHAR 128, PK), entity_id (UUID, nullable),
heuristic_association_strength (FLOAT [0,1], nullable),
first_seen_txid, last_seen_txid, tx_count (INTEGER).

**Terminology note:** Field is named heuristic_association_strength, NOT
"cluster_confidence". A heuristic linkage score is not a calibrated probability.

### 9.7 Entity

| Field | Type | Required | Description |
|---|---|---|---|
| entity_id | UUID | Yes | Primary key |
| address_count | INTEGER | Yes | Addresses in cluster |
| cluster_method | VARCHAR(64) | Yes | common_input / change_address / combined |
| heuristic_association_strength | FLOAT | Yes | [0,1] — strength of heuristic evidence; NOT a calibrated probability |
| mixing_exposure | BOOLEAN | No | Any associated TX has mixing_flag=true |
| service_pattern_flag | BOOLEAN | No | High-input-count consolidation pattern detected |
| chain_anomaly_score | FLOAT | No | Normalized [0,1] IF/LOF output |
| network_context_score | FLOAT | No | Aggregated network evidence quality [0,1] (see Section 10) |
| investigative_priority_score | FLOAT | No | Output of evidence fusion model [0,1] |
| created_at | TIMESTAMPTZ | Yes | Cluster creation time |
| last_updated_at | TIMESTAMPTZ | Yes | Last update |

### 9.8 EntityNetworkContext

| Field | Type | Required | Description |
|---|---|---|---|
| id | UUID | Yes | Primary key |
| entity_id | UUID | Yes | Entity this context applies to |
| observed_tx_count | INTEGER | Yes | TXs in T(E) that have network observations |
| total_tx_count | INTEGER | Yes | All TXs in T(E) |
| observation_coverage_rate | FLOAT | Yes | observed_tx_count / total_tx_count [0,1] |
| weighted_evidence_quality | FLOAT | Yes | See Section 10 aggregation policy |
| max_evidence_quality | FLOAT | Yes | Highest evidence_quality among observed TXs |
| strongest_tx_txid | VARCHAR(64) | No | TXID with highest evidence_quality |
| computed_at | TIMESTAMPTZ | Yes | When computed |

### 9.9 FusionEvidence

| Field | Type | Required | Description |
|---|---|---|---|
| id | UUID | Yes | Primary key |
| entity_id | UUID | Yes | Entity |
| chain_anomaly_score | FLOAT | Yes | a: normalized IF/LOF output [0,1] |
| network_context_score | FLOAT | Yes | n: EntityNetworkContext.weighted_evidence_quality [0,1] |
| linkage_reliability | FLOAT | Yes | r: heuristic_association_strength, possibly reduced by mixing |
| mixing_flag | BOOLEAN | Yes | Whether mixing detected for this entity |
| raw_logit | FLOAT | Yes | Linear combination before sigmoid |
| investigative_priority_score | FLOAT | Yes | sigmoid(raw_logit) [0,1] |
| chain_only_rank | INTEGER | No | Rank if sorted by chain_anomaly_score only |
| fused_rank | INTEGER | No | Rank when sorted by investigative_priority_score |
| rank_shift | INTEGER | No | chain_only_rank − fused_rank |
| model_version | VARCHAR(32) | Yes | Version of fusion model used |
| w_chain | FLOAT | Yes | Weight for chain anomaly term |
| w_network | FLOAT | Yes | Weight for network context term |
| w_linkage | FLOAT | Yes | Weight for linkage reliability term |
| b | FLOAT | Yes | Bias term |
| computed_at | TIMESTAMPTZ | Yes | When computed |

### 9.10 ExperimentRun / GroundTruthScenario

ExperimentRun: experiment_id (UUID PK), experiment_name, hypothesis (TEXT),
arm (VARCHAR 32), dataset_version, random_seed (INTEGER), split_type
(temporal / entity_disjoint), precision [PENDING], recall [PENDING],
f1 [PENDING], auprc [PENDING], calibration_error [PENDING],
mean_rank_shift [PENDING], notes (TEXT), run_at, status (pending/running/complete/failed).

GroundTruthScenario: scenario_id (UUID PK), scenario_name, scenario_type
(normal / suspicious / mixing / custodial), is_anomalous (BOOLEAN),
entity_ids (UUID[]), txids (VARCHAR[]), data_provenance (VARCHAR 32), seeded_at.



---

## 10. Network Evidence Model

### 10.1 Critical Limitation — State Everywhere

Bitcoin transaction relay (Bitcoin Core >= 0.10) introduces per-peer randomized
propagation delays specifically to defeat timing-based origin inference. A single
observing node's first_seen timestamp is a weak probabilistic signal, not a
transaction-origin determination. Even multi-vantage timing analysis (as studied by
Koshy et al.) exploited structural weaknesses that were subsequently addressed by
relay-delay randomization and Dandelion proposals.

**What a network observation establishes:**
- A propagation event was observed by this node from this peer
- With these timing characteristics

**What a network observation does NOT establish:**
- The originating IP of the transaction
- Whether the peer IP is the broadcaster or a relay
- The identity or intention of any party
- Whether the transaction is anomalous

**Implication for the system:**
Network evidence quality must be represented separately from anomaly signal.
The evidence fusion model must experimentally establish — not assume — what
investigative signal network observations carry.

### 10.2 Per-Transaction Network Evidence Quality

For each transaction t with network observations:

```
evidence_quality(t) =
  f(observer_count, timing_spread_seconds, peer_consistency)

Components:
  observation_coverage(t) = clip(observer_count(t) / N_REF, 0, 1)
    N_REF = configurable reference threshold (default 3)
    Interpretation: fraction of reference observer count achieved

  timing_tightness(t) = 1 - clip(timing_spread_seconds(t) / T_REF, 0, 1)
    T_REF = configurable reference spread (default 120s)
    Interpretation: 1 = all observers agree on timing; 0 = maximum spread

  peer_consistency(t):
    1.0 — all observers report consistent peer for this TXID
    0.5 — minor discrepancy across observers (different peers, overlapping timing)
    0.0 — contradictory reports (significant timing conflict)

  evidence_quality(t) =
    observation_coverage(t) * timing_tightness(t) * peer_consistency(t)

  When no observations exist: evidence_quality(t) = 0.0

Range: [0, 1]
Interpretation: quality/reliability of network propagation observation.
This is NOT a suspiciousness score.
```

**Important:** evidence_quality = 1.0 means the transaction was observed by many
consistent peers with tight timing. It says nothing about whether the transaction
is legitimate or suspicious.

### 10.3 Entity Network Context Aggregation

T(E) = set of all TXIDs where any address in entity E appears as input or output.

```
observed_tx_count(E) = |{t ∈ T(E) : network observations exist for t}|
total_tx_count(E)    = |T(E)|
observation_coverage_rate(E) = observed_tx_count(E) / total_tx_count(E)

weighted_evidence_quality(E):
  = (Σ evidence_quality(t) for t ∈ T(E) with observations)
    / total_tx_count(E)

  Note: unobserved TXs contribute 0 to the numerator
  This is a coverage-weighted mean, not a simple mean of observed TXs

max_evidence_quality(E) = max(evidence_quality(t) for t ∈ T(E)) if any observed else 0

network_context_score(E) = weighted_evidence_quality(E)
```

**Aggregation rationale:**
Coverage-weighted mean penalizes sparse observation. An entity with 1000 transactions
of which 2 are observed gets a lower score than an entity with 5 transactions all
observed. This is correct: the former has much lower network evidence coverage.

**What these fields represent:**
- network_context_score: how well the entity's transaction activity was captured
  in network observations
- This informs the fusion model; the experiment determines whether this context
  adds investigative discriminative power

### 10.4 Network Evidence Requirements

| ID | Requirement | Acceptance Criteria |
|---|---|---|
| NET-001 | Ingest network observations in CSV, JSON, or XML | 100-record batch ingested; each queryable by TXID |
| NET-002 | Reject records missing required fields; log; continue | 10 malformed rows in 100-row batch: 90 stored, 10 rejection log entries |
| NET-003 | Deduplicate on (observed_txid, observing_node_id, peer_ip, first_seen_utc) | 5 duplicate rows produce 5 fewer stored records |
| NET-004 | GeoIP enrichment offline only | After enrichment with network disconnected: all records have geoip_status; no external connection in service logs |
| NET-005 | Compute TransactionNetworkEvidence per TXID | One record per unique TXID in any ingested batch |
| NET-006 | Compute EntityNetworkContext per entity after correlation | EntityNetworkContext record created; weighted_evidence_quality computed correctly |
| NET-007 | No network observation must be labelled as origin identification | UI, API, and logs must use "observed", "peer", "propagation observation" — never "origin IP" or "sender" |

---

## 11. Blockchain / UTXO Model

### 11.1 Data Source

Bitcoin Core in regtest mode provides a fully controlled private chain with real
Bitcoin transaction semantics: actual input/output structure, real UTXO consumption,
real TXID derivation. This validates blockchain-layer parsing and entity resolution.

**Regtest validates:** blockchain semantics, UTXO model, transaction graph.
**Regtest does NOT validate:** real-world Internet propagation, network observation
behavior, adversarial CoinJoin at scale.

### 11.2 UTXO Model

```
Transaction T1 creates output T1:vout0 → UTXO {txid:T1, vout:0, address:A, value:X}
  → UTXO is_spent = false

Transaction T2 spends T1:vout0 as input
  → TransactionInput {txid:T2, prev_txid:T1, prev_vout:0}
  → UTXO {txid:T1, vout:0} marked is_spent=true, spent_by_txid=T2

Each UTXO is consumed exactly once. This constraint is what makes
common-input-ownership inference possible and what CoinJoin exploits to break it.
```

### 11.3 Requirements

| ID | Requirement | Acceptance Criteria |
|---|---|---|
| BC-001 | Parse Bitcoin transactions: inputs, outputs, amounts, fee | Multi-input regtest TX: correct TransactionInput and TransactionOutput records; amounts sum to input - fee |
| BC-002 | Reconstruct UTXO state correctly | UTXO marked is_spent=true when consumed; spent_by_txid set correctly |
| BC-003 | Coinbase transactions never used in ownership heuristics | Coinbase TX: is_coinbase=true; excluded from common-input clustering |
| BC-004 | All data provenance fields set correctly | Every ingested record has data_provenance field set before storage |

---

## 12. Entity Resolution

### 12.1 Definition

An entity cluster is a set of Bitcoin addresses inferred to share a common controller.
This is a probabilistic heuristic inference. It is not an identity proof. It is not
a legal determination. Use: "probable entity", "candidate common-controller group",
"heuristically linked addresses". Never use: "wallet owner", "person", "identified
individual".

### 12.2 Common-Input-Ownership Heuristic

**Basis:** Ron & Shamir (2013) [SOURCE VERIFY: Financial Cryptography 2013].
If multiple distinct addresses appear as inputs in the same transaction, infer common
controller. Implemented using patterns from established tools (bitcoingraph / GraphSense
as reference implementation; TraceLayer maintains its own canonical data model and
integration — not a direct dependency on any specific repository state).

**Failure modes — both must be handled:**

| Failure | Mechanism | Handling |
|---|---|---|
| CoinJoin | Multiple unrelated parties contribute inputs to one transaction | Mixing detection (Section 19) reduces heuristic_association_strength |
| Custodial consolidation | An exchange batches many customer deposits | Service-pattern flag (Section 20) reduces heuristic_association_strength |
| Change-address misidentification | Heuristic incorrectly tags a non-change output as change | Applied with low confidence in ambiguous cases; stated as heuristic |

**Distinction between CoinJoin and custodial consolidation:**
- CoinJoin: deliberate mixing by unrelated parties to create ambiguous ownership
- Custodial consolidation: a service legitimately controlling many user funds in batch

These produce similar multi-input patterns but have different implications.
They must be detected and handled separately.

### 12.3 Change-Address Heuristic

Applied as a supplementary signal where applicable. Not applied to mixing transactions.
Outputs flagged as is_change_candidate = true carry is_change_candidate in their record
and the inference is visible in the evidence card. This heuristic is unreliable in many
transaction types; it is applied with low confidence and explicitly stated as heuristic.

### 12.4 Requirements

| ID | Requirement | Acceptance Criteria |
|---|---|---|
| ENT-001 | Apply common-input-ownership heuristic to all multi-input non-coinbase non-mixing TXs | 3-input synthetic TX produces one entity cluster; addresses linked in Neo4j |
| ENT-002 | Never apply common-input heuristic to coinbase transactions | Coinbase TX: no entity linkage created from coinbase inputs |
| ENT-003 | Detect and flag mixing TXs before applying entity linkage (Section 19) | Synthetic Whirlpool-candidate TX: mixing_flag=true; heuristic_association_strength reduced |
| ENT-004 | Detect and flag service-pattern (high-input-count) TXs (Section 20) | High-input-count TX: service_pattern_flag=true; heuristic_association_strength reduced |
| ENT-005 | heuristic_association_strength is visible in evidence card with its basis stated | Judge can see the field value and the heuristic that produced it |

---

## 13. Evidence Correlation

### 13.1 The TXID Join — Starting Point, Not Contribution

The fundamental correlation is: network observations share observed_txid with
blockchain transactions. This is a necessary join, not the system's contribution.

**TraceLayer's contribution above the join:**
1. Per-TX network evidence quality computation (Section 10.2)
2. Entity-level network context aggregation with explicit coverage policy (Section 10.3)
3. Separation of evidence quality from anomaly signal (Section 16 design)
4. Evidence fusion model combining evidence streams (Section 16)
5. Rank shift measurement establishing experimental value of network evidence
6. Evidence subgraph with full provenance from observation to entity

### 13.2 TXID Correlation Requirements

| ID | Requirement | Acceptance Criteria |
|---|---|---|
| CORR-001 | Every network observation is linked to a Transaction record if TXID matches | A TXID present in both tables: NetworkObservation has FK to Transaction |
| CORR-002 | Unknown TXIDs in network observations are stored but flagged for later correlation | needs_blockchain_correlation=true; no rejection; correlated when TX is later ingested |
| CORR-003 | EntityNetworkContext computed after entity resolution is complete | After Phase 4 entity resolution: all entities have EntityNetworkContext records including entities with zero observations |
| CORR-004 | Zero-observation entities have network_context_score = 0.0, not NULL | Entity with no network observations: network_context_score=0.0; network_evidence_present=false |

### 13.3 Investigative Ranking Logic

**Chain-only ranking:** Sort entities by chain_anomaly_score descending.
**Fused ranking:** Sort entities by investigative_priority_score descending.
**Rank shift:** chain_only_rank − fused_rank
- Positive: entity moved up (fused evidence elevated it)
- Negative: entity moved down
- Zero: no change
The experiment measures whether this shift correlates with ground truth labels.

---

## 14. Feature Engineering

### 14.1 Chain-Side Graph Features (MVP)

These features are computed by the Python intelligence service from entity and
transaction data. They capture graph-structural anomaly signals.

| Feature | Type | Description | Anomaly Intuition |
|---|---|---|---|
| address_count | Integer | Addresses in cluster | Very large or very small may be notable |
| tx_count | Integer | Total transactions | Very high volume |
| input_tx_count | Integer | TXs where entity appears as input | Outgoing activity |
| output_tx_count | Integer | TXs where entity appears as output | Incoming activity |
| input_to_output_ratio | Float | input_tx_count / output_tx_count | Directionality imbalance |
| address_degree_avg | Float | Average connections per address in graph | Hub-like behavior |
| address_degree_max | Float | Maximum connections per address | Single high-degree address |
| hop_depth | Integer | Longest chain of TXs through the entity's graph | Deep layering |
| address_reuse_rate | Float | Fraction of addresses used more than once | High = attributable; low = privacy-conscious |
| heuristic_association_strength | Float | Strength of clustering evidence | Low = weak linkage |
| mixing_exposure | Boolean | Any TX has mixing_flag=true | Linkage weakened |
| service_pattern_flag | Boolean | High-input consolidation pattern | Possible custodial |

### 14.2 Network-Side Context Features (for fusion)

| Feature | Type | Description |
|---|---|---|
| network_context_score | Float | weighted_evidence_quality(E) [0,1] |
| observation_coverage_rate | Float | observed_tx_count / total_tx_count [0,1] |
| max_evidence_quality | Float | Strongest per-TX evidence quality |
| observed_tx_count | Integer | How many entity TXs have network observations |

**Critical:** These features describe network observation coverage and quality,
not suspiciousness. The fusion model learns (experimentally) whether and how these
features correlate with anomaly ground truth labels.

### 14.3 node2vec Features (Conditional)

node2vec graph embeddings will be evaluated only if:
(a) the ablation experiment (Experiment 8) shows >2 F1-point improvement over
hand-crafted graph features alone on the held-out seeded test set, AND
(b) the additional latency is acceptable.

Default MVP: hand-crafted graph features only.

---

## 15. Anomaly Detection

### 15.1 Model Selection

| Candidate | Basis | Selection |
|---|---|---|
| Isolation Forest (IF) | sklearn.ensemble.IsolationForest; effective on tabular data; fast inference | Primary candidate |
| Local Outlier Factor (LOF) | sklearn.neighbors.LocalOutlierFactor; density-based; good on clustered data | Secondary candidate |
| Naive z-score threshold | Z-score on tx_count and address_degree_avg | Baseline only |

Selection: Both IF and LOF will be trained on the same dataset. Model with higher
F1 on held-out seeded ground truth will be used in the fused system.
Selection rationale must be documented in ExperimentRun.

### 15.2 Anomaly Score Normalization

Raw IF/LOF scores are normalized to [0,1] using the training set score distribution:
```
chain_anomaly_score = clip((raw_score - min_train) / (max_train - min_train), 0, 1)
```
This normalized score is NOT a calibrated probability. It is an anomaly index.

### 15.3 Explainability

Top-3 contributing features per entity identified from IF decision paths or
LOF neighborhood analysis. These must be visible in the evidence card with actual
feature values — not just feature names.

### 15.4 Requirements

| ID | Requirement | Acceptance Criteria |
|---|---|---|
| ML-001 | Train IF and LOF; select by F1 on held-out seeded test; document in ExperimentRun | ExperimentRun record with both F1 values; selection recorded |
| ML-002 | chain_anomaly_score in [0,1] for all entities | No entity has score outside [0,1] |
| ML-003 | Top-3 contributing features with actual values available via API | GET /entities/{id} returns top3_features array with name and value |
| ML-004 | Entity-disjoint or temporal split enforced; no random split | ExperimentRun.split_type = entity_disjoint or temporal |
| ML-005 | Random seed fixed and recorded | ExperimentRun.random_seed populated for every run |

---

## 16. Evidence Fusion Model

### 16.1 Design Principle

**What v1 got wrong:** The v1 formula treated network evidence as a direct additive
term toward suspiciousness. This is logically invalid: observation quality does not
equal criminal signal. A widely-observed transaction is not more suspicious by virtue
of being widely observed.

**The corrected design:**
The evidence fusion model is a logistic regression over multiple evidence signals.
The weights are learned from seeded ground truth. The experiment determines whether
network context is a useful predictor — it is not assumed to be.

If the experiment shows network context is not a useful predictor (i.e., w_network ≈ 0
after training), that is a valid scientific result. We report it honestly.

### 16.2 Evidence Fusion Formula

```
investigative_priority_score(E) = σ( w_chain · a(E)
                                    + w_network · n(E)
                                    + w_linkage · r(E)
                                    + b )

where σ(x) = 1 / (1 + exp(-x))

a(E) = chain_anomaly_score(E)         [0,1] — chain-side IF/LOF output
n(E) = network_context_score(E)       [0,1] — EntityNetworkContext.weighted_evidence_quality
r(E) = linkage_reliability(E)         [0,1] — heuristic_association_strength, possibly
                                              reduced by mixing (Section 19)
b    = bias term

w_chain, w_network, w_linkage, b: learned weights from logistic regression
```

This is a **logistic evidence-fusion model**, not "Platt scaling."

Platt scaling is a specific technique for calibrating a *single* classifier score.
What we have is logistic regression over multiple evidence features. If a separate
calibration step is needed after this model, it will be described and justified
separately in Section 17.

**What each weight encodes:**

| Weight | Encodes |
|---|---|
| w_chain | How much chain-side anomaly signal predicts investigative relevance |
| w_network | Whether and how much network context predicts investigative relevance |
| w_linkage | How entity cluster strength affects the prediction |
| b | Baseline intercept |

**The experiment (Section 26) establishes:**
- Does w_network differ significantly from zero?
- Does including network context improve F1/AUPRC over chain-only model?
- If w_network ≈ 0: H0 holds; network evidence adds no measurable value.

### 16.3 Mixing Signal Handling

When mixing_flag = true for an entity:
- heuristic_association_strength is reduced (Section 19 formula)
- r(E) in the fusion formula reflects reduced linkage reliability
- Evidence card explicitly states the reduction and its cause

**This is NOT a penalty on suspiciousness.** It is correct representation of
uncertainty: when mixing breaks our clustering assumption, we honestly reduce
the confidence in the entity cluster itself.

### 16.4 Requirements

| ID | Requirement | Acceptance Criteria |
|---|---|---|
| FUS-001 | investigative_priority_score ∈ [0,1] for all entities | No entity score outside [0,1] |
| FUS-002 | Fusion model weights learned from seeded ground truth; not hand-picked | w_chain, w_network, w_linkage, b come from logistic regression fit |
| FUS-003 | FusionEvidence record stores all components and weights | Every entity has FusionEvidence with all fields populated |
| FUS-004 | Chain-only rank and fused rank both computed and stored | chain_only_rank and fused_rank populated; rank_shift computed |
| FUS-005 | Evidence card shows decomposed formula with actual values | Judge sees a(E), n(E), r(E), w values, raw logit, and final score |

---

## 17. Calibration

### 17.1 What Requires Calibration

The logistic fusion model produces scores in [0,1] that may be interpreted as
investigative priority. If we want to claim these scores are approximately calibrated
probabilities (i.e., entities scored 0.7 are truly anomalous ~70% of the time on
our ground truth), we must evaluate this explicitly.

**We do not call a score "calibrated" until calibration has been evaluated.**

### 17.2 Methodology

Post-training calibration evaluation:
1. Fit logistic fusion model on training split
2. Compute scores on held-out calibration test split (entity-disjoint from training)
3. Group predictions into probability bins
4. Plot reliability diagram: predicted probability vs. observed frequency per bin
5. Compute Expected Calibration Error (ECE) or similar metric
6. Compute Brier score as overall quality measure

**We do NOT use an arbitrary universal threshold (e.g., "Brier > 0.25 = bad").
Instead:** Report the Brier score, the reliability diagram, and the ECE. Compare
against the Brier score of a naive baseline (e.g., class prevalence as constant
prediction). Interpret in context of class prevalence in seeded ground truth.

The fusion model's logistic regression already produces sigmoid outputs, which
tends to be approximately calibrated when trained on sufficient labeled data. We
evaluate whether this holds on our synthetic seeded ground truth and report what
we find.

### 17.3 What Calibration Proves and Does Not Prove

| Proves | Does Not Prove |
|---|---|
| Internal consistency with seeded synthetic ground truth | Real-world accuracy on actual Bitcoin transactions |
| Basis for the word "calibrated" in the UI | Generalization to unseen transaction patterns |
| That scores are not arbitrary | That the synthetic distribution matches real Bitcoin behavior |

**Required scope statement whenever citing calibration:**
"Calibration is evaluated against seeded synthetic ground truth. Scores reflect
approximate consistency with our synthetic distribution; real-world calibration is unknown."

---

## 18. Uncertainty Model

### 18.1 Five Distinct Quantities

These must never be conflated in code, UI, or verbal answers:

| Quantity | Field | Meaning | Calibrated probability? |
|---|---|---|---|
| Chain-side anomaly index | chain_anomaly_score | Normalized IF/LOF output | No |
| Network evidence quality | evidence_quality | Observation reliability index | No |
| Network context score | network_context_score | Coverage-weighted quality per entity | No |
| Heuristic association strength | heuristic_association_strength | Cluster heuristic evidence | No — heuristic only |
| Investigative priority score | investigative_priority_score | Logistic fusion output | Evaluated via calibration |

### 18.2 Uncertainty Propagation Rules

| Condition | Effect |
|---|---|
| Zero network observations for entity | network_context_score = 0.0; network_evidence_present = false; stated in evidence card |
| Mixing detected | heuristic_association_strength reduced; linkage_reliability reduced; limitation notice in evidence card |
| Service-pattern flag | heuristic_association_strength reduced; limitation notice in evidence card |
| Single observer only | evidence_quality ≤ clip(1/N_REF,0,1); stated as weak evidence in evidence card |
| Conflicting observations | peer_consistency ≤ 0.5; propagated to evidence_quality |
| Private IP range | geoip_status=private_range; observation counted; geo context unavailable |

### 18.3 Evidence Card Mandatory Elements

```
ENTITY: [entity_id]
INVESTIGATIVE PRIORITY: [investigative_priority_score — real value]

CHAIN EVIDENCE
  Chain anomaly index:          [chain_anomaly_score — real value]
  Top contributing features:    [feature_name: actual_value] (top 3)

NETWORK CONTEXT
  Network context score:        [network_context_score — real value]
  Observation coverage:         [observed_tx_count] / [total_tx_count] TXs observed
  Max evidence quality:         [max_evidence_quality — real value]
  Strongest observed TX:        [txid or "none"]
  Network evidence present:     [true / false]

LINKAGE RELIABILITY
  Heuristic association:        [heuristic_association_strength — real value]
  Cluster method:               [common_input / change_address / combined]
  Mixing exposure:              [true / false]
  Service pattern flag:         [true / false]

FUSION FORMULA
  investigative_priority = σ([w_chain]·[a] + [w_network]·[n] + [w_linkage]·[r] + [b])
                         = σ([raw_logit]) = [investigative_priority_score]

LIMITATIONS
  [All applicable — mixing, low coverage, heuristic uncertainty, synthetic data]

CALIBRATION STATUS: [Evaluated — see calibration view] OR [PENDING calibration run]
DATA PROVENANCE: [provenance of underlying observations]
```

---

## 19. CoinJoin / Mixing Handling

### 19.1 What We Do and Do Not Claim

**We detect structural mixing patterns.** This does not prove criminal intent.
CoinJoin is a legitimate Bitcoin privacy technique. We detect when a transaction
matches structural signatures associated with mixing, and we reduce linkage
confidence accordingly.

**We do not defeat or deanonymize CoinJoin.**

### 19.2 Structural Signatures [VERIFY before hard-coding]

The following signatures are based on published descriptions of known CoinJoin
implementations. All thresholds are labeled [VERIFY] — Prachi must confirm
exact applicability from primary sources before implementation encodes them as
hard requirements.

| Pattern | Structural Signature | Source | Status |
|---|---|---|---|
| Whirlpool-style | Fixed set of inputs/outputs; all outputs equal fixed denomination (0.5, 0.05, 0.01, 0.001 BTC ± fee tolerance); equal input and output counts (5 or 8) | Ficsor / Gavenda et al. 2025 [VERIFY] | [VERIFY exact thresholds] |
| Wasabi 1.x-style | ≥10 equal-value outputs; most frequent value ~0.1 BTC ± tolerance; ≥2 distinct output values; input count ≥ occurrences of modal output value | Schnoering & Vazirgiannis 2023 [VERIFY] | [VERIFY exact thresholds] |
| Generic structural | Output count > input UTXO count; high frequency of duplicate output values | Literature-derived | Apply as low-confidence flag |

**mixing_type** values use _candidate suffix: whirlpool_candidate, wasabi_v1_candidate,
generic_candidate. These are structural pattern matches, not confirmed identifications.

### 19.3 Effect on Entity Linkage

When mixing_flag = true for a transaction that any entity address participates in:

```
linkage_reliability(E) = heuristic_association_strength(E) × mixing_reduction_factor

mixing_reduction_factor: [PENDING — determined by experiment or set to configurable
default, e.g., 0.3, meaning linkage_reliability = 0.3 × heuristic_association_strength]
```

This reduction is propagated into the fusion model's r(E) term.

### 19.4 Two Separate Measurements

CoinJoin handling requires two distinct capability evaluations (Experiment 4):
1. **Mixing detection precision/recall:** How accurately does structural pattern
   matching identify synthetic mixing transactions? (separate from entity linkage)
2. **Entity linkage degradation under mixing:** How much does linkage accuracy change
   when mixing transactions are present? This is an empirical result, not a target.

Published literature (Gavenda et al. 2025) reports linkage accuracy degradation
under mixing. We measure our own result and compare against literature after the
experiment — we do not define a target degradation in advance.

### 19.5 Requirements

| ID | Requirement | Acceptance Criteria |
|---|---|---|
| MIX-001 | Detect structural mixing patterns for all ingested transactions | Synthetic Whirlpool-candidate TX: mixing_flag=true, mixing_type=whirlpool_candidate |
| MIX-002 | Reduce linkage_reliability for entities involved in mixing TXs | Entity touching mixing TX has linkage_reliability < heuristic_association_strength |
| MIX-003 | Evidence card states mixing exposure and its effect explicitly | Mixing flag shown; linkage reliability reduction stated; limitation notice visible |
| MIX-004 | Mixing detection never stated as proof of criminal intent in any UI text | Audit all UI strings |
| MIX-005 | Literature thresholds verified before encoding as hard requirements | Prachi: confirm Whirlpool/Wasabi thresholds from primary sources |

---

## 20. Custodial / Service Patterns

### 20.1 Definition

A custodial entity (exchange, payment processor) may legitimately control inputs
from many unrelated users in a single transaction. This creates a multi-input pattern
that the common-input heuristic would interpret as a single entity — producing a false
merge of many unrelated users.

Custodial consolidation and CoinJoin produce similar patterns but have different causes.

### 20.2 Detection

```
service_pattern_flag(TX) = true when:
  input_count(TX) > HIGH_INPUT_THRESHOLD (default: 20, configurable)

service_pattern_flag(E) = true when:
  any TX in T(E) has service_pattern_flag = true
```

When service_pattern_flag = true on a TX:
- heuristic_association_strength is reduced for all entities whose addresses
  participated in that TX via the common-input heuristic
- Evidence card shows limitation notice
- Analyst is warned to verify whether this represents a service

### 20.3 Requirements

| ID | Requirement | Acceptance Criteria |
|---|---|---|
| CUST-001 | Flag transactions with input_count > HIGH_INPUT_THRESHOLD | Synthetic 25-input TX: service_pattern_flag=true on TX |
| CUST-002 | Propagate flag to entity records | Entity containing addresses from flagged TX: service_pattern_flag=true |
| CUST-003 | Reduce linkage reliability for service-flagged entities | heuristic_association_strength reduced when service_pattern_flag=true |
| CUST-004 | Evidence card shows service pattern warning | Analyst sees limitation notice |

---

## 21. Evidence Graph

### 21.1 Why Neo4j

The core investigative query — "how does this network observation connect to this
entity, through how many hops, with what confidence?" — is a native graph traversal.
The same query in PostgreSQL requires complex multi-way self-joins.

**Neo4j is the evidence model, not primarily a visualization backend.**
Cytoscape.js is the visualization frontend. These must not be conflated.

### 21.2 Node Schema

| Label | Key Properties |
|---|---|
| :NetworkObservation | observed_txid, peer_ip, observing_node_id, first_seen_utc, evidence_quality, data_provenance |
| :Transaction | txid, input_count, output_count, mixing_flag, mixing_type, data_provenance |
| :UTXO | txid, vout, value_satoshis, is_spent |
| :Address | address |
| :Entity | entity_id, chain_anomaly_score, network_context_score, investigative_priority_score, heuristic_association_strength, mixing_exposure, service_pattern_flag |

### 21.3 Relationship Schema

| Relationship | From → To | Key Properties |
|---|---|---|
| :NETWORK_OBSERVED | :NetworkObservation → :Transaction | evidence_quality, peer_consistency, created_at, source_component |
| :HAS_INPUT | :Transaction → :UTXO | input_index, created_at, source_component |
| :HAS_OUTPUT | :Transaction → :UTXO | vout, value_satoshis, created_at, source_component |
| :LOCKS_TO | :UTXO → :Address | created_at, source_component |
| :BELONGS_TO | :Address → :Entity | heuristic_association_strength, cluster_method, created_at, source_component |
| :PARTICIPATES_IN | :Entity → :Transaction | role (input/output/both), created_at, source_component |

### 21.4 Provenance Requirements

Every edge must carry: created_at, source_component (which service/rule produced it),
evidence_type (heuristic/model/observation/derived), confidence where applicable.

**Why:** An analyst presenting a lead must be able to prove when and how each
evidence edge was derived. Post-creation modification of evidence edges is prohibited.

### 21.5 Requirements

| ID | Requirement | Acceptance Criteria |
|---|---|---|
| GRAPH-001 | Evidence subgraph extractable per entity within 2 hops | Query returns all nodes/edges within 2 hops in < 2 seconds on prototype dataset |
| GRAPH-002 | Every edge has provenance fields | Judge can ask "why does this edge exist?" and get a specific, verifiable answer |
| GRAPH-003 | Graph populated from real backend data only | No hardcoded demo graph; all nodes created by backend pipeline |
| GRAPH-004 | ADR-004 benchmark pending | Run Neo4j 3-hop query vs PostgreSQL self-join benchmark before national submission to quantify the performance justification [PENDING] |

**Evidence subgraph Cypher:**
```cypher
MATCH path = (e:Entity {entity_id: $entity_id})-[*1..2]-(n)
RETURN path LIMIT 500
```

---

## 22. API Contracts

Base: http://localhost:8080/api/v1 · Content-Type: application/json

Standard error: `{"error":{"code":"string","message":"string","field":"string|null"}}`

| ID | Method | Route | Purpose |
|---|---|---|---|
| API-001 | POST | /ingest/network | Ingest network observations; data_provenance required |
| API-002 | POST | /ingest/transactions | Ingest blockchain transactions; data_provenance required |
| API-003 | GET | /batches/{id} | Batch status; accepted/rejected counts |
| API-004 | GET | /batches/{id}/rejections | Paginated rejection log with field-level reasons |
| API-005 | GET | /transactions/{txid} | Full TX detail with inputs, outputs, mixing status, network evidence |
| API-006 | GET | /network/evidence/{txid} | TransactionNetworkEvidence for a TXID |
| API-007 | GET | /entities/{id} | Full entity: all scores, top3_features, EntityNetworkContext, limitations |
| API-008 | GET | /entities | Paginated entity list; sort_by=investigative_priority_score or chain_anomaly_score |
| API-009 | GET | /entities/{id}/transactions | All TXs for entity |
| API-010 | POST | /detection/run | Run detection on batch; 202 + job_id |
| API-011 | GET | /detection/results | Paginated DetectionResults with chain_only_rank and fused_rank |
| API-012 | GET | /evidence/subgraph/{id}?hops=2 | Evidence subgraph: nodes, edges, provenance, evidence_summary |
| API-013 | GET | /evidence/compare/{id} | chain_only vs fused: both ranks, rank_shift, what changed |
| API-014 | POST | /experiments/run | Run experiment; 202 + experiment_id |
| API-015 | GET | /experiments/{id} | Full ExperimentRun record |
| API-016 | GET | /experiments | List experiments by status |
| API-017 | GET | /health | {"status":"healthy/degraded/unhealthy","services":{...},"schema_version":"..."} |

**API-013 response (core comparison endpoint):**
```json
{
  "entity_id": "...",
  "chain_only": {
    "rank": "[computed]",
    "chain_anomaly_score": "[computed]",
    "top_features": [{"name":"...","value":"..."}]
  },
  "fused": {
    "rank": "[computed]",
    "investigative_priority_score": "[computed]",
    "components": {
      "chain_anomaly_score": "[computed]",
      "network_context_score": "[computed]",
      "linkage_reliability": "[computed]"
    }
  },
  "rank_shift": "[computed — positive means network evidence elevated this entity]",
  "network_context": {
    "observed_tx_count": "[computed]",
    "total_tx_count": "[computed]",
    "observation_coverage_rate": "[computed]",
    "weighted_evidence_quality": "[computed]"
  }
}
```

All placeholder values labeled [computed] must come from real backend computation.
No hardcoded example values may appear in any response at demo time.

---

## 23. Frontend Requirements

### 23.1 Core Design Principle

Every number displayed comes from the API. No frontend-fabricated values.
The comparison toggle (chain-only vs fused) calls the real API; it does not
simulate a ranking difference.

### 23.2 Core Surfaces (Priority Order)

**Surface 1 — Investigation Dashboard**
Ranked entity list sorted by investigative_priority_score.
Per row: entity_id, score, chain_anomaly_score, network_context_score,
mixing_exposure, service_pattern_flag, observation_coverage_rate.
Sort toggle: chain-only view vs fused view (calls API; re-ranks).

**Surface 2 — Entity / Evidence Detail**
Full entity evidence card per Section 18.3.
Fusion formula decomposed with actual values.
Top-3 anomaly features with actual values.
Network context summary.
Mixing and service-pattern limitation notices.
Calibration status.
Data provenance.

**Surface 3 — Evidence Graph**
Cytoscape.js renders the evidence subgraph from GET /evidence/subgraph/{id}.
Nodes: NetworkObservation → Transaction → UTXO → Address → Entity.
Edge tooltips show provenance fields.
No hardcoded graph; all nodes from real backend.

**Surface 4 — Comparison / Experiment View**
Chain-only rank vs fused rank for any entity (GET /evidence/compare/{id}).
Experiment results table: all ExperimentRun records with real F1/precision/recall.
Calibration reliability diagram from real experiment data.
If experiment not run: display "PENDING" — not a placeholder metric.

Secondary surfaces (can be subviews/panels, not separate pages):
- Transaction detail (inputs, outputs, mixing status, network evidence)
- Batch ingestion status / rejection log
- Network observation list per TXID

### 23.3 Prohibited Frontend Behaviours

- Displaying hardcoded/fabricated numbers in demo context
- Simulating ranking or score changes without API call
- Showing placeholder experiment results as real values
- Labelling any IP as "origin IP" or "sender IP"
- Labelling any entity cluster as a person or identity

---

## 24. Synthetic Data / Propagation Generator

### 24.1 Purpose and Scope

Because we cannot claim global Bitcoin network vantage, we build a controlled
synthetic propagation-observation generator for experiment design.

This is NOT real Bitcoin network telemetry.
All generated records have data_provenance = SYNTHETIC.
All experiments using synthetic data are explicitly scoped as synthetic.

### 24.2 Generator Design

```
Input:
  known_txid (from synthetic blockchain scenario)
  scenario_type
  rng_seed (for reproducibility)

Scenario types for network observations:
  STRONG:       5-8 observers; tight timing (< 5s); high peer_consistency
  NORMAL:       3-5 observers; moderate timing (5-30s); consistent peers
  SPARSE:       1-2 observers; any timing
  DELAYED:      2-4 observers; wide timing (30-120s)
  CONFLICTING:  observers report contradictory peers; peer_consistency = 0
  NOISY:        extra spurious duplicate-like records
  ABSENT:       no network observations for this TXID

Output: NetworkObservation records; data_provenance = SYNTHETIC

Ground truth stored in GroundTruthScenario:
  scenario_type, is_anomalous, entity_ids, txids, seeded_at
```

### 24.3 Critical Design Requirement

**Avoid baking the label into the network data.**

The synthetic generator must NOT systematically assign STRONG network scenarios
to anomalous entities and ABSENT/SPARSE to normal entities. If it does, the
experiment will appear to show that network evidence helps when in fact it is
recovering an encoding artifact.

**Correct approach:**
1. Generate blockchain scenarios with ground truth labels (normal/anomalous)
2. Independently vary network observation scenarios for each TXID
3. Ensure multiple entities share the same blockchain ground truth label
   but differ in their network observation scenario
4. This allows measurement of network context's discriminative power while
   controlling for blockchain features

The permutation control (Experiment 2) provides additional validation.

---

## 25. Dataset Provenance

Data provenance is a first-class field in every evidence record.

| Value | Meaning |
|---|---|
| REAL_NODE_CAPTURE | Captured from actual Bitcoin network nodes |
| REAL_PUBLIC | Public dataset (e.g., Elliptic++) |
| SYNTHETIC | Generated by TraceLayer's synthetic data generator |
| DERIVED | Computed from another TraceLayer record (e.g., entity cluster from TX) |

**Rules:**
- Every NetworkObservation, Transaction, and GroundTruthScenario must have data_provenance set before storage
- No ingestion accepted without data_provenance field
- UI must display data_provenance on every evidence card and graph node
- Demo must state clearly when synthetic data is being shown

**Bitcoin Core regtest provenance:**
Transactions from Bitcoin Core regtest are data_provenance = SYNTHETIC (controlled).
They provide real Bitcoin transaction semantics but are not drawn from the real Bitcoin
network. Do not describe regtest transactions as "real Bitcoin data."

---

## 26. Experiment Design

**ALL results are PENDING. No result may be presented before it is computed.**

### Experiment 1 — Chain-Only vs Fused (Core Ablation) [MUST]

| Field | Value |
|---|---|
| Hypothesis | H1: Adding network context to chain features improves investigative ranking over chain-only analysis |
| Dataset | Seeded synthetic; entity-disjoint or temporal split |
| Arm A | Naive z-score baseline (tx_count, address_degree_avg thresholds) |
| Arm B | Chain graph features only → IF/LOF → rank by chain_anomaly_score |
| Arm C | Chain graph features + network context features → logistic fusion model → rank by investigative_priority_score |
| Key metric | F1, AUPRC, mean rank shift of truly anomalous entities |
| Interpretation if H1 | Arm C F1 > Arm B F1; positive mean rank shift for anomalous entities |
| Interpretation if H0 | Arm C F1 ≈ Arm B F1; rank shift ≈ 0; w_network ≈ 0 |
| Honest reporting | Both outcomes valid; H0 result demonstrates experimental rigor |
| Status | **[PENDING]** |

### Experiment 2 — Network Permutation Control [MUST]

| Field | Value |
|---|---|
| Purpose | Establish that network evidence association with investigative outcome is genuine, not a generator artifact |
| Method | Run Arm C with real synthetic network assignments; then run again with network context features randomly reassigned across entities (shuffle) |
| Comparison | Real association F1 vs shuffled F1 |
| Interpretation if genuine | Real assignment outperforms shuffled |
| Interpretation if artifact | Real ≈ shuffled; generator was encoding the label |
| Status | **[PENDING]** |

### Experiment 3 — Calibration Evaluation [MUST]

| Field | Value |
|---|---|
| Purpose | Determine whether investigative_priority_score is approximately calibrated |
| Method | Held-out calibration test split (entity-disjoint from training); reliability diagram; ECE; Brier score |
| Baseline | Brier score of naive baseline (class prevalence as constant prediction) |
| Reporting | Reliability diagram + ECE + Brier score vs baseline; state class prevalence |
| Status | **[PENDING]** |

### Experiment 4 — CoinJoin / Mixing Stress Test [MUST]

| Field | Value |
|---|---|
| Measurement 1 | Mixing detection precision/recall on synthetic mixing TXs |
| Measurement 2 | Entity linkage accuracy before vs after mixing TXs in ground truth |
| Expected | Linkage accuracy degrades under mixing; exact magnitude is empirical |
| Post-experiment | Compare against published literature (Gavenda et al. 2025) after measurement |
| Do not pre-define | Target degradation percentage — measure first, compare after |
| Status | **[PENDING]** |

### Experiment 5 — Custodial Consolidation Stress Test [SHOULD]

| Field | Value |
|---|---|
| Metric | False merge rate: fraction of distinct true entities incorrectly merged into one cluster |
| Baseline | False merge rate without service_pattern_flag |
| Experimental | False merge rate with service_pattern_flag and linkage reduction |
| Status | **[PENDING]** |

### Experiment 6 — Chain-Only External Validation [SHOULD]

| Dataset | Elliptic / Elliptic++ (chain-side only; no network telemetry) |
| Scope | Chain-side entity/address anomaly detection only; NOT network fusion validation |
| Split | Temporal (time-step based per Elliptic++ documentation) |
| Metrics | F1, AUPRC, precision, recall — not raw accuracy (class imbalance) |
| Required scope statement | "Elliptic++ validates chain-side detection only. It cannot validate network evidence fusion, which has no network telemetry." |
| Status | **[PENDING]** |

### Experiment 7 — Network-Only Baseline [SHOULD]

| Purpose | Establish whether network context features alone provide any discriminative power |
| Method | Use only EntityNetworkContext features → anomaly detection |
| Comparison | Network-only F1 vs chain-only (Exp 1 Arm B) vs fused (Exp 1 Arm C) |
| Status | **[PENDING]** |

### Experiment 8 — node2vec vs Hand-Crafted Features Ablation [OPTIONAL]

| Decision rule | Include node2vec only if F1 improvement > 2 points AND latency acceptable |
| Status | **[PENDING — do not implement node2vec before this experiment]** |

---

## 27. Evaluation Metrics

| Metric | Definition | Use |
|---|---|---|
| Precision@K | Fraction of top-K ranked entities that are truly anomalous | Investigative relevance of the ranked list |
| Recall | Fraction of truly anomalous entities flagged above threshold | Coverage |
| F1 | Harmonic mean of precision and recall | Overall detection quality |
| AUPRC | Area under precision-recall curve | Performance across thresholds; better than AUC-ROC under class imbalance |
| Mean rank shift | Mean(chain_only_rank − fused_rank) for truly anomalous entities | Direct measure of whether network evidence elevates relevant leads |
| Brier score | Mean squared error between score and binary label | Calibration quality |
| ECE | Expected calibration error across probability bins | Reliability diagram quality |
| False merge rate | Fraction of distinct true entities merged into one cluster | Entity resolution accuracy under adversarial conditions |
| Mixing detection F1 | F1 of structural mixing pattern detection | Mixing detector quality |

**Never use raw accuracy as primary metric.** Under class imbalance (anomalous
entities are rare), accuracy is dominated by the majority class and is misleading.

---

## 28. Leakage Prevention

| Rule | Implementation |
|---|---|
| No random splits | split_type must be entity_disjoint or temporal; random splits prohibited |
| Entity-disjoint | No entity appears in both training and test sets |
| Temporal | All training entities are from earlier time steps than test entities |
| Feature timing | No feature may use information only available after the label is assigned |
| Calibration holdout | Calibration evaluation uses a third split, separate from training and validation |
| Fixed seed | Every experiment has a fixed, recorded random_seed in ExperimentRun |
| Generator independence | Synthetic generator does not use entity labels when creating network scenarios |

---

## 29. External Validation

Elliptic / Elliptic++ provides labeled Bitcoin transaction/entity data.

**What it validates:** Chain-side anomaly detection component.
**What it does NOT validate:** Network evidence fusion (Elliptic++ contains no network telemetry).

Required scope statement in all presentations, slides, and verbal answers:
> "Elliptic++ validates our chain-side detection component on real labeled data.
> It cannot and does not validate network-layer evidence fusion, which is evaluated
> only on our seeded synthetic scenarios."

Dataset compatibility check [PENDING]:
- Verify label format matches our entity/transaction schema
- Verify temporal structure for temporal split
- Compute class prevalence
- Confirm evaluation protocol per dataset documentation

---

## 30. Security / Offline Deployment

### 30.1 Offline Architecture (Hard Requirement)

All services, databases, ML models, GeoIP data must operate with zero runtime
internet dependency. This is demonstrable live.

### 30.2 Docker Compose (Canonical)

```yaml
services:
  postgres:
    image: postgres:16
    volumes: [postgres_data:/var/lib/postgresql/data, ./init/postgres:/docker-entrypoint-initdb.d]
    environment: {POSTGRES_DB: tracelayer, POSTGRES_USER: tracelayer, POSTGRES_PASSWORD: "${POSTGRES_PASSWORD}"}
    healthcheck: {test: ["CMD-SHELL","pg_isready -U tracelayer"], interval: 10s, retries: 5}

  neo4j:
    image: neo4j:5.20
    volumes: [neo4j_data:/data]
    environment: {NEO4J_AUTH: "neo4j/${NEO4J_PASSWORD}", NEO4J_PLUGINS: '[]'}

  go-api:
    build: ./cmd/api
    depends_on: {postgres: {condition: service_healthy}, neo4j: {condition: service_healthy}}
    environment: {DATABASE_URL: "${DATABASE_URL}", NEO4J_URI: "${NEO4J_URI}", INTELLIGENCE_SERVICE_URL: "http://intelligence:8000"}

  intelligence:
    build: ./intelligence
    volumes: [./intelligence/data:/app/data]

  frontend:
    build: ./frontend
    depends_on: [go-api]

volumes: {postgres_data: {}, neo4j_data: {}}
```

### 30.3 Security Requirements

| ID | Requirement |
|---|---|
| SEC-001 | All ingestion endpoints validate every field before writing to any store |
| SEC-002 | All string inputs bounded in length at API layer |
| SEC-003 | All SQL: parameterized statements only; no string interpolation |
| SEC-004 | All Cypher: parameterized queries only; no string interpolation |
| SEC-005 | Evidence provenance records immutable after creation |
| SEC-006 | Non-root container users; no privileged containers |
| SEC-007 | Secrets in environment variables only; never committed |
| SEC-008 | PostgreSQL application user: SELECT/INSERT/UPDATE on required tables only; no superuser |

### 30.4 Offline Demonstration

Demonstrable via: docker compose up on network-disconnected machine.
Verifiable via: ss -tulpn (no external connections), docker compose logs (no outbound calls).
Do not waste primary demo time on shell commands unless judge specifically asks.
Keep ss/iptables output as backup evidence.

---

## 31. Failure Modes

| Failure | Detection | System Behavior | User-Visible | Logging |
|---|---|---|---|---|
| Missing required field in ingestion | Schema validation | Record rejected; batch continues | Rejection count in batch status | batch_id, row index, field, reason |
| Duplicate observation | Dedup check | Silently deduplicated | Accepted count reflects dedup | Debug log |
| Invalid timestamp | Schema validation | Record rejected | Rejection log entry | Specific parse error |
| Unknown TXID (no blockchain match) | Post-ingest correlation | NetworkObservation stored; needs_blockchain_correlation=true | Not visible; resolved when TX later ingested | Info log |
| Negative amount | Schema validation | Record rejected | Rejection log entry | Specific error |
| Empty batch | Ingest handler | Empty success response | 200 OK; 0 accepted, 0 rejected | Info log |
| Malformed IP | Validation | Record rejected | Rejection log entry | Specific error |
| Private IP range | GeoIP enrichment | geoip_status=private_range; observation retained | Visible in evidence card | Debug log |
| Missing network observations for entity | Correlation | network_context_score=0.0; network_evidence_present=false | Stated in evidence card | Debug log |
| Mixing detected | Mixing detector | linkage_reliability reduced; mixing_flag=true | Limitation in evidence card | Info log |
| Service pattern detected | Pattern detector | service_pattern_flag=true; linkage_reliability reduced | Limitation in evidence card | Info log |
| Neo4j unavailable | Health check | Graph endpoints return 503; ingest continues to PostgreSQL | Degraded status in health API | Error log |
| Intelligence service unavailable | Health check | Detection endpoints return 503; last scores from PostgreSQL with stale flag | Degraded status | Error log |
| PostgreSQL unavailable | Connection attempt | All services unhealthy; no writes | Unhealthy in health API | Critical log |
| Corrupt GeoIP data | Startup check | GeoIP enrichment fails; geoip_status=error; ingest continues | Degraded status | Error log at startup |
| Calibration not run | Status check | investigative_priority_score labeled UNCALIBRATED in UI | Visible in evidence card | — |
| Experiment not run | Status check | PENDING shown in all experiment views | Visible in experiment view | — |

---

## 32. Logging / Observability

All logs: structured JSON with fields: timestamp, service, level, message, correlation_id, batch_id (where applicable).

| Log Type | Fields |
|---|---|
| Ingestion rejection | batch_id, row_index, field_name, rejection_reason, data_provenance |
| Entity creation | entity_id, cluster_method, address_count, heuristic_association_strength |
| Mixing detection | txid, mixing_type, confidence_level, affected_entities |
| Experiment run | experiment_id, arm, random_seed, split_type, status |
| Health check | service_name, status, schema_version |

GET /health response:
```json
{
  "status": "healthy|degraded|unhealthy",
  "schema_version": "1.x",
  "services": {
    "postgres": "healthy|unhealthy",
    "neo4j": "healthy|unhealthy",
    "intelligence": "healthy|degraded|unhealthy",
    "geoip": "healthy|error"
  }
}
```

---

## 33. Open-Source Foundations

| Component | Classification | Notes |
|---|---|---|
| Bitcoin Core (regtest) | Foundation | Real Bitcoin transaction semantics |
| Common-input / change-address heuristics | Established algorithm | Ron & Shamir 2013 [VERIFY] |
| bitcoingraph / GraphSense | Reference implementation | Not a direct code dependency; reference for heuristic implementation |
| scikit-learn (IF, LOF, LogisticRegression) | Foundation | Standard ML toolkit |
| Neo4j | Foundation | Graph database |
| PostgreSQL | Foundation | Relational storage |
| GeoIP2Fast | Foundation | Offline IP enrichment |
| Cytoscape.js | Foundation | Graph visualization |
| React | Foundation | Frontend |
| Docker / Docker Compose | Foundation | Deployment |
| Elliptic++ | External dataset | Chain-side validation only |
| node2vec | Conditional foundation | Only if Experiment 8 justifies |

Do not claim inherited open-source functionality as TraceLayer's innovation.

---

## 34. TraceLayer-Built Components

| Component | Owner | Description |
|---|---|---|
| Network observation schema and ingestion | Aniruddha + Dipak | Canonical schema; CSV/JSON/XML parsing; validation; dedup; GeoIP integration |
| Per-TX network evidence quality computation | Sayali | evidence_quality formula (Section 10.2) |
| EntityNetworkContext aggregation | Sayali | Coverage-weighted aggregation policy (Section 10.3) |
| TXID correlation engine | Dipak | Joins network observations to blockchain transactions to entities |
| Entity resolution integration | Aakanksha | Common-input heuristic integration; Neo4j entity graph construction |
| Mixing detection | Sayali | Structural pattern matching; linkage_reliability reduction |
| Custodial pattern detection | Aakanksha | Service-pattern flag; linkage reduction |
| Graph feature extraction pipeline | Sayali | All features in Section 14 |
| Logistic evidence-fusion model | Sayali | Section 16 formula; weight learning |
| Calibration evaluation | Sayali | Reliability diagram; ECE; Brier score |
| Evidence subgraph extraction | Aakanksha | Cypher queries; provenance-tagged edges |
| Evidence card layer | Dipak + Pushkar | Decomposed formula display; uncertainty flags |
| Rank shift computation | Dipak | chain_only_rank vs fused_rank |
| Chain-only vs fused comparison view | Pushkar + Dipak | Real API toggle; actual ranking comparison |
| Synthetic data generator | Sayali | Section 24; seeded blockchain + network scenarios |
| Experiment harness (Experiments 1–8) | Sayali | ExperimentRun framework; reproducibility |
| Go orchestration API | Dipak | All 17 API endpoints |
| Offline deployment packaging | Dipak + Aniruddha | Docker Compose; health checks; air-gapped validation |
| Data provenance enforcement | All | data_provenance field in all evidence records |
| Documentation / research verification | Prachi | PRD; citations; SIH submission package |

---

## 35. Team Ownership

### Dipak — Tech Lead
**Owns:** Go API (all 17 endpoints), ingestion pipeline, TXID correlation engine,
fusion model orchestration (calls intelligence service), rank shift computation,
API contract (OpenAPI YAML — published Phase 0 before all other work begins),
Docker Compose, offline validation, system integration.
**Boundary:** Canonical schemas belong to the API contract; the intelligence/ML team
does not independently redefine backend schemas.

### Pushkar — Frontend
**Owns:** All 4 core surfaces (Section 23.2), Cytoscape.js integration, chain-only
vs fused toggle (must call real API), evidence card display.
**Constraint:** Zero fabricated values. All data from API.

### Sayali — Intelligence / ML
**Owns:** Network evidence quality computation, EntityNetworkContext aggregation,
graph feature extraction, IF + LOF training and selection, evidence fusion model,
calibration evaluation, mixing detection, synthetic data generator, all 8 experiments,
Python intelligence service (FastAPI thin wrapper).
**Constraint:** Schemas defined by Dipak's API contract. Data formats must match.
Sayali's generator output must be ingestable by Go ingestion service without modification.

### Prachi — Documentation / Research
**Owns:** Research source verification, PRD maintenance, SIH submission package.
**HIGHEST PRIORITY ACTION:** Retrieve full PS26146 PDF to resolve AI/ML Focus Areas
table. If that table specifies required ML technique families, report to team immediately
— ML architecture is not frozen until this is verified.

### Aniruddha — Backend Support
**Owns:** Bitcoin Core regtest integration, transaction parsing (inputs/outputs/UTXOs),
network observation ingestion pipeline, PostgreSQL queries, GeoIP2Fast integration,
rejection logging, batch status endpoints.

### Aakanksha — Graph / Entity Support
**Owns:** Common-input-ownership clustering integration, change-address heuristic,
service-pattern flag, Neo4j entity graph construction (all nodes and edges),
evidence subgraph Cypher queries.

---

## 36. Implementation Phases

### Phase 0 — Contracts (Week 1) — EXIT: All contracts published
- Git repo structure initialized
- API contract: OpenAPI YAML for all 17 endpoints (Dipak)
- PostgreSQL schema migrations (Aniruddha)
- Neo4j schema scripts (Aakanksha)
- Python intelligence service interface defined (Sayali)
- .env.example, docker-compose.yml, all services skeleton with health endpoints returning 501/503
- Synthetic data schema defined and documented (Sayali → Dipak approval)

### Phase 1 — Vertical Slice (Week 1-2) — EXIT: One complete path end-to-end
- Single network obs → TXID → Transaction → entity (static/manual) → evidence card in UI
- Hardcoded entity assignment acceptable for this phase only

### Phase 2 — Blockchain Layer (Week 2-3) — EXIT: Real TX parsed and stored
- Bitcoin Core regtest integration
- Full TransactionInput / TransactionOutput / UTXO parsing
- GET /transactions/{txid} returns real data

### Phase 3 — Network Layer (Week 2-3, parallel) — EXIT: Batch ingested correctly
- Full CSV/JSON/XML ingestion
- Schema validation and rejection logging
- GeoIP2Fast enrichment
- TransactionNetworkEvidence computation
- data_provenance field enforced

### Phase 4 — Entity Resolution (Week 3) — EXIT: Entity graph in Neo4j
- Common-input-ownership clustering
- Change-address heuristic
- Service-pattern flag
- Entity nodes + relationships in Neo4j
- TXID correlation complete
- EntityNetworkContext computed

### Phase 5 — Intelligence (Week 3-4) — EXIT: Scores computed
- Graph feature extraction pipeline
- IF and LOF training and scoring
- chain_anomaly_score normalized to [0,1]
- Top-3 feature attribution
- Mixing detection heuristics
- Intelligence service endpoints functional

### Phase 6 — Fusion + Experiments (Week 4-5) — EXIT: Experiment 1 has real results
- Logistic evidence-fusion model trained on seeded ground truth
- FusionEvidence records computed
- chain_only_rank and fused_rank computed
- Experiment 1 all three arms run; real F1 values in ExperimentRun
- Experiment 2 (permutation control) run
- Experiment 3 (calibration) run

### Phase 7 — Frontend (Week 4-5, parallel) — EXIT: Full demo path functional
- All 4 core surfaces with real API data
- Chain-only vs fused comparison calls real API
- Evidence card shows real decomposed formula
- PENDING shown for unrun experiments

### Phase 8 — Adversarial + External Validation (Week 5-6) — EXIT: Exp 4,5,6 done
- CoinJoin stress test (Experiment 4)
- Custodial consolidation stress test (Experiment 5)
- Elliptic++ chain-side validation (Experiment 6)
- All adversarial failure modes from Section 31 tested

### Phase 9 — Offline Hardening (Week 6) — EXIT: Air-gapped demo works
- Network-disconnected docker compose up verified
- Experiment reproducibility verified (same seed → same results)
- Health checks for all services

### Phase 10 — National Submission (Week 6-7) — EXIT: Definition of Done met
- All experiment results final (real numbers)
- Prachi: PS26146 AI/ML focus areas verified; citations verified
- Demo script rehearsed; 3-5 min without notes
- Judge attack matrix rehearsed by all members
- SIH submission package complete

---

## 37. Scope: MUST / SHOULD / OPTIONAL

### MUST (minimum national demo without these)

| ID | Requirement |
|---|---|
| FR-001 | Network observation ingestion (CSV/JSON/XML); data_provenance required |
| FR-002 | Transaction ingestion with inputs/outputs/UTXOs |
| FR-003 | Schema validation; rejection logging; batch continues on partial failure |
| FR-004 | Offline GeoIP enrichment |
| FR-005 | TXID correlation (network obs → transaction → entity) |
| FR-006 | Common-input-ownership entity clustering |
| FR-007 | Service-pattern flag and linkage reduction |
| FR-008 | TransactionNetworkEvidence computation; evidence_quality formula |
| FR-009 | EntityNetworkContext computation; coverage-weighted aggregation |
| FR-010 | Graph feature extraction (Section 14 features) |
| FR-011 | IF or LOF anomaly scoring; chain_anomaly_score normalized to [0,1] |
| FR-012 | Top-3 feature attribution per entity |
| FR-013 | Mixing structural detection; linkage_reliability reduction |
| FR-014 | Logistic evidence-fusion model; investigative_priority_score |
| FR-015 | chain_only_rank and fused_rank computed and stored |
| FR-016 | Evidence subgraph extraction (Neo4j; 2-hop per entity) |
| FR-017 | All 17 API endpoints functional |
| FR-018 | 4 core frontend surfaces with real API data |
| FR-019 | Chain-only vs fused comparison view calling real API |
| FR-020 | Evidence card with fully decomposed formula and actual values |
| FR-021 | Air-gapped docker compose up; GET /health returns healthy |
| FR-022 | Experiment 1 (all 3 arms) with real F1 results |
| FR-023 | data_provenance field on all evidence records |
| NFR-001 | System healthy within 3 minutes of docker compose up |
| NFR-002 | No production service makes outbound internet calls |
| NFR-003 | Experiments deterministic given same seed and data |
| NFR-004 | All scores decomposable into named auditable components |

### SHOULD (strong national differentiation)

| ID | Requirement |
|---|---|
| FR-024 | Experiment 2 (permutation/shuffle control) |
| FR-025 | Experiment 3 (calibration: reliability diagram, ECE, Brier score) |
| FR-026 | Experiment 4 (CoinJoin stress test) |
| FR-027 | Experiment 6 (Elliptic++ chain-side validation) |
| FR-028 | Change-address heuristic in entity clustering |
| FR-029 | entity_disjoint or temporal split enforced in all experiments |
| FR-030 | Batch rejection log accessible via API |

### OPTIONAL (if time after MUST and SHOULD are solid)

| ID | Requirement |
|---|---|
| FR-031 | Experiment 5 (custodial consolidation stress test) |
| FR-032 | Experiment 7 (network-only baseline) |
| FR-033 | Experiment 8 (node2vec ablation) |
| FR-034 | node2vec embeddings — only if Experiment 8 justifies |
| FR-035 | Neo4j vs PostgreSQL 3-hop query benchmark (ADR-004 quantification) |
| FR-036 | Throughput benchmarking at synthetic scale |

---

## 38. Acceptance Criteria

| FR | Acceptance Criterion |
|---|---|
| FR-001 | Given a valid CSV with 100 NetworkObservation records and data_provenance set: all records stored; all queryable by TXID |
| FR-003 | Given a batch with 10 malformed rows (missing required field) in 100 rows: 90 stored; 10 rejection log entries with field_name and reason |
| FR-004 | After GeoIP enrichment with no network connection: all records have geoip_status; no outbound network call in service logs |
| FR-008 | Given a TXID with 4 observers, timing_spread=8s, peer_consistency=1.0: evidence_quality = clip(4/3,0,1) × (1-8/120) × 1.0 = computed value |
| FR-009 | Given entity E with 10 TXs of which 3 have observations (evidence_quality: 0.8, 0.5, 0.0): weighted_evidence_quality = (0.8+0.5+0.0+0+0+0+0+0+0+0)/10 = 0.13 |
| FR-011 | All entities have chain_anomaly_score in [0,1]; no entity returns NULL |
| FR-012 | GET /entities/{id} returns top3_features array with name and actual value |
| FR-013 | Synthetic Whirlpool-candidate TX: mixing_flag=true, mixing_type=whirlpool_candidate; entity linkage_reliability < heuristic_association_strength |
| FR-014 | investigative_priority_score ∈ [0,1] for all entities; formula components stored in FusionEvidence |
| FR-015 | After detection run: every entity in DetectionResult has chain_only_rank, fused_rank, rank_shift |
| FR-016 | GET /evidence/subgraph/{id}?hops=2 returns full 2-hop subgraph in < 2 seconds on prototype dataset |
| FR-019 | Chain-only vs fused toggle on Surface 1 calls GET /evidence/compare/{id}; re-ranks in real time |
| FR-020 | Evidence card shows a(E), n(E), r(E), w_chain, w_network, w_linkage, b, raw_logit, investigative_priority_score as actual computed values |
| FR-021 | docker compose up on network-disconnected machine; GET /health returns {"status":"healthy"} within 3 minutes |
| FR-022 | ExperimentRun records for all 3 arms of Experiment 1 have status=complete and real precision/recall/f1 values |
| FR-023 | Every NetworkObservation, Transaction, GroundTruthScenario has data_provenance set; ingestion rejects records missing it |
| NFR-001 | Measured: docker compose up → GET /health returns healthy; measured time ≤ 3 minutes |

---

## 39. Demo Plan

### 39.1 Design Principles

- Every displayed value from real backend; no hardcoded storytelling
- PENDING clearly shown for any experiment not yet run
- All numbers use [computed] placeholders in normative planning; actual values from implementation
- Primary demo: 3–5 minutes; offline proof shown only if asked

### 39.2 Recommended Narrative Structure

```
00:00–00:30  PROBLEM FRAMING
  "Two evidence streams. Investigators see them separately.
   The question: does combining them change which leads get priority?"

00:30–01:15  INGEST + PIPELINE
  Trigger ingestion of pre-loaded synthetic batch.
  Show: [computed] records accepted, [computed] rejected (logged, not lost).
  Show health endpoint: all services healthy.

01:15–02:00  ENTITY + EVIDENCE CARD
  Navigate to a flagged entity.
  Show evidence card with real values:
    Chain anomaly index: [computed] — driven by [top features with values]
    Network context score: [computed] — [observed_tx_count] of [total_tx_count] TXs observed
    Linkage reliability: [computed]
    Investigative priority: sigmoid([logit]) = [computed]

02:00–02:45  CHAIN-ONLY vs FUSED COMPARISON
  Show Surface 4: chain-only rank [computed] → fused rank [computed].
  "The chain-side anomaly score alone ranked this entity [computed]th.
   Adding network context — [computed] TXs with observations, coverage [computed] —
   moved it to rank [computed]. Rank shift: [computed]."
  Show Experiment 1 results: three arms, real F1 values.

02:45–03:30  EVIDENCE GRAPH
  Cytoscape.js subgraph: NetworkObservation → Transaction → UTXO → Address → Entity.
  "Every edge is provenance-tagged. Click any edge to see which component produced it."

03:30–04:15  ADVERSARIAL CASE — MIXING
  Switch to pre-loaded CoinJoin scenario.
  Show: mixing_type = whirlpool_candidate; linkage_reliability reduced to [computed].
  Evidence card: limitation notice visible.
  "We detected the structural pattern. We reduced linkage confidence.
   We do not claim to defeat CoinJoin or prove criminal intent."

04:15–05:00  FAILURE BOUNDARY + WHY HUMAN REVIEW
  "The system produces ranked leads with explicit uncertainty.
   The analyst decides. The system never makes a determination."
  Optionally show calibration reliability diagram if Experiment 3 is complete.
```

---

## 40. National-Level Judge Attack Matrix

Answers must be given verbatim-memorized by all team members. No marketing language.
If the correct answer is "we don't know" or "we cannot determine," say that.

**1. Why is this not just a database join?**
The TXID join is the starting point, not the contribution. What we build above it:
per-TX observation quality computation (coverage × timing tightness × peer consistency),
entity-level network context aggregation with an explicit coverage-weighted policy,
a logistic evidence-fusion model that learns whether network context is a useful predictor,
and an experimental measurement of whether adding network context changes investigative
ranking. The join is one line. The evidence model and experimental framework are
weeks of design.

**2. What does network evidence actually add?**
We don't assume it adds anything — we test it. Experiment 1 compares three arms:
naive baseline, chain-only, and fused. If w_network is near zero or Arm C F1 ≈ Arm B F1,
H0 holds and we report that honestly. If Arm C outperforms Arm B, we show by how much
and what kinds of scenarios drove the improvement. Both outcomes are valid.

**3. How do you know your synthetic network data is realistic?**
We don't claim it is. The synthetic generator creates controlled scenarios for
experimental evaluation. Every record is data_provenance = SYNTHETIC. We don't use
synthetic data to claim real-world accuracy. The permutation control (Experiment 2)
tests whether any observed improvement is genuine or an artifact of how we labeled
the generator scenarios.

**4. Can you identify the true origin IP?**
No. Bitcoin relay randomization was introduced specifically to defeat this. A first-seen
observation is a weak probabilistic prior at best. We never claim origin identification.
We observe propagation events from a vantage point. The UI never labels any IP as
"origin" or "sender."

**5. What happens with NAT?**
A peer behind NAT may present a non-routable IP. geoip_status = private_range;
the observation is retained and counted; geographic enrichment is unavailable.
Stated explicitly in the evidence card.

**6. What happens with VPN / Tor?**
VPN: IP traces to VPN provider, not user. geoip_status = resolved but enrichment
reflects VPN, not origin. Evidence card states geographic context may be unreliable.
Tor: exit node IP observed. Same treatment. We do not claim to pierce VPN or Tor.

**7. What happens with CoinJoin?**
We detect structural mixing patterns using published structural signatures. When
detected: mixing_flag = true; heuristic_association_strength is reduced;
linkage_reliability in the fusion model is reduced. Evidence card shows the limitation.
We do not claim to defeat CoinJoin, deanonymize it, or prove criminal intent.

**8. What happens with exchanges / custodians?**
High-input-count transactions trigger service_pattern_flag. This reduces
heuristic_association_strength and adds a limitation notice. The analyst is warned
that this may represent a custodial service and the cluster may contain unrelated users.
We do not auto-merge custodial consolidation patterns into one entity.

**9. What prevents false merges?**
Two mechanisms: (1) mixing_flag detected → linkage_reliability reduced → analyst warned.
(2) service_pattern_flag detected → linkage_reliability reduced → analyst warned.
We do not claim zero false merges. We quantify the false merge rate in Experiment 5
and the analyst always sees heuristic_association_strength.

**10. Why use ML at all?**
To detect graph-structural anomaly patterns across multiple features simultaneously.
Single-feature thresholds (z-score on tx_count alone) miss complex patterns. IF and LOF
are appropriate for unsupervised anomaly detection without labeled training data in
production; we use seeded synthetic ground truth to evaluate. Experiment 1 Arm A
(naive baseline) quantifies how much ML adds over simple thresholds.

**11. Why not just use graph heuristics?**
Heuristics are the entity-resolution layer (common-input, change-address). The anomaly
detection layer uses ML because graph-structural anomaly patterns involve multiple
interdependent features simultaneously. The ablation (Experiment 1 Arm A) quantifies
whether the complexity is justified.

**12. Why Isolation Forest?**
Effective on tabular data, computationally efficient, interpretable via decision paths
(enabling top-feature attribution), and standard in scikit-learn. LOF is evaluated
as an alternative. The one with better F1 on held-out seeded data is used.

**13. Why node2vec?**
We don't include node2vec by default. It is conditional on Experiment 8 showing
> 2 F1-point improvement over hand-crafted graph features with acceptable latency.
Default MVP uses hand-crafted features. We do not include it because it sounds advanced.

**14. How do you prove fusion helps?**
Experiment 1. Three arms. Same dataset. Same ground truth. We measure F1, AUPRC,
and mean rank shift for truly anomalous entities across all three arms. If the
fused arm outperforms chain-only, we show by exactly how much. If not, we report H0.

**15. How is confidence calibrated?**
The investigative_priority_score is a logistic regression output. We evaluate its
calibration on a held-out test split by plotting a reliability diagram and computing
ECE and Brier score. We compare against a naive baseline. We never call a score
"calibrated" until this evaluation is done and shows acceptable calibration.

**16. What is ground truth?**
Seeded synthetic scenarios labeled is_anomalous = true/false at scenario creation
time. We control ground truth because we generate the data. This is explicitly
synthetic. It does not represent real Bitcoin investigation outcomes.

**17. Where does ground truth come from?**
From our synthetic scenario generator. We define which transaction patterns are
labeled anomalous (e.g., multi-hop obfuscation sequences) and which are normal.
All ground truth is in GroundTruthScenario records with data_provenance = SYNTHETIC.

**18. How do you prevent leakage?**
Entity-disjoint or temporal splits enforced for all experiments. No entity appears
in both training and evaluation sets. Fixed random seeds recorded in every ExperimentRun.
Calibration evaluation uses a third split separate from training. Generator does not
use entity labels when creating network scenarios.

**19. Why Elliptic++?**
It provides labeled real Bitcoin transaction/entity data for chain-side anomaly
detection validation. It is a published, peer-reviewed dataset appropriate for
blockchain forensics research.

**20. What does Elliptic++ NOT prove?**
It contains no network telemetry. It cannot validate network evidence fusion.
Our Elliptic++ result validates chain-side detection only. Any network fusion
validation uses our synthetic seeded scenarios only.

**21. What is genuinely built by your team?**
Network observation schema and ingestion; per-TX evidence quality formula;
coverage-weighted entity network context aggregation; TXID correlation engine;
entity resolution integration; mixing detection and linkage reduction;
custodial pattern detection; graph feature extraction pipeline;
logistic evidence-fusion model; calibration evaluation methodology;
evidence subgraph extraction with provenance-tagged edges; evidence card
explainability layer; rank shift computation; chain-only vs fused comparison UI;
synthetic data generator; experiment harness (8 experiments); Go API;
offline deployment packaging; data provenance enforcement.

**22. What comes from open source?**
Bitcoin Core (regtest); common-input + change-address heuristic algorithms
(Ron & Shamir 2013 via bitcoingraph reference); scikit-learn (IF, LOF, logistic
regression); Neo4j; PostgreSQL; GeoIP2Fast; Cytoscape.js; React; Docker.

**23. Can this run offline?**
Yes, demonstrated live. docker compose up on network-disconnected machine.
All models, GeoIP data, and databases are local. No runtime external API calls.

**24. What happens when the model fails?**
If the intelligence service is unavailable: detection returns 503; last-computed
scores served from PostgreSQL with stale_flag = true; health reports degraded.
If calibration is not run: investigative_priority_score labeled UNCALIBRATED in UI.

**25. What happens when evidence conflicts?**
Conflicting observer reports → peer_consistency ≤ 0.5 → lower evidence_quality →
lower network_context_score. The conflict is visible in TransactionNetworkEvidence
and stated in the evidence card. We do not hide conflicting evidence.

**26. Can an analyst inspect why something was flagged?**
Yes. The evidence card shows: top-3 anomaly features with actual values;
network context score with per-TX breakdown; linkage reliability with its basis;
the fusion formula with all actual component values; all limitation notices;
the evidence graph with provenance-tagged edges.

**27. Can the system be fooled?**
Yes, in several ways: Tor routing makes peer_ip uninformative; CoinJoin breaks
the common-input heuristic; a large exchange's consolidation triggers false service
flags; a privacy-conscious entity with low address reuse reduces chain-side signal.
These are all known failure modes, documented, and handled with explicit uncertainty.

**28. What are the limitations?**
(1) Network observations cannot establish transaction origin. (2) Entity clusters
are heuristic inferences, not identity proofs. (3) CoinJoin breaks entity clustering.
(4) Calibration is against synthetic ground truth only. (5) System is not benchmarked
at production scale. (6) Elliptic++ validates chain-side only. (7) Synthetic generator
realism is unknown without real-world comparative data. (8) node2vec is not included
without experimental justification.

**29. What would you build next for production?**
(1) Real-world network telemetry from deployed observer nodes. (2) Production-scale
horizontal scaling. (3) Real-world calibration against confirmed investigative outcomes.
(4) Multi-vantage network evidence fusion to improve propagation origin probability
estimates. (5) Integration with investigative case management systems.

**30. Why is this better for this PS than a conventional blockchain analytics pipeline?**
This PS explicitly asks for network-layer monitoring and analysis combined with
blockchain analysis. A conventional chain-only pipeline does not address the network
layer. TraceLayer is purpose-built for the specific fusion question in PS26146,
designed for offline air-gapped deployment, and includes a testable evaluation
framework for the hypothesis the PS implies.

---

## 41. Limitations (Explicit)

These must be stated in all derivative materials — never hidden.

| Limitation | Impact | Handling |
|---|---|---|
| Network observations cannot establish transaction origin | No origin IP claim possible | Stated in UI; "origin" terminology prohibited |
| Bitcoin relay randomization defeats timing-based origin inference | Evidence quality is probabilistic, not deterministic | Stated in evidence card |
| Entity clusters are heuristic inferences | Not identity proofs | heuristic_association_strength field; terminology policy |
| CoinJoin breaks common-input heuristic | False merges under mixing | Detected; linkage_reliability reduced; stated |
| Custodial consolidation mimics suspicious patterns | False merges and false positives | Service-pattern flag; linkage reduced; stated |
| Calibration is against synthetic ground truth only | Real-world accuracy unknown | Scope statement required in all calibration references |
| Synthetic data realism is unverified | Experiment results may not transfer to real data | Explicitly scoped as synthetic |
| System not benchmarked at production scale | Prototype targets only | Stated in Section 26 |
| node2vec not included without experimental justification | No embedding-based features in MVP | Condition documented |
| PS AI/ML focus areas table not yet verified | ML architecture may need updating | Verification gate in Section 44 |
| Literature thresholds for mixing detection not yet verified | Hard-coded thresholds may be wrong | [VERIFY] labels; thresholds configurable |

---

## 42. Future Production Evolution

State these when asked "what would you build for production?" Do not claim these are in scope.

1. Real-world network observer node deployment and multi-vantage telemetry collection
2. Production-scale horizontal scaling (stateless API; replicated intelligence workers)
3. Real-world calibration against confirmed investigative outcomes
4. Improved origin probability estimation via multi-vantage timing with formal diffusion models
5. WabiSabi / Wasabi 2.x / JoinMarket CoinJoin detection
6. RBAC and secure analyst session management
7. Integration with investigative case management systems
8. Continuous learning from analyst feedback (with appropriate safeguards)
9. Multi-chain extension (if PS scope expands)

---

## 43. Research References

All citations marked [VERIFY] must be confirmed by Prachi before use in any slide,
write-up, or verbal answer. Do not fabricate citations. Do not cite a paper for a
claim it does not actually support.

| Reference | Use | Status |
|---|---|---|
| Ron & Shamir, "Quantitative Analysis of the Full Bitcoin Transaction Graph," Financial Cryptography 2013 | Common-input-ownership heuristic | [VERIFY: exact venue, DOI] |
| Meiklejohn et al., "A Fistful of Bitcoins," ACM IMC 2013 | Address clustering / blockchain analysis | [VERIFY: DOI 10.1145/2504730.2504747] |
| Pham & Lee, "Anomaly Detection in Bitcoin Network Using Unsupervised Learning Methods," arXiv 2016 | Unsupervised anomaly detection on Bitcoin | [VERIFY: arXiv:1611.03941] |
| Gavenda, Svenda, Bobon, Sedlacek, "Analysis of Input-Output Mappings in CoinJoin Transactions with Arbitrary Values," 2025 | CoinJoin structural signatures; mixing | [VERIFY: arXiv:2510.17284] |
| Schnoering & Vazirgiannis, "Heuristics for Detecting CoinJoin Transactions on the Bitcoin Blockchain," 2023 | CoinJoin heuristics | [VERIFY: arXiv:2311.12491] |
| Bitcoin Core documentation | Relay delay randomization; P2P protocol | [VERIFY: which version introduced diffusion relay] |
| Grover & Leskovec, "node2vec: Scalable Feature Learning for Networks," KDD 2016 | Graph embeddings | [VERIFY: only if Experiment 8 justifies use] |
| Elliptic++ dataset documentation | Dataset structure; evaluation protocol | [VERIFY: label format; temporal split protocol] |
| bitcoingraph repository | Common-input heuristic reference implementation | [VERIFY: run against test data before citing] |

---

## 44. Open Questions / Verification Gates

These must be resolved before the associated work is frozen. Do not implement around them.

| Gate | Owner | Blocking | Status |
|---|---|---|---|
| SIH26146 AI/ML Focus Areas table retrieved and reviewed | Prachi | ML architecture freeze | **[PENDING — HIGHEST PRIORITY]** |
| All mixing detection thresholds verified against primary sources | Prachi | MIX-005; hard-coding mixing heuristics | **[PENDING]** |
| Elliptic++ label format and temporal split protocol verified | Sayali | Experiment 6 | **[PENDING]** |
| bitcoingraph repository run against test data; Ron & Shamir implementation verified | Aakanksha | Entity resolution implementation | **[PENDING]** |
| Bitcoin Core relay randomization version verified | Dipak | Technical accuracy of network limitation claim | **[PENDING]** |
| GeoIP2Fast zero-runtime-call property verified via source code and offline test | Aniruddha | NET-004 claim | **[PENDING]** |
| Neo4j vs PostgreSQL 3-hop query benchmark run | Aniruddha | ADR-004 quantification | **[PENDING — before national submission]** |
| All experiment results computed | Sayali | Final demo and slides | **[PENDING]** |
| Calibration evaluated on held-out test split | Sayali | Any "calibrated" claim | **[PENDING]** |

---

## 45. Final Definition of Done

The system is NOT done because the dashboard renders or the ML model produces a score.

National-level Definition of Done requires ALL of the following:

**Pipeline:**
☐ Valid NetworkObservation can enter the system, be enriched, and be stored
☐ Valid Transaction can enter the system, be parsed to inputs/outputs/UTXOs, and be stored
☐ TXID correlation links network obs to blockchain transaction to entity
☐ Entity resolved via common-input heuristic with heuristic_association_strength computed
☐ data_provenance field populated on all evidence records

**Intelligence:**
☐ chain_anomaly_score computed for all entities via IF or LOF; in [0,1]
☐ Top-3 contributing features with actual values available per entity
☐ network_context_score computed for all entities; 0.0 for unobserved (not NULL)
☐ investigative_priority_score computed via logistic fusion model
☐ chain_only_rank and fused_rank computed for all entities

**Evidence Quality:**
☐ Mixing structural detection flags appropriate TXs; reduces linkage_reliability
☐ Service-pattern detection flags high-input-count TXs; reduces linkage_reliability
☐ Evidence card shows decomposed formula with actual values
☐ All limitation notices appear correctly for mixing, service pattern, low coverage, absent network obs
☐ No UI text uses "origin IP", "sender IP", or "identified individual"

**Experiments:**
☐ Experiment 1 (all 3 arms): complete; real precision/recall/F1 in ExperimentRun
☐ Experiment 2 (permutation control): complete; result compared to Experiment 1 Arm C
☐ Experiment 3 (calibration): complete; reliability diagram and ECE computed
☐ Experiment 4 (mixing stress test): complete; detection F1 and linkage degradation measured

**Demo:**
☐ Evidence graph rendered from real backend data in Cytoscape.js
☐ Chain-only vs fused comparison toggle calls real API; re-ranks
☐ All displayed values from real backend; zero hardcoded demo values
☐ PENDING shown for any incomplete experiment
☐ Air-gapped docker compose up works; GET /health returns healthy
☐ Full 3-5 minute demo rehearsed without notes

**Submission:**
☐ PS26146 AI/ML focus areas table reviewed; any required architecture changes made
☐ All citations verified by Prachi (exact DOIs / arXiv IDs)
☐ SIH submission package complete before deadline
☐ All judge attack questions rehearsed by all team members without notes

---

## Appendix A — Requirement ID Index

| Range | Section |
|---|---|
| FR-001 – FR-036, NFR-001 – NFR-004 | Section 37 — Scope |
| NET-001 – NET-007 | Section 10 — Network Evidence Model |
| BC-001 – BC-004 | Section 11 — Blockchain / UTXO Model |
| ENT-001 – ENT-005 | Section 12 — Entity Resolution |
| CORR-001 – CORR-004 | Section 13 — Evidence Correlation |
| ML-001 – ML-005 | Section 15 — Anomaly Detection |
| FUS-001 – FUS-005 | Section 16 — Evidence Fusion Model |
| MIX-001 – MIX-005 | Section 19 — CoinJoin / Mixing Handling |
| CUST-001 – CUST-004 | Section 20 — Custodial / Service Patterns |
| GRAPH-001 – GRAPH-004 | Section 21 — Evidence Graph |
| API-001 – API-017 | Section 22 — API Contracts |
| SEC-001 – SEC-008 | Section 30 — Security / Offline Deployment |

---

## PRD v2 Change Log

### Corrected

| Item | v1 Error | v2 Correction |
|---|---|---|
| Fusion model terminology | Called "Platt scaling" — incorrect; Platt scales a single score | Renamed "Logistic evidence-fusion model"; calibration step separated and correctly described |
| Network evidence = suspiciousness | Formula added network evidence as positive term toward suspiciousness | Redesigned: network evidence is evidence quality (reliability measure), not anomaly signal; experiment tests whether it has investigative discriminative power |
| Entity aggregation | Simple mean of observed TXs (dilution problem) | Coverage-weighted mean over all TXs (observed contribute value; unobserved contribute 0) |
| Entity "cluster_confidence" | Misleading calibrated-probability implication | Renamed heuristic_association_strength throughout |
| Brier > 0.25 rule | Arbitrary universal threshold | Removed; replaced with ECE, reliability diagram, baseline comparison |
| Invented example numbers | rank #17→#3, confidence 0.71, 4 observers, 8.3s hardcoded in normative text | All replaced with [computed] placeholders |
| "10-50% degradation expected" | Pre-defined target for mixing experiment | Removed; replaced with "measure empirically, compare to literature after" |
| "Platt scaling" for weight fitting | Technically wrong | Weights are logistic regression coefficients; calibration is a separate evaluation step |
| bitcoingraph dependency | "implemented via bitcoingraph" | Repositioned as reference implementation; TraceLayer owns its canonical model |
| IB / R&AW downstream mention | Unsupported organizational claim | Replaced with "authorized investigative consumers" |
| "foreign commercial tools" attack | Unsupported categorical claims | Replaced with technical positioning around offline capability and PS-specific design |
| src/dst IP terminology | Incorrectly implied source/destination semantics | Corrected to observing_node and peer_ip throughout |
| 9 frontend screens | Excessive for hackathon timeline | Reduced to 4 core surfaces |
| Demo over-packed | 10+ items in 5 minutes | Trimmed to 7 narrative beats with clear timing |
| Whirlpool "5 or 8" hard-coded | Not yet verified | Added [VERIFY] labels; mixine_type uses _candidate suffix |

### Removed

- Arbitrary Brier score threshold
- Invented rank/score/timing example numbers from normative requirements
- Pre-defined mixing degradation target
- "IB" and "R&AW" as named downstream consumers
- Categorical claims about commercial blockchain analytics platforms
- 5 of 9 frontend screens (reduced to 4 core surfaces)
- "ss -tulpn / iptables -L" from primary demo script

### Redesigned

- Network evidence model: evidence quality formula separated from anomaly signal
- Entity network context: coverage-weighted aggregation replacing naive mean
- Fusion model: three-term logistic regression; weights learned; not assumed positive for network evidence
- Calibration: evaluation framework (reliability diagram, ECE, Brier vs baseline) replacing single arbitrary threshold
- Experiment 1: strengthened with explicit controls
- Experiment 2: new permutation/shuffle control added
- Mixing detection: _candidate suffix; [VERIFY] on thresholds; two separate measurements defined
- Data provenance: first-class field (REAL_NODE_CAPTURE / REAL_PUBLIC / SYNTHETIC / DERIVED)
- Frontend: 4 core surfaces replacing 9 independent pages
- Demo: 7-beat narrative structure; no wasted time on shell commands

### Remains Verification-Gated

- SIH26146 AI/ML focus areas table (ML architecture not frozen)
- Mixing detection thresholds from literature (Whirlpool / Wasabi structural signatures)
- All experimental results (Experiments 1–8)
- Elliptic++ compatibility verification
- bitcoingraph implementation verification
- Bitcoin Core relay randomization version
- GeoIP2Fast offline property verification
- Neo4j vs PostgreSQL benchmark

### Implement First (Priority Order)

1. API contract (OpenAPI YAML) — must exist before Phase 1
2. Schema migrations (PostgreSQL + Neo4j) — must exist before Phase 1
3. Vertical slice (Phase 1) — proves the pipeline end-to-end before adding complexity
4. Network observation ingestion (Phase 3) — core evidence stream
5. TXID correlation + entity resolution (Phase 4) — core structural contribution
6. Evidence quality computation + EntityNetworkContext (Phase 4/5) — the fusion input
7. Chain-side anomaly scoring (Phase 5) — required for Experiment 1
8. Logistic fusion model + Experiment 1 (Phase 6) — the core hypothesis test
9. Evidence card UI (Phase 7) — judge-facing validation of explainability
10. Mixing detection (Phase 5) — required for adversarial case

---

*Version 2.0 — Team Mempool — SIH26146 — NTRO — 2026-09-04*
*This document supersedes PRD v1 in all respects.*
*No claim labeled [PENDING], [HYPOTHESIS], or [VERIFY] may be presented as established
until resolved. No invented numbers. No fabricated citations. No unsupported claims.*
