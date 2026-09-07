# TraceLayer Intelligence Module — Complete Documentation

## Overview

The Intelligence Module is a **stateless Python FastAPI microservice** that receives Bitcoin transaction data enriched with P2P network observations from the Go API, and returns anomaly scores, pattern flags, and human-readable explanations.

**Port:** `8000`  
**Endpoint:** `POST /intelligence/score`  
**Health Check:** `GET /health`  
**Swagger UI:** `http://localhost:8000/docs`

---

## Architecture

```
Go API Server (port 3000)
    │
    │  POST /intelligence/score
    │  (sends transaction + network observations)
    │
    ▼
┌──────────────────────────────────────────────────┐
│           Intelligence Worker (port 8000)          │
│                                                    │
│  1. Feature Extraction (11 features)               │
│     └─ 7 blockchain + 4 network features           │
│                                                    │
│  2. Isolation Forest (anomaly scoring)             │
│     └─ Pre-trained on Elliptic dataset (203K tx)   │
│                                                    │
│  3. Pattern Detectors                              │
│     ├─ Peeling Chain (value-splitting laundering)  │
│     └─ CoinJoin (multi-party mixing)               │
│                                                    │
│  4. Explainability (SHAP / LIME)                   │
│     └─ "WHY is this suspicious?"                   │
│                                                    │
│  5. Logistic Fusion                                │
│     └─ Combines all scores → single priority       │
│                                                    │
│  Returns: scores, flags, explanations              │
└──────────────────────────────────────────────────┘
```

### Key Constraints
- **Stateless**: Holds no database connections. All data comes from the request payload.
- **Anti-leakage**: NEVER reads from `data/eval/` at runtime. Only the evaluation harness accesses ground truth.
- **Loopback only**: Listens on `0.0.0.0:8000`, accessed by Go API via `http://localhost:8000`.

---

## Folder Structure — What Each File Does

