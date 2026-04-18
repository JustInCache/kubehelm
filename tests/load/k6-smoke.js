import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 75,
  duration: '45s',
  thresholds: {
    http_req_duration: ['p(95)<500']
  }
};

const BASE = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const health = http.get(`${BASE}/healthz`);
  check(health, { 'healthz is 200': (r) => r.status === 200 });
  sleep(0.2);
}

