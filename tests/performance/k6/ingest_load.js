import http from 'k6/http';
import { check, sleep } from 'k6';

// SLO targets (initial baseline — revise after first run)
// p95 ingest < 150ms, error rate < 0.1%, throughput 500 req/s sustained

export const options = {
  stages: [
    { duration: '2m', target: 100 },
    { duration: '5m', target: 500 },
    { duration: '2m', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
  },
};

const BASE_URL = __ENV.API_BASE_URL || 'http://localhost:3000';
const API_KEY = __ENV.API_KEY || 'dev-api-key';

export default function () {
  const payload = JSON.stringify({
    schema_version: '1.0',
    idempotency_key: `k6-${__VU}-${__ITER}-${Date.now()}`,
    channel: 'email',
    priority: 'normal',
    recipient: { address: 'load@example.com' },
    template: { id: 'welcome' },
  });

  const res = http.post(`${BASE_URL}/v1/notifications`, payload, {
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': API_KEY,
    },
  });

  check(res, {
    'status is 202': (r) => r.status === 202,
    'has delivery_id': (r) => r.json('delivery_id') !== undefined,
  });

  sleep(0.1);
}