```
intelligence/
│
├── __init__.py                     # Package marker
├── main.py                         # FastAPI app — the entry point
│                                   #   - Starts the server on port 8000
│                                   #   - Loads the model at startup
│                                   #   - Wires all components together
│                                   #   - Handles POST /intelligence/score
│                                   #   - Handles GET /health
│
├── config.py                       # ALL tunable parameters in one place
│                                   #   - Feature names (11 features)
│                                   #   - Fusion weights (w_chain, w_net, w_mixing, bias)
│                                   #   - IF hyperparameters (n_estimators, contamination)
│                                   #   - Detector thresholds (peel ratio, CoinJoin min inputs)
│                                   #   - File paths (model, data directories)
│                                   #   - Q(tx) constants (N_THRESHOLD=3, MAX_SPREAD=120s)
│
├── requirements.txt                # Python dependencies (pip install -r requirements.txt)
├── Dockerfile                      # Docker container build file
│
├── schemas/                        # Pydantic models for API request/response
│   ├── __init__.py
│   ├── request.py                  # INPUT format — what Go sends us:
│   │                               #   ScoreRequest → list of TransactionInput
│   │                               #   TransactionInput → txid, amounts, addresses,
│   │                               #                      network_observations
│   │                               #   NetworkObservationInput → peer IP, timestamp,
│   │                               #                             country, ASN
│   │
│   └── response.py                 # OUTPUT format — what we return:
│                                   #   ScoreResponse → list of TransactionScore
│                                   #   TransactionScore → chain_score, network_score,
│                                   #                      fused_score, flags, top_features,
│                                   #                      explanation, explainability_method
│                                   #   FeatureAttribution → feature name, SHAP value, direction
│
├── features/                       # Feature extraction pipeline
│   ├── __init__.py
│   ├── blockchain_features.py      # Extracts 7 chain features from transaction data:
│   │                               #   1. tx_amount        — total BTC moved
│   │                               #   2. fee              — transaction fee
│   │                               #   3. input_count      — number of senders
│   │                               #   4. output_count     — number of receivers
│   │                               #   5. input_output_ratio — inputs / outputs
│   │                               #   6. amount_variance  — how spread out are output values
│   │                               #   7. fee_rate         — fee as % of amount
│   │
│   ├── network_features.py         # Extracts 4 network features from P2P observations:
│   │                               #   8.  network_observation_count — peers that saw this tx
│   │                               #   9.  unique_peer_count         — distinct IPs
│   │                               #   10. timing_spread_seconds     — time spread
│   │                               #   11. network_quality           — Q(tx) formula
│   │                               #
│   │                               #   Also computes Q(tx):
│   │                               #   Q(tx) = min(1, N/3) × max(0, 1 − spread/120)
│   │
│   └── pipeline.py                 # Combines blockchain + network → 11-element vector
│                                   #   CombinedFeatures.to_array() → numpy array
│                                   #   CombinedFeatures.to_dict()  → {name: value}
│                                   #   FeaturePipeline.extract(tx) → CombinedFeatures
│                                   #   FeaturePipeline.extract_batch(txs) → matrix
│
├── models/                         # ML models
│   ├── __init__.py
│   ├── isolation_forest.py         # Isolation Forest wrapper:
│   │                               #   - load_or_initialize() — load .joblib or create fresh
│   │                               #   - train(X) — fit on feature matrix
│   │                               #   - save(path) — dump model + scaler + normalization params
│   │                               #   - score(vector) → float [0, 1] (1 = most anomalous)
│   │                               #   - score_batch(X) → array of scores
│   │                               #   - Normalizes raw IF scores from [-0.5, 0.5] to [0, 1]
│   │
│   └── pretrained/                 # Saved model files
│       ├── .gitkeep
│       └── isolation_forest_v1.joblib  # Generated after training (not in git)
│
├── data/                           # Data loading utilities
│   ├── __init__.py
│   ├── elliptic_loader.py          # Loads Elliptic Kaggle dataset (203K real Bitcoin tx):
│   │                               #   - load() → reads 3 CSVs
│   │                               #   - get_labeled_data() → (X, y) with binary labels
│   │                               #   - get_temporal_split() → train/test by time step
│   │                               #   - Labels: 1=illicit→1, 2=licit→0, unknown→dropped
│   │
│   ├── synthetic_loader.py         # Loads TraceLayer seed-42 synthetic data:
│   │                               #   - load() → reads transactions + observations JSONs
│   │                               #   - get_correlated_data() → tx with matched observations
│   │
│   ├── feature_mapper.py           # Maps Elliptic's 166 features → our 11 features:
│   │                               #   - map_elliptic_to_tracelayer(df) → 11-column df
│   │                               #   - Network features set to 0.0 (Elliptic has no network)
│   │
│   └── elliptic/                   # Manually downloaded Elliptic dataset
│       ├── elliptic_txs_features.csv    # 203,769 × 167 (txid + 166 features)
│       ├── elliptic_txs_classes.csv     # 203,769 × 2 (txid, class)
│       └── elliptic_txs_edgelist.csv    # 234,355 × 2 (tx → tx edges)
│
├── training/                       # Offline training scripts (run once)
│   ├── __init__.py
│   ├── train_isolation_forest.py   # Train the model:
│   │                               #   1. Load Elliptic (or fallback to synthetic)
│   │                               #   2. Temporal split 70/30 (by time step, not random)
│   │                               #   3. Map features to our 11-feature schema
│   │                               #   4. Train Isolation Forest (unsupervised)
│   │                               #   5. Evaluate against Elliptic labels
│   │                               #   6. Save to models/pretrained/
│   │
│   └── evaluate_model.py           # Evaluation metrics:
│                                   #   - AUC-ROC, Average Precision
│                                   #   - Precision, Recall, F1 at threshold
│                                   #   - Precision@K (top K accuracy)
│                                   #   - Confusion matrix (TP, FP, TN, FN)
│
├── detectors/                      # Structural pattern detectors (rule-based)
│   ├── __init__.py
│   ├── base.py                     # Abstract interface:
│   │                               #   - BaseDetector.detect(tx) → DetectionResult
│   │                               #   - DetectionResult: score, flagged, reason, signals
│   │
│   ├── peeling_chain.py            # Peeling chain detector:
│   │                               #   Looks for: 2 outputs, high value disparity (>5:1),
│   │                               #   single input, unique output addresses
│   │                               #   Flags: PEELING_CHAIN when score ≥ 0.6
│   │                               #   Disparity is a GATE — without it, score collapses
│   │
│   └── coinjoin.py                 # CoinJoin/mixing detector:
│                                   #   Looks for: ≥3 inputs AND ≥3 outputs,
│                                   #   equal-denomination outputs, low address overlap
│                                   #   Flags: COINJOIN_MIXING when score ≥ 0.5
│
├── explainability/                 # Explainable AI (XAI) layer
│   ├── __init__.py
│   ├── shap_explainer.py           # SHAP TreeExplainer (PRIMARY — always on):
│   │                               #   - ~10ms per transaction
│   │                               #   - Mathematically exact Shapley values
│   │                               #   - explain(vector) → top 5 feature attributions
│   │                               #   - Each attribution: {feature, value, direction}
│   │                               #   - Fallback to z-score deviation if SHAP fails
│   │
│   └── lime_explainer.py           # LIME TabularExplainer (SECONDARY — on demand):
│                                   #   - ~500ms-2s per transaction
│                                   #   - Creates local linear surrogate model
│                                   #   - Use by passing explainer="lime" in request
│                                   #   - Different perspective for analyst second opinion
│
├── fusion/                         # Evidence fusion
│   ├── __init__.py
│   └── logistic_fusion.py          # Logistic sigmoid fusion formula:
│                                   #   fused = σ(w₁·chain + w₂·net·Q(tx) − w₃·mix + bias)
│                                   #
│                                   #   Default weights:
│                                   #     w_chain = 2.0  (chain evidence weight)
│                                   #     w_net   = 1.5  (network evidence weight)
│                                   #     w_mix   = 1.0  (mixing penalty weight)
│                                   #     bias    = -1.0 (shifts baseline below 0.5)
│                                   #
│                                   #   Key: network is DISCOUNTED by Q(tx)
│                                   #   If Q(tx)=0, network contribution vanishes
│                                   #
│                                   #   Also provides:
│                                   #   - fuse_chain_only() — for rank shift comparison
│                                   #   - decompose() — breakdown for evidence cards
│
├── evaluation/                     # Offline evaluation harness
│   ├── __init__.py
│   ├── metrics.py                  # Rank shift analysis:
│   │                               #   - precision_at_k, recall_at_k
│   │                               #   - rank_shift_analysis — Spearman/Kendall correlation
│   │                               #   - Top movers report
│   │
│   └── harness.py                  # End-to-end evaluation:
│                                   #   - ONLY file that reads data/eval/ground_truth.json
│                                   #   - Runs full pipeline on seed-42 data
│                                   #   - Reports: rank shift, precision@K, pattern detection
│                                   #   - Tests core hypothesis: does network evidence help?
│
└── tests/                          # Unit tests (pytest)
    ├── __init__.py
    ├── test_features.py            # 15 tests: blockchain features, network features,
    │                               #   Q(tx) formula, pipeline shape, batch extraction
    │
    ├── test_detectors.py           # 8 tests: classic peel, equal outputs (not peel),
    │                               #   many outputs, single output, classic CoinJoin,
    │                               #   normal tx, varied outputs, too few inputs
    │
    ├── test_explainability.py      # 3 tests: SHAP returns attributions, valid feature
    │                               #   names, LIME returns attributions
    │
    ├── test_fusion.py              # 8 tests: sigmoid bounds, Q=0 no network contribution,
    │                               #   network increases score, mixing decreases score,
    │                               #   custom weights, decomposition
    │
    └── test_api.py                 # 6 tests: /health endpoint, single transaction scoring,
                                    #   no network observations, batch scoring, empty batch
                                    #   rejected, LIME explainer selection
```

