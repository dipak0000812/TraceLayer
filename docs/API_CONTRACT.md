# TraceLayer — API Contract v1.0

**Document Status:** Approved Architectural and Contract Baseline  
**Scope:** Round-2 Integration Boundary  
**Architecture:** 12 External Go REST Endpoints + 1 Internal Python Intelligence Endpoint  

---

## 1. Overview & Service Topology

```
                  ┌────────────────────────────────────────┐
                  │          Analyst Web Browser           │
                  └───────────────────┬────────────────────┘
                                      │
                                      ▼ HTTP (JSON)
                  ┌────────────────────────────────────────┐
                  │         Go API Server (Port 8080)      │
                  │        (12 External REST Endpoints)    │
                  └─────────────┬──────────────────────────┘
                                │
                                ▼ HTTP (Internal Port 8000)
                  ┌────────────────────────────────────────┐
                  │     Python Intelligence Worker         │
                  │   (POST /intelligence/score - Internal)│
                  └────────────────────────────────────────┘
```

The Go server exposes 12 external endpoints for frontend and operational consumers.  
The Python FastAPI intelligence worker exposes 1 internal endpoint (`POST /intelligence/score`), which is never exposed through the frontend.

---

## 2. External Go REST Endpoints (12 Endpoints)

### 2.1 System & Operations

#### `GET /health`
- **Description:** Returns aggregate health of the Go process, PostgreSQL, Neo4j, and Python worker.
- **Response `200 OK`:**
  ```json
  {
    "status": "HEALTHY",
    "timestamp": "2026-09-06T11:00:00Z",
    "services": {
      "postgres": "UP",
      "neo4j": "UP",
      "intelligence_worker": "UP"
    }
  }
  ```

#### `GET /api/v1/config/fusion`
- **Description:** Returns active fusion weights, quality discount thresholds, and heuristic model parameters.
- **Response `200 OK`:**
  ```json
  {
    "model_version": "1.0.0",
    "weights": {
      "w_chain": 2.0,
      "w_net": 1.5,
      "w_mixing": 1.0,
      "bias": -1.0
    },
    "quality_parameters": {
      "n_threshold": 3,
      "max_spread_seconds": 120.0
    }
  }
  ```

---

### 2.2 Decoupled Data Ingestion

#### `POST /api/v1/ingest/blockchain`
- **Description:** Ingests raw Bitcoin transaction records (`data/raw/transactions.csv` or `.json`).
- **Request Body (Multipart or JSON Array):**
  ```json
  [
    {
      "txid": "b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b",
      "timestamp": "2025-03-22T16:34:25Z",
      "input_addresses": ["sbc18d20c1d613b34c0e6946f41fc34692fc9daf10"],
      "output_addresses": ["sbc1231363128bc87877088348b38c25cc78ac7f16"],
      "input_amounts": [0.74038447],
      "output_amounts": [0.73932792],
      "fee": 0.00105655,
      "script_type": "P2PKH",
      "provenance": "SYNTHETIC",
      "dataset_id": "6bc084b677a63411",
      "generator_version": "1.0.0"
    }
  ]
  ```
- **Response `200 OK`:**
  ```json
  {
    "ingested_count": 108,
    "duplicate_count": 1,
    "rejected_count": 0,
    "dataset_id": "6bc084b677a63411"
  }
  ```

#### `POST /api/v1/ingest/network`
- **Description:** Ingests raw P2P network observations (`data/raw/network_observations.csv` or `.json`). Decoupled from transactions.
- **Request Body (JSON Array):**
  ```json
  [
    {
      "observation_id": "obs_0195e6f3",
      "timestamp": "2025-03-22T16:34:24Z",
      "src_ip": "172.16.2.66",
      "dst_ip": "198.51.100.199",
      "src_port": 34690,
      "dst_port": 7589,
      "txid": "b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b",
      "provenance": "SYNTHETIC",
      "dataset_id": "6bc084b677a63411",
      "generator_version": "1.0.0"
    }
  ]
  ```
- **Response `200 OK`:**
  ```json
  {
    "ingested_count": 368,
    "duplicate_count": 0,
    "rejected_count": 0
  }
  ```

#### `POST /api/v1/ingest/bulk`
- **Description:** Triggers bulk ingestion of the canonical directory `data/raw/` (ingesting `transactions.csv` and `network_observations.csv`).
- **Response `200 OK`:**
  ```json
  {
    "status": "COMPLETED",
    "transactions_ingested": 109,
    "transactions_unique": 108,
    "transactions_duplicates": 1,
    "network_observations_ingested": 368,
    "duration_ms": 142
  }
  ```

#### `GET /api/v1/ingest/status`
- **Description:** Returns current dataset counts, rejection logs, and correlation statistics.
- **Response `200 OK`:**
  ```json
  {
    "transactions_total": 108,
    "network_observations_total": 368,
    "correlated_observations": 352,
    "orphan_observations": 16,
    "rejected_records": []
  }
  ```

---

### 2.3 Correlation, Leads & Evidence

#### `POST /api/v1/correlate`
- **Description:** Executes TXID logical correlation, Common-Input DSU clustering, and calls Python intelligence scoring.
- **Response `200 OK`:**
  ```json
  {
    "status": "SUCCESS",
    "entities_clustered": 20,
    "leads_generated": 20,
    "observations_correlated": 352
  }
  ```

