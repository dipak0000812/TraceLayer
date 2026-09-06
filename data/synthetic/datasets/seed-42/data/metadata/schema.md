# TraceLayer Round 2 — Synthetic Dataset Schema (v1.0.0)

**provenance: SYNTHETIC** — no real Bitcoin network captures or seized data
were used anywhere in this generation process. Provenance is asserted
per-record (not only in the manifest) — see below.

## Directory layout (this is the contract the Go backend codes against)

```
data/
├── raw/                          <- the ONLY inputs the application ingests
│   ├── transactions.csv
│   ├── transactions.json
│   ├── network_observations.csv
│   └── network_observations.json
├── eval/                         <- evaluator-only, must never be read by
│   │                                the detection/enrichment pipeline
│   ├── ground_truth.json
│   └── geoip_fixture.json
└── metadata/
    ├── manifest.json
    ├── schema.md
    └── validation_report.json
```

## data/raw/transactions.csv|.json

| field | type | notes |
|---|---|---|
| txid | string (64 hex chars) | deterministic synthetic identifier, seed-derived |
| timestamp | ISO-8601 UTC | |
| input_addresses | JSON array of strings (encoded as a JSON string inside the CSV cell) | synthetic addresses, prefixed `sbc1` |
| output_addresses | JSON array of strings (same encoding) | |
| input_amounts | JSON array of floats | **unit: BTC**, 8 decimal places |
| output_amounts | JSON array of floats | **unit: BTC**, 8 decimal places |
| fee | float | BTC; always `sum(input_amounts) - sum(output_amounts)`, never negative |
| script_type | enum | P2PKH / P2WPKH / P2SH / P2TR |
| provenance | string | always `SYNTHETIC`, asserted on every row |
| dataset_id | string | matches metadata/manifest.json `dataset_id` for this run |
| generator_version | string | generator release that produced this row |

Array-valued CSV cells use plain `json.dumps` encoding, e.g.:
`input_addresses` cell contents: `["sbc1a1b2...", "sbc1c3d4..."]`
Parse with `json.loads(cell_value)` in any downstream consumer.

## data/raw/network_observations.csv|.json

This is the RAW record — as if it arrived off the wire, before enrichment.
It deliberately does **not** contain `geo_country` or `asn`. Those are
derived fields the offline GeoIP/ASN enrichment component is responsible
for producing:

```
Raw NetworkObservation (data/raw/)
        |
        v
offline GeoIP/ASN enrichment component
        |
        v
Enriched NetworkObservation (geo_country, asn attached — application-owned,
                              not generator output)
```

| field | type | notes |
|---|---|---|
| observation_id | string | |
| timestamp | ISO-8601 UTC | independently jittered vs. the linked tx timestamp |
| src_ip / dst_ip | IPv4 dotted-quad | drawn ONLY from RFC 5737 (TEST-NET-1/2/3) and RFC 1918 private ranges |
| src_port / dst_port | int | 1024-65535 |
| txid | string | foreign key into transactions.txid |
| provenance | string | always `SYNTHETIC`, asserted on every row |
| dataset_id | string | matches metadata/manifest.json `dataset_id` for this run |
| generator_version | string | generator release that produced this row |

## data/eval/geoip_fixture.json (EVALUATION ONLY)

The generator internally assigns a deterministic "true" `geo_country`/`asn`
to every synthetic IP it ever emits, but that value is written **only**
here, keyed by IP — never onto the raw observation. Use this fixture to
grade the enrichment component's output; do not let the enrichment
component read this file at runtime, or it can trivially "pass" by copying
the answer instead of doing GeoIP/ASN lookup.

Format: `{"<ip>": {"geo_country": "...", "asn": "AS....."}, ...}`

## data/eval/ground_truth.json (EVALUATION ONLY — never fed to the ML pipeline)

One record per scenario group:
`scenario_id, scenario_type, txids, ground_truth_entity_ids, anomaly_label,
expected_relationships, network_condition, seed`

`scenario_type` and `anomaly_label` are the ONLY place anomaly ground truth
appears in the entire dataset. They do not appear in `data/raw/*`.

## data/metadata/manifest.json

`dataset_id, generator_version, random_seed, generated_at, record_counts,
schema_version, provenance, scenario_counts, checksums`

## Anti-leakage boundary

The Go backend's normal detection code path must have **no import, no
file-read, no config reference** to anything under `data/eval/`. Only a
separate evaluation harness (offline, CI-only) may open those files.