---

## How to Run Everything

### 1. Install Dependencies
```bash
cd TraceLayer/intelligence
pip install -r requirements.txt
```

### 2. Train the Model (run once)
```bash
cd TraceLayer
python -m intelligence.training.train_isolation_forest
```
**What happens:**
- Loads Elliptic dataset from `intelligence/data/elliptic/`
- Splits by time (70% train, 30% test) — no data leakage
- Trains Isolation Forest (unsupervised, 200 trees)
- Prints AUC-ROC, precision@K metrics
- Saves model to `intelligence/models/pretrained/isolation_forest_v1.joblib`

### 3. Start the Server
```bash
cd TraceLayer
python -m intelligence.main
```
Server starts at `http://localhost:8000`

### 4. Test via Swagger UI (Browser)
Open: **http://localhost:8000/docs**
1. Click `POST /intelligence/score`
2. Click "Try it out"
3. Paste this sample input:

```json
{
  "transactions": [
    {
      "txid": "test_tx_001",
      "amount_btc": 1.25,
      "fee_btc": 0.0001,
      "input_count": 3,
      "output_count": 2,
      "input_addresses": ["addr1", "addr2", "addr3"],
      "output_addresses": ["addr4", "addr5"],
      "input_amounts": [0.5, 0.5, 0.2501],
      "output_amounts": [1.0, 0.25],
      "script_type": "P2PKH",
      "network_observations": [
        {
          "observation_id": "obs1",
          "observed_at": "2025-03-22T16:34:24Z",
          "src_ip": "172.16.2.66",
          "geo_country": "BR",
          "asn": "AS65096"
        },
        {
          "observation_id": "obs2",
          "observed_at": "2025-03-22T16:34:30Z",
          "src_ip": "10.0.1.55",
          "geo_country": "DE",
          "asn": "AS65097"
        },
        {
          "observation_id": "obs3",
          "observed_at": "2025-03-22T16:34:36Z",
          "src_ip": "192.168.1.10",
          "geo_country": "IN",
          "asn": "AS65098"
        }
      ]
    }
  ],
  "explainer": "shap"
}
```
4. Click **Execute** — see the full output

