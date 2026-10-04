import http from 'k6/http';
import { check } from 'k6';

const target = __ENV.TARGET_URL || 'http://localhost:8080';
const rate = Number(__ENV.RATE || 1200);
const duration = __ENV.DURATION || '30s';

export const options = {
  scenarios: {
    work: {
      executor: 'constant-arrival-rate',
      rate: rate,
      timeUnit: '1s',
      duration: duration,
      preAllocatedVUs: 300,
      maxVUs: 2000,
    },
  },
  thresholds: {
    checks: ['rate>0.99'],
  },
};

export default function () {
  const res = http.get(`${target}/api/work`);
  check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
  });
}
