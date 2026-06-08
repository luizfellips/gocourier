#!/usr/bin/env bash
set -euo pipefail

PROM="${PROMETHEUS_URL:-http://localhost:9090}"
GRAFANA="${GRAFANA_URL:-http://localhost:3001}"

echo "Checking Prometheus targets..."
curl -sf "$PROM/api/v1/targets" | grep -q '"health":"up"'

echo "Checking alert rules..."
curl -sf "$PROM/api/v1/rules" | grep -q 'HighAPIErrorRate'

echo "Checking Grafana health..."
curl -sf "$GRAFANA/api/health" | grep -q 'ok'

echo "Telemetry stack validation passed."
