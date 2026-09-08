# TraceLayer Intelligence Worker

This directory documents the separate Python worker implemented on `origin/feature/intelligence`.

## Contract

The worker is a stateless FastAPI service with:

```text
GET  /health
POST /intelligence/score
```

`POST /intelligence/score` accepts a non-empty `transactions` array. Each transaction includes TXID, BTC amount fields, input/output counts, address and amount arrays, script type, and correlated network observations. The optional `explainer` field selects SHAP or LIME.

The response contains per-transaction chain, network, mixing, quality, fused, flag, and explanation fields.

The worker does not connect to PostgreSQL. Go sends the request over HTTP and owns persistence and orchestration. The worker loads the checked-in Isolation Forest artifact under `intelligence/models/pretrained/`.

## Precision boundary

The Go domain uses exact satoshi-based values. The current Go client serializes `amount_btc` and `fee_btc` as `float64`, and the Python Pydantic schema receives floating-point JSON values. Exact decimal preservation inside Python is not guaranteed by the current implementation.

## Runtime data

The scoring endpoint uses request data. Evaluation fixtures are not part of its runtime contract. The seed-42 raw dataset is owned by the Go ingestion pipeline.

## Integration status

This worker is not yet integrated into a verified containerized runtime. The current Dockerfile needs its build context and package path corrected before Phase 14 container work.
