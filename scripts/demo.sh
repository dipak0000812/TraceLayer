#!/usr/bin/env bash
set -Eeuo pipefail

PROJECT_NAME="tracelayer-round2"
API_URL="${API_URL:-http://localhost:8080}"
INTELLIGENCE_URL="${INTELLIGENCE_URL:-http://localhost:8000}"
FRONTEND_URL="${FRONTEND_URL:-http://localhost:3000}"
DATASET_DIR="data/synthetic/datasets/seed-42/data/raw"
TRANSACTIONS_CSV="${DATASET_DIR}/transactions.csv"
NETWORK_CSV="${DATASET_DIR}/network_observations.csv"
WAIT_SECONDS="${WAIT_SECONDS:-120}"

fail() {
  printf 'TraceLayer demo failed: %s\n' "$1" >&2
  exit 1
}

warn() {
  printf 'WARN: %s\n' "$1" >&2
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "required command not found: $1"
}

port_available() {
  python3 - "$1" <<'PY'
import socket
import sys

with socket.socket() as sock:
    sock.settimeout(0.2)
    sys.exit(0 if sock.connect_ex(("127.0.0.1", int(sys.argv[1]))) != 0 else 1)
PY
}

json_value() {
  python3 - "$1" "$2" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as stream:
    value = json.load(stream)
for part in sys.argv[2].split("."):
  value = value[int(part)] if part.isdigit() else value[part]
print(value)
PY
}

verify_lead_response() {
  python3 - "$1" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as stream:
  body = json.load(stream)
if not body.get("lead_id") or not body.get("primary_txid"):
  raise SystemExit("lead response did not contain lead_id and primary_txid")
PY
}

verify_evidence_response() {
  python3 - "$1" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as stream:
  body = json.load(stream)
if not body.get("transaction"):
  raise SystemExit("evidence response did not contain a transaction")
PY
}

wait_for_services() {
  printf 'Starting Compose services...\n'
  POSTGRES_HOST_PORT="$POSTGRES_HOST_PORT" docker compose -p "$PROJECT_NAME" up -d >/dev/null

  printf 'Waiting for PostgreSQL, intelligence, API, and frontend...\n'
  for ((attempt = 1; attempt <= WAIT_SECONDS; attempt++)); do
    postgres_ok=0
    api_ok=0
    intelligence_ok=0
    frontend_ok=0

    postgres_container="${PROJECT_NAME}-postgres-1"
    postgres_health=$(docker inspect --format '{{if .State.Health}}{{.State.Health.Status}}{{else}}unknown{{end}}' "$postgres_container" 2>/dev/null || true)
    [[ "$postgres_health" == "healthy" ]] && postgres_ok=1

    if api_health=$(curl -fsS "$API_URL/health" 2>/dev/null); then
      python3 - "$api_health" <<'PY' >/dev/null 2>&1 && api_ok=1 || true
import json
import sys
health = json.loads(sys.argv[1])
assert health.get("status") == "HEALTHY"
assert health.get("services", {}).get("postgres") == "UP"
PY
    fi

    if intelligence_health=$(curl -fsS "$INTELLIGENCE_URL/health" 2>/dev/null); then
      python3 - "$intelligence_health" <<'PY' >/dev/null 2>&1 && intelligence_ok=1 || true
import json
import sys
health = json.loads(sys.argv[1])
assert health.get("status") == "ok"
assert health.get("model_loaded") is True
PY
    fi

    curl -fsSI "$FRONTEND_URL/" >/dev/null 2>&1 && frontend_ok=1 || true

    if ((postgres_ok && api_ok && intelligence_ok && frontend_ok)); then
      printf 'All services are ready.\n'
      return
    fi
    sleep 1
  done

  docker compose -p "$PROJECT_NAME" ps >&2 || true
  fail "services did not become ready within ${WAIT_SECONDS}s"
}

require_command docker
require_command curl
require_command python3

docker compose version >/dev/null 2>&1 || fail "Docker Compose is unavailable"
[[ -n "${POSTGRES_PASSWORD:-}" ]] || fail "POSTGRES_PASSWORD must be set in the environment"
[[ -f "$TRANSACTIONS_CSV" ]] || fail "missing dataset file: $TRANSACTIONS_CSV"
[[ -f "$NETWORK_CSV" ]] || fail "missing dataset file: $NETWORK_CSV"

# PostgreSQL is internal to the stack; choose a host port only for optional local access.
if [[ -z "${POSTGRES_HOST_PORT:-}" && -n "$(docker ps -q -f "name=^${PROJECT_NAME}-postgres-1$")" ]]; then
  POSTGRES_HOST_PORT=55432
