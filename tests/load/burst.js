import http from 'k6/http';
import { check } from 'k6';

export const options = {
  scenarios: {
    burst: {
      executor: 'ramping-arrival-rate',
      startRate: 100,
      timeUnit: '1s',
      preAllocatedVUs: 200,
      maxVUs: 2000,
      stages: [
        { target: 2000, duration: '30s' },
        { target: 200, duration: '1m' },
        { target: 0, duration: '30s' },
      ],
    },
  },
};

const BASE_URL = __ENV.API_BASE_URL || 'http://localhost:3000';
const API_KEY = __ENV.API_KEY || 'dev-api-key';

export default function () {
  const payload = JSON.stringify({
    schema_version: '1.0',
    idempotency_key: `burst-${__VU}-${__ITER}-${Date.now()}`,
    channel: 'email',
    priority: 'normal',
    recipient: { address: 'load@example.com' },
    template: { id: 'welcome', data: {} },
  });

  const res = http.post(`${BASE_URL}/v1/notifications`, payload, {
    headers: { 'Content-Type': 'application/json', 'X-API-Key': API_KEY },
  });
  check(res, { 'status ok': (r) => r.status >= 200 && r.status < 500 });
}
