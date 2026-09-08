# TraceLayer Data Contract

**Status:** Current implementation contract

This document separates runtime input, normalized persistence, derived data, and evaluation-only artifacts. The current Round-2 runtime is PostgreSQL-backed and does not require graph storage, GeoIP, ASN enrichment, or graph identifiers.

## 1. Runtime dataset

The mandatory synthetic runtime dataset is:

```text
data/synthetic/datasets/seed-42/data/raw/
```

The manifest identifies:

```text
dataset_id: 6bc084b677a63411
generator_version: 1.0.0
provenance: SYNTHETIC
transactions: 109 records
network_observations: 368 records
random_seed: 42
```

The runtime must use the raw transaction and network files, not evaluation fixtures.

## 2. Evaluation-only data

These files are not runtime intelligence or detection inputs:

```text
data/synthetic/datasets/seed-42/data/eval/ground_truth.json
data/synthetic/datasets/seed-42/data/eval/geoip_fixture.json
```

They are reserved for evaluation and fixture use. The second file does not create a current runtime enrichment requirement.

## 3. Raw transaction input

Raw transaction records contain:

| Field | Meaning |
|---|---|
| `txid` | 64-character lowercase hexadecimal transaction identifier |
| `timestamp` | Transaction timestamp |
| `input_addresses` | JSON array of input addresses |
| `output_addresses` | JSON array of output addresses |
| `input_amounts` | JSON array of BTC decimal amounts |
| `output_amounts` | JSON array of BTC decimal amounts |
| `fee` | BTC fee |
| `script_type` | Current supported script type enum |
| `provenance` | Seed-42 records use `SYNTHETIC` |
| `dataset_id` | Seed-42 manifest identifier |
| `generator_version` | Seed-42 generator version |

The CSV and JSON forms represent the same logical records.

## 4. Raw network input

Raw network observations contain:

| Field | Meaning |
|---|---|
| `observation_id` | Stable observation identifier |
| `timestamp` | Observation timestamp |
| `src_ip` | Source peer IP |
| `dst_ip` | Destination IP |
| `src_port` | Source port |
| `dst_port` | Destination port |
| `txid` | Observed transaction ID, normalized as `observed_txid` |
| `provenance` | Seed-42 records use `SYNTHETIC` |
| `dataset_id` | Seed-42 manifest identifier |
| `generator_version` | Seed-42 generator version |

Network ingestion is decoupled from transaction ingestion. There is deliberately no transaction foreign key that prevents orphan observations before correlation.

## 5. Normalized and persisted transactions

The `transactions` table persists transaction identity, timestamp, input/output addresses, input/output amounts, fee, script type, provenance, dataset ID, generator version, and ingestion timestamp.

The database uses `VARCHAR(64)` for TXIDs and exact PostgreSQL numeric storage for BTC amounts. The Go domain parses and validates amounts using satoshi precision rather than `float64`.

Validation includes:

- 64 lowercase hexadecimal TXID.
- Matching input and output amount/address structure.
- Non-negative fee.
- Fee equal to input total minus output total.
- Supported script type.

## 6. Normalized and persisted network observations

The `network_observations` table persists observation ID, observed TXID, timestamp, source and destination IPs, ports, nullable network metadata fields, provenance, dataset ID, generator version, and correlation status. The nullable metadata fields are `geo_country`, `asn`, and `propagation_delay_ms`.

The lifecycle distinguishes observations that correlate to a transaction from observations that remain orphaned.

Those nullable metadata fields are retained by the current schema and evidence response, but no enrichment service is required by the current Round-2 runtime and the current raw seed-42 input does not require them.

## 7. Entities

Entity resolution uses deterministic common-input ownership clustering in Go. An entity is persisted with:

- Deterministic `entity_id`.
- Member addresses.
- Cluster size.
- Provenance.
- Dataset ID.
- Timestamps defined by the schema.

Entity IDs are derived from sorted member addresses. No graph ID or external graph projection is part of the current contract.

## 8. Forensic leads

Forensic leads are derived from correlation, entity resolution, evidence features, intelligence scores where available, fusion, and ranking. Persisted lead data includes:

- `lead_id`
- `entity_id`
- `primary_txid`
- `chain_only_score`
- `network_score`
- `fused_score`
- `chain_only_rank`
- `fused_rank`
- `rank_shift`
- Explanation and association fields emitted by ranking.

Scores are investigative ranking values, not calibrated probabilities.

## 9. Intelligence input

The Python worker receives request data from Go. Its current Pydantic schema accepts:

- `txid`
- `amount_btc`
- `fee_btc`
- `input_count`
- `output_count`
- Address arrays.
- Amount arrays.
- `script_type`
- Correlated network observations.
- Optional fields in the worker schema for enriched network metadata.

The optional worker fields do not make enrichment a current runtime requirement. The current Go pipeline does not provide a required enrichment service.

## 10. Amount precision boundary

Go uses exact satoshi-based values. The Go-to-Python client currently serializes `amount_btc` and `fee_btc` as `float64`, and the Python worker receives JSON floating-point values. Exact decimal preservation inside Python is therefore not guaranteed.

## 11. Provenance

The seed-42 runtime uses:

```text
provenance = SYNTHETIC
dataset_id = 6bc084b677a63411
generator_version = 1.0.0
```

Derived entity and lead records retain provenance and dataset context as defined by the current storage layer.

## 12. Scope exclusions

The following are excluded from the current data contract:

- Graph nodes and relationships.
- Evidence graph identifiers.
- Evidence subgraph data.
- Runtime enrichment fields.
- Runtime reads of evaluation fixtures.

They may be reconsidered only through a separate future scope decision.