elif [[ -z "${POSTGRES_HOST_PORT:-}" ]]; then
  POSTGRES_HOST_PORT=55432
  while ! port_available "$POSTGRES_HOST_PORT"; do
    POSTGRES_HOST_PORT=$((POSTGRES_HOST_PORT + 1))
  done
fi
export POSTGRES_HOST_PORT

transaction_result=$(mktemp)
network_result=$(mktemp)
correlate_result=$(mktemp)
leads_result=$(mktemp)
lead_result=$(mktemp)
evidence_result=$(mktemp)
trap 'rm -f "$transaction_result" "$network_result" "$correlate_result" "$leads_result" "$lead_result" "$evidence_result"' EXIT

wait_for_services

# -- Detect database state --------------------------------------------------
db_state_result=$(mktemp)
trap 'rm -f "$transaction_result" "$network_result" "$correlate_result" "$leads_result" "$lead_result" "$evidence_result" "$db_state_result"' EXIT
curl -fsS "$API_URL/api/v1/leads?page=1&limit=1" >"$db_state_result" 2>/dev/null || true
existing_leads=$(json_value "$db_state_result" total_leads 2>/dev/null || echo "0")
if (( existing_leads > 0 )); then
  printf '\n'
  printf 'NOTICE: Database already contains %s lead(s).\n' "$existing_leads"
  printf 'This is a REPEAT RUN. Ingestion will report duplicates (not errors).\n'
  printf 'Correlation will report 0 newly-correlated observations (already done).\n'
  printf 'To start fresh: POSTGRES_PASSWORD=<pw> bash scripts/demo-reset.sh\n'
  printf '\n'
fi

printf 'Ingesting seed-42 blockchain CSV...\n'
curl -fsS -X POST "$API_URL/api/v1/ingest/blockchain" \
  -H 'Content-Type: text/csv' \
  --data-binary "@${TRANSACTIONS_CSV}" >"$transaction_result" \
  || fail "blockchain ingestion request failed"

printf 'Ingesting seed-42 network CSV...\n'
curl -fsS -X POST "$API_URL/api/v1/ingest/network" \
  -H 'Content-Type: text/csv' \
  --data-binary "@${NETWORK_CSV}" >"$network_result" \
  || fail "network ingestion request failed"

printf 'Running correlation, entity resolution, intelligence, and ranking...\n'
curl -fsS -X POST "$API_URL/api/v1/correlate" >"$correlate_result" \
  || fail "correlation request failed"

curl -fsS "$API_URL/api/v1/leads?page=1&limit=1" >"$leads_result" \
  || fail "lead verification request failed"

transactions=$(( $(json_value "$transaction_result" ingested_count) + $(json_value "$transaction_result" duplicate_count) ))
network_observations=$(( $(json_value "$network_result" ingested_count) + $(json_value "$network_result" duplicate_count) ))
correlated=$(json_value "$correlate_result" observations_correlated)
entities=$(json_value "$correlate_result" entities_clustered)
leads=$(json_value "$leads_result" total_leads)

(( leads > 0 )) || fail "correlation completed but generated no leads"

lead_id=$(json_value "$leads_result" leads.0.lead_id)
primary_txid=$(json_value "$leads_result" leads.0.primary_txid)
curl -fsS "$API_URL/api/v1/leads/$(python3 -c 'import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1], safe=""))' "$lead_id")" >"$lead_result" \
  || fail "lead detail request failed"
verify_lead_response "$lead_result" || fail "lead detail response was invalid"
curl -fsS "$API_URL/api/v1/evidence/$(python3 -c 'import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1], safe=""))' "$primary_txid")" >"$evidence_result" \
  || fail "evidence request failed"
verify_evidence_response "$evidence_result" || fail "evidence response was invalid"

printf '\nTraceLayer Demo Ready\n'
printf '%s\n' '---------------------'
printf 'Dataset:                     seed-42 (SYNTHETIC)\n'
printf 'CSV rows processed:          %s (transactions) / %s (network)\n' "$transactions" "$network_observations"
printf 'Newly ingested this pass:    %s tx / %s obs\n' \
  "$(json_value "$transaction_result" ingested_count)" \
  "$(json_value "$network_result" ingested_count)"
printf 'Newly correlated this pass:  %s (zero on repeat run is correct)\n' "$correlated"
printf 'Address clusters (entities): %s (59 expected from seed-42 DSU)\n' "$entities"
printf 'Forensic leads:              %s\n' "$leads"
printf 'Intelligence:                READY (Isolation Forest loaded)\n'
printf 'Frontend:                    %s\n' "$FRONTEND_URL"

if command -v xdg-open >/dev/null 2>&1; then
  xdg-open "$FRONTEND_URL" >/dev/null 2>&1 || true
fi
