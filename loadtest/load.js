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
  metrics[e.name + '_200'] = new Counter(e.name + '_200');
  metrics[e.name + '_429'] = new Counter(e.name + '_429');
  metrics[e.name + '_unexpected'] = new Counter(e.name + '_unexpected');
  for (const b of e.backends) {
    metrics[e.name + '_' + b] = new Counter(e.name + '_' + b);
  }
}

for (const policy of ['order', 'common', 'search']) {
  metrics[policy + '_200'] = new Counter(policy + '_200');
  metrics[policy + '_429'] = new Counter(policy + '_429');
  metrics[policy + '_unexpected'] = new Counter(policy + '_unexpected');
}

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
  }

  try {
    const body = r.json();
    if (body.backend && metrics[e.name + '_' + body.backend]) {
      metrics[e.name + '_' + body.backend].add(1);
    }
  } catch (_) {}
}

export function java_order() { hit(endpoints[0]); }
export function java_common() { hit(endpoints[1]); }
export function go_order() { hit(endpoints[2]); }
export function go_common() { hit(endpoints[3]); }
export function python_order() { hit(endpoints[4]); }
export function python_search() { hit(endpoints[5]); }
