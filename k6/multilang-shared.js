import http from 'k6/http';
import { check } from 'k6';
import { Counter } from 'k6/metrics';

http.setResponseCallback(http.expectedStatuses(200, 429));

const target = __ENV.TARGET_URL || 'http://localhost:8080';
const ratePython = Number(__ENV.RATE_PYTHON || 10);
const rateJava = Number(__ENV.RATE_JAVA || 10);
const duration = __ENV.DURATION || '30s';

const python200 = new Counter('python_200');
const python429 = new Counter('python_429');
const java200 = new Counter('java_200');
const java429 = new Counter('java_429');
const total200 = new Counter('shared_200');
const total429 = new Counter('shared_429');
const unexpected = new Counter('responses_unexpected');

export const options = {
  scenarios: {
    python: { executor: 'constant-arrival-rate', exec: 'pythonTraffic', rate: ratePython, timeUnit: '1s', duration, preAllocatedVUs: 50, maxVUs: 300 },
    java: { executor: 'constant-arrival-rate', exec: 'javaTraffic', rate: rateJava, timeUnit: '1s', duration, preAllocatedVUs: 50, maxVUs: 300 },
  },
};

function record(res, okCounter, deniedCounter) {
  if (res.status === 200) {
    okCounter.add(1); total200.add(1);
  } else if (res.status === 429) {
    deniedCounter.add(1); total429.add(1);
  } else {
    unexpected.add(1);
  }
  check(res, { 'status is 200 or 429': r => r.status === 200 || r.status === 429 });
}

export function pythonTraffic() { record(http.get(`${target}/python/report`), python200, python429); }
export function javaTraffic() { record(http.get(`${target}/java/orders`), java200, java429); }
