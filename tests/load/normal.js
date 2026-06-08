import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  scenarios: {
    steady: {
      executor: 'constant-arrival-rate',
      rate: 200,
      timeUnit: '1s',
      duration: '2m',
      preAllocatedVUs: 50,
      maxVUs: 200,
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
  },
};

const BASE_URL = __ENV.API_BASE_URL || 'http://localhost:3000';
const API_KEY = __ENV.API_KEY || 'dev-api-key';

export default function () {
  const payload = JSON.stringify({
    schema_version: '1.0',
    idempotency_key: `normal-${__VU}-${__ITER}-${Date.now()}`,
    channel: 'email',
    priority: 'normal',
    recipient: { address: 'load@example.com' },
    template: { id: 'welcome', data: {} },
  });

  const res = http.post(`${BASE_URL}/v1/notifications`, payload, {
    headers: { 'Content-Type': 'application/json', 'X-API-Key': API_KEY },
  });
  check(res, { 'accepted': (r) => r.status === 202 || r.status === 200 });
  sleep(0.01);
}
