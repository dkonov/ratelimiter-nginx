import http from 'k6/http';
import { Counter } from 'k6/metrics';

http.setResponseCallback(http.expectedStatuses(200, 429));

const target = __ENV.TARGET_URL || 'http://127.0.0.1:8080';
const duration = __ENV.DURATION || '30s';

const endpoints = [
  {name:'java_order', policy:'order', path:'/api/java/order', rate:Number(__ENV.JAVA_ORDER_RPS || 120), backends:['java1','java2','java3']},
  {name:'java_common', policy:'common', path:'/api/java/common', rate:Number(__ENV.JAVA_COMMON_RPS || 180), backends:['java1','java2','java3']},
  {name:'go_order', policy:'order', path:'/api/go/ordrer', rate:Number(__ENV.GO_ORDER_RPS || 120), backends:['go1','go2','go3']},
  {name:'go_common', policy:'common', path:'/api/go/common', rate:Number(__ENV.GO_COMMON_RPS || 180), backends:['go1','go2','go3']},
  {name:'python_order', policy:'order', path:'/api/python/order', rate:Number(__ENV.PYTHON_ORDER_RPS || 120), backends:['python1','python2','python3']},
  {name:'python_search', policy:'search', path:'/api/python/search', rate:Number(__ENV.PYTHON_SEARCH_RPS || 30), backends:['python1','python2','python3']},
];

const metrics = {};

for (const e of endpoints) {
  for (const suffix of ['200', '429', 'bypass', 'unavailable', 'unexpected']) {
    metrics[e.name + '_' + suffix] = new Counter(e.name + '_' + suffix);
  }
  for (const b of e.backends) {
    metrics[e.name + '_' + b] = new Counter(e.name + '_' + b);
  }
}

for (const policy of ['order', 'common', 'search']) {
  for (const suffix of ['200', '429', 'bypass', 'unavailable', 'unexpected']) {
    metrics[policy + '_' + suffix] = new Counter(policy + '_' + suffix);
  }
}

metrics.http_500 = new Counter('http_500');
metrics.http_502 = new Counter('http_502');
metrics.http_503 = new Counter('http_503');
metrics.http_other_unexpected = new Counter('http_other_unexpected');

export const options = { scenarios: {} };
for (const e of endpoints) {
  options.scenarios[e.name] = {
    executor: 'constant-arrival-rate',
    exec: e.name,
    rate: e.rate,
    timeUnit: '1s',
    duration,
    preAllocatedVUs: Math.max(20, e.rate),
    maxVUs: Math.max(100, e.rate * 3),
  };
}

function headerTrue(headers, wanted) {
  const key = Object.keys(headers).find(k => k.toLowerCase() === wanted.toLowerCase());
  return key ? String(headers[key]).toLowerCase() === 'true' : false;
}

function hit(e) {
  const r = http.get(target + e.path);

  if (r.status === 200) {
    metrics[e.name + '_200'].add(1);
    metrics[e.policy + '_200'].add(1);
  } else if (r.status === 429) {
    metrics[e.name + '_429'].add(1);
    metrics[e.policy + '_429'].add(1);
  } else {
    metrics[e.name + '_unexpected'].add(1);
    metrics[e.policy + '_unexpected'].add(1);

    if (r.status === 500) metrics.http_500.add(1);
    else if (r.status === 502) metrics.http_502.add(1);
    else if (r.status === 503) metrics.http_503.add(1);
    else metrics.http_other_unexpected.add(1);
  }

  let body = null;
  try {
    body = r.json();
  } catch (_) {}

  const bypass = Boolean(body && body.bypass === true) || headerTrue(r.headers, 'X-RateLimit-Bypass');
  const unavailable = Boolean(body && body.unavailable === true) || headerTrue(r.headers, 'X-RateLimit-Unavailable');

  metrics[e.name + '_bypass'].add(bypass ? 1 : 0);
  metrics[e.policy + '_bypass'].add(bypass ? 1 : 0);
  metrics[e.name + '_unavailable'].add(unavailable ? 1 : 0);
  metrics[e.policy + '_unavailable'].add(unavailable ? 1 : 0);

  if (body && body.backend && metrics[e.name + '_' + body.backend]) {
    metrics[e.name + '_' + body.backend].add(1);
  }
}

export function java_order() { hit(endpoints[0]); }
export function java_common() { hit(endpoints[1]); }
export function go_order() { hit(endpoints[2]); }
export function go_common() { hit(endpoints[3]); }
export function python_order() { hit(endpoints[4]); }
export function python_search() { hit(endpoints[5]); }