### 5. Test a Peeling Chain Pattern
```json
{
  "transactions": [
    {
      "txid": "peeling_test",
      "amount_btc": 10.0,
      "fee_btc": 0.001,
      "input_count": 1,
      "output_count": 2,
      "input_addresses": ["sender1"],
      "output_addresses": ["big_change", "small_peel"],
      "input_amounts": [10.001],
      "output_amounts": [9.5, 0.5],
      "script_type": "P2PKH",
      "network_observations": []
    }
  ]
}
```
**Expected:** `peeling_chain_flag: true` (9.5:0.5 = 19:1 ratio)

### 6. Test a CoinJoin Pattern
```json
{
  "transactions": [
    {
      "txid": "coinjoin_test",
      "amount_btc": 0.5,
      "fee_btc": 0.005,
      "input_count": 5,
      "output_count": 5,
      "input_addresses": ["in1", "in2", "in3", "in4", "in5"],
      "output_addresses": ["out1", "out2", "out3", "out4", "out5"],
      "input_amounts": [0.1, 0.1, 0.1, 0.1, 0.105],
      "output_amounts": [0.1, 0.1, 0.1, 0.1, 0.1],
      "script_type": "P2SH",
      "network_observations": []
    }
  ]
}
```
**Expected:** `mixing_flag: true` (5 equal-denomination outputs)

### 7. Run Unit Tests
```bash
cd TraceLayer
python -m pytest intelligence/tests/ -v
```
**Expected:** 40 passed, 0 failed

### 8. Run Evaluation Harness
```bash
cd TraceLayer
python -m intelligence.evaluation.harness
```
**What it tests:**
- Rank shift (does network evidence change rankings?) → H1 SUPPORTED
- Precision@K (chain-only vs fused)
- Pattern detection accuracy (TP/FP/FN for peeling and CoinJoin)

---

## Input / Output Format

### Input (what Go sends)

