# Performance test SLO targets
#
# Load test (k6 ingest_load.js):
#   p50 ingest < 50ms
#   p95 ingest < 150ms
#   p99 ingest < 300ms
#   throughput 500 req/s sustained
#   error rate < 0.1%
#
# Stress test (vegeta):
#   document max RPS before error rate exceeds 1%
#
# Spike test:
#   2000 req/s for 60s, recovery within 5m
#
# Soak test:
#   200 req/s for 4h, stable memory, outbox pending near zero