#### `GET /api/v1/leads`
- **Description:** Returns ranked forensic investigative leads for analyst triage.
- **Query Parameters:** `page` (default 1), `limit` (default 20), `sort_by` (`fused_rank` | `rank_shift`).
- **Response `200 OK`:**
  ```json
  {
    "total_leads": 20,
    "page": 1,
    "limit": 20,
    "leads": [
      {
        "lead_id": "lead_1a2b3c4d",
        "entity_id": "ent_9f8e7d6c",
        "primary_txid": "b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b",
        "chain_only_score": 0.42,
        "network_score": 0.88,
        "fused_score": 0.79,
        "chain_only_rank": 8,
        "fused_rank": 2,
        "rank_shift": 6,
        "network_evidence_quality": 0.94,
        "heuristic_association_strength": 0.81,
        "anomaly_flags": ["RAPID_DISPERSION", "HIGH_FAN_OUT"],
        "explanation": "Network propagation dispersion across 4 ASNs within 12s shifts entity priority by +6 ranks."
      }
    ]
  }
  ```

#### `GET /api/v1/leads/{id}`
- **Description:** Detailed evidence card and breakdown for a specific lead or entity.
- **Response `200 OK`:** Single lead object identical to the element schema above, with additional metadata.

#### `GET /api/v1/evidence/{txid}`
- **Description:** Retrieves the complete dual-layer evidence bundle for a transaction.
- **Response `200 OK`:**
  ```json
  {
    "transaction": {
      "txid": "b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b",
      "timestamp": "2025-03-22T16:34:25Z",
      "input_addresses": ["sbc18d20c1d613b34c0e6946f41fc34692fc9daf10"],
      "output_addresses": ["sbc1231363128bc87877088348b38c25cc78ac7f16"],
      "fee": 0.00105655,
      "script_type": "P2PKH"
    },
    "network_observations": [
      {
        "observation_id": "obs_0195e6f3",
        "observed_at": "2025-03-22T16:34:24Z",
        "first_heard_peer_ip": "172.16.2.66",
        "src_port": 34690,
        "geo_country": "BR",
        "asn": "AS65096",
        "propagation_delay_ms": -1000
      }
    ],
    "quality_metric": 0.94,
    "disclaimer": "First-heard peer IP indicates vantage relay observation, NOT cryptographic sender identity."
  }
  ```

#### `GET /api/v1/evidence/subgraph/{id}`
- **Description:** Returns the Neo4j topological neighborhood for Cytoscape.js rendering.
- **Query Parameters:** `depth` (default 2, max 2), `max_nodes` (default 100).
- **Response `200 OK`:**
  ```json
  {
    "elements": {
      "nodes": [
        {"data": {"id": "ent_9f8e7d6c", "label": "Entity", "type": "entity"}},
        {"data": {"id": "sbc18d20...", "label": "Address", "type": "address"}},
        {"data": {"id": "tx_b85038...", "label": "Transaction", "type": "transaction"}},
        {"data": {"id": "tx_b85038:0", "label": "UTXO", "type": "utxo"}}
      ],
      "edges": [
        {"data": {"source": "ent_9f8e7d6c", "target": "sbc18d20...", "label": "CONTROLS"}},
        {"data": {"source": "sbc18d20...", "target": "tx_b85038...", "label": "INPUT_OF"}}
      ]
    }
  }
  ```

#### `GET /api/v1/evidence/compare/{id}`
- **Description:** Returns the dual-hypothesis comparison for entity `{id}`.
- **Response `200 OK`:**
  ```json
  {
    "entity_id": "ent_9f8e7d6c",
    "chain_only_rank": 8,
    "fused_rank": 2,
    "rank_shift": 6,
    "chain_only_score": 0.42,
    "fused_score": 0.79,
    "network_contribution": 0.37,
    "hypothesis_evaluation": "Network propagation dispersion elevates risk priority."
  }
  ```

---

## 3. Internal Python Intelligence Endpoint (1 Endpoint)

#### `POST /intelligence/score`
- **Caller:** Go API server (`internal/intelligence/client.go`).
- **Audience:** Internal microservice (Port 8000). Not accessible from frontend.
- **Request Body:**
  ```json
  {
    "transactions": [
      {
        "txid": "b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b",
        "amount_btc": 0.74038447,
        "fee_btc": 0.00105655,
        "input_count": 1,
        "output_count": 1,
        "script_type": "P2PKH",
        "network_observations": [
          {
            "observed_at": "2025-03-22T16:34:24Z",
            "src_ip": "172.16.2.66",
            "geo_country": "BR",
            "asn": "AS65096"
          }
        ]
      }
    ]
  }
  ```
- **Response `200 OK`:**
  ```json
  {
    "scores": [
      {
        "txid": "b85038db8d34756615a8c82752b931ded57f99c1911097b5d0e21f0c648e638b",
        "chain_score": 0.42,
        "network_score": 0.88,
        "mixing_penalty": 0.0,
        "network_quality_q": 0.94,
        "fused_score": 0.79,
        "heuristic_association_strength": 0.81,
        "flags": ["RAPID_DISPERSION"],
        "explanation": "High peer dispersion with rapid broadcast across Latin America."
      }
    ]
  }
  ```
- **Error Behavior:** If the Python worker fails, it returns `500 Internal Server Error`. Go records the run as `FAILED` and surfaces the error to the analyst, maintaining scientific test integrity without silent fallback.
