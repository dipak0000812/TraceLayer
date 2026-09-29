#!/usr/bin/env bash
# scripts/demo-reset.sh
#
# DESTRUCTIVE — removes the tracelayer-round2 postgres volume so the next
# demo.sh run starts from an empty database.
#
# This script is the ONLY permitted way to reset the demo database.
# Do not run this during a live recording.
#
# Usage:
#   POSTGRES_PASSWORD=<password> bash scripts/demo-reset.sh
set -Eeuo pipefail

PROJECT_NAME="tracelayer-round2"
POSTGRES_VOLUME="${PROJECT_NAME}_postgres-data"

# -- Safeguards -------------------------------------------------------------
[[ -n "${POSTGRES_PASSWORD:-}" ]] || {
  printf 'ERROR: POSTGRES_PASSWORD must be set in the environment.\n' >&2
  exit 1
}

printf '\n'
printf '================================================================\n'
printf '          TraceLayer Demo Database Reset\n'
printf '================================================================\n'
printf '  This will permanently DELETE the Docker volume:\n'
printf '    %s\n' "${POSTGRES_VOLUME}"
printf '\n'
printf '  All ingested transactions, network observations, entities,\n'
printf '  forensic leads, and ingestion history will be lost.\n'
printf '\n'
printf '  Only run this before a fresh demo recording.\n'
printf '================================================================\n'
printf '\n'
printf 'Type RESET to confirm: '
read -r confirm
if [[ "$confirm" != "RESET" ]]; then
  printf 'Aborted -- database was not touched.\n'
  exit 0
fi

# -- Stop services and remove volumes ---------------------------------------
printf '\nStopping TraceLayer compose stack and removing volumes...\n'
docker compose -p "$PROJECT_NAME" down -v --timeout 10 2>/dev/null || true

# -- Verify volume exists before trying to remove ---------------------------
if docker volume inspect "$POSTGRES_VOLUME" >/dev/null 2>&1; then
  printf 'Removing volume %s...\n' "$POSTGRES_VOLUME"
  docker volume rm -f "$POSTGRES_VOLUME" >/dev/null 2>&1 || true
  printf 'Volume removed.\n'
else
  printf 'Volume %s cleaned up successfully.\n' "$POSTGRES_VOLUME"
fi

printf '\n'
printf 'Reset complete. Run scripts/demo.sh to start a clean demonstration.\n'
