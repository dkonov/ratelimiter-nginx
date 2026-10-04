import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

const target = __ENV.TARGET_URL || 'http://localhost:8080';
const rate = Number(__ENV.RATE || 120);
const duration = __ENV.DURATION || '30s';

const responses200 = new Counter('responses_200');
const responses429 = new Counter('responses_429');
const responsesUnexpected = new Counter('responses_unexpected');

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

  if (res.status === 200) {
    responses200.add(1);
  } else if (res.status === 429) {
    responses429.add(1);
  } else {
    responsesUnexpected.add(1);
  }

  check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
  });
}
