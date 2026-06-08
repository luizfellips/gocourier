import http from 'k6/http';
import { check } from 'k6';

export const options = {
  vus: 10,
  iterations: 500,
};

const BASE_URL = __ENV.API_BASE_URL || 'http://localhost:3000';
const API_KEY = __ENV.API_KEY || 'dev-api-key';
const KEY = __ENV.IDEMPOTENCY_KEY || 'duplicate-storm-key';

export default function () {
  const payload = JSON.stringify({
    schema_version: '1.0',
    idempotency_key: KEY,
    channel: 'email',
    priority: 'normal',
    recipient: { address: 'load@example.com' },
    template: { id: 'welcome', data: {} },
  });

  const res = http.post(`${BASE_URL}/v1/notifications`, payload, {
    headers: { 'Content-Type': 'application/json', 'X-API-Key': API_KEY },
  });
  check(res, { 'status ok': (r) => r.status === 200 || r.status === 202 });
}