| Field | Type | Description |
|---|---|---|
| `txid` | string | 64-hex-char transaction ID |
| `amount_btc` | float | Total output value in BTC |
| `fee_btc` | float | Transaction fee in BTC |
| `input_count` | int | Number of inputs (senders) |
| `output_count` | int | Number of outputs (receivers) |
| `input_addresses` | string[] | Sender addresses |
| `output_addresses` | string[] | Receiver addresses |
| `input_amounts` | float[] | Input amounts in BTC |
| `output_amounts` | float[] | Output amounts in BTC |
| `script_type` | string | P2PKH, P2WPKH, P2SH, P2TR |
| `network_observations` | object[] | P2P network sightings |
| `explainer` | string | "shap" (default) or "lime" |

### Output (what we return)

| Field | Type | Range | Description |
|---|---|---|---|
| `chain_score` | float | 0-1 | How anomalous on the blockchain |
| `network_score` | float | 0-1 | How anomalous on the P2P network |
| `mixing_penalty` | float | 0-1 | CoinJoin/peeling penalty |
| `network_quality_q` | float | 0-1 | How reliable is network evidence |
| `fused_score` | float | 0-1 | **Final priority score** (higher = investigate first) |
| `heuristic_association_strength` | float | 0-1 | Combined evidence strength |
| `peeling_chain_flag` | bool | — | Peeling chain pattern detected? |
| `mixing_flag` | bool | — | CoinJoin mixing detected? |
| `flags` | string[] | — | Additional flags (HIGH_FAN_OUT, RAPID_DISPERSION, etc.) |
| `top_features` | object[] | — | Top 5 SHAP/LIME feature attributions |
| `explanation` | string | — | Human-readable explanation for the analyst |
| `explainability_method` | string | — | "shap" or "lime" |

---

## The 11 Features

| # | Feature | Source | What It Measures |
|---|---|---|---|
| 1 | `tx_amount` | Chain | Total BTC moved in this transaction |
| 2 | `fee` | Chain | Transaction fee paid to miners |
| 3 | `input_count` | Chain | Number of sender addresses |
| 4 | `output_count` | Chain | Number of receiver addresses |
| 5 | `input_output_ratio` | Chain | Inputs / Outputs (fan-in vs fan-out) |
| 6 | `amount_variance` | Chain | How spread out are the output values |
| 7 | `fee_rate` | Chain | Fee as a fraction of total amount |
| 8 | `network_observation_count` | Network | How many peers saw this tx |
| 9 | `unique_peer_count` | Network | Distinct source IPs |
| 10 | `timing_spread_seconds` | Network | Time between first and last observation |
| 11 | `network_quality` | Network | Q(tx) = coverage × freshness |

---

## Fusion Formula

```
fused_score = σ(2.0 × chain_score + 1.5 × network_score × Q(tx) − 1.0 × mixing_penalty − 1.0)
```

Where σ(x) = 1 / (1 + e^(-x)) is the logistic sigmoid.

**Why this works:**
- `chain_score` has the **highest weight** (2.0) — blockchain evidence is most reliable
- `network_score` is **discounted by Q(tx)** — poor quality network evidence is downweighted
- `mixing_penalty` **reduces** the score — CoinJoin makes network evidence unreliable
- Negative `bias` (-1.0) ensures normal transactions default to below 0.5

---

## Evaluation Results (Current)

| Metric | Value |
|---|---|
| **Hypothesis** | H1 SUPPORTED (rank shift ≠ 0) |
| **Precision@5 (fused)** | 100% (+40% over chain-only) |
| **Precision@10 (fused)** | 90% (+10% over chain-only) |
| **Peeling Chain** | Precision=94.1%, Recall=100%, F1=97.0% |
| **CoinJoin** | Precision=100%, Recall=100%, F1=100% |
| **Unit Tests** | 40/40 passed |

---

## Important Rules

1. **Always run from the TraceLayer root**, not from inside `intelligence/`:
   ```bash
   cd TraceLayer
   python -m intelligence.main          # ✅ Correct
   python -m intelligence.training.train_isolation_forest  # ✅ Correct
   ```

2. **Never import from `data/eval/`** in any file except `evaluation/harness.py`

3. **The model must be trained before starting the server** — run `train_isolation_forest.py` first

4. **Browser URL is `http://localhost:8000`**, not `http://0.0.0.0:8000`
