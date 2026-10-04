import http from 'k6/http';
import { check } from 'k6';

const target = __ENV.TARGET_URL || 'http://localhost:8080';
const rate = Number(__ENV.RATE || 120);
const duration = __ENV.DURATION || '30s';

export const options = {
  scenarios: {
    slow: {
      executor: 'constant-arrival-rate',
      rate: rate,
      timeUnit: '1s',
      duration: duration,
      preAllocatedVUs: 250,
      maxVUs: 1000,
    },
  },
};

export default function () {
  const res = http.get(`${target}/api/slow`);
  check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
  });
}
