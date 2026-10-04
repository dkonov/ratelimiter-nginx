import http from 'k6/http';
import { Counter } from 'k6/metrics';

http.setResponseCallback(http.expectedStatuses(200, 429));

const target = __ENV.TARGET_URL || 'http://localhost:8080';
const duration = __ENV.DURATION || '30s';

const endpointDefs = {
  java_order:    { path: '/api/java/order',    rate: Number(__ENV.JAVA_ORDER_RATE || 60) },
  java_common:   { path: '/api/java/common',   rate: Number(__ENV.JAVA_COMMON_RATE || 200) },
  go_ordrer:     { path: '/api/go/ordrer',     rate: Number(__ENV.GO_ORDRER_RATE || 60) },
  go_common:     { path: '/api/go/common',     rate: Number(__ENV.GO_COMMON_RATE || 200) },
  python_order:  { path: '/api/python/order',  rate: Number(__ENV.PYTHON_ORDER_RATE || 60) },
  python_search: { path: '/api/python/search', rate: Number(__ENV.PYTHON_SEARCH_RATE || 20) },
};

const backendNames = [
  'java1', 'java2', 'java3',
  'go1', 'go2', 'go3',
  'python1', 'python2', 'python3',
];

const endpointMetrics = {};
for (const name of Object.keys(endpointDefs)) {
  endpointMetrics[name] = {
    total: new Counter(`${name}_total`),
    ok: new Counter(`${name}_200`),
    limited: new Counter(`${name}_429`),
    other: new Counter(`${name}_other`),
  };
}

const backendMetrics = {};
for (const name of backendNames) {
  backendMetrics[name] = {
    total: new Counter(`backend_${name}_total`),
    ok: new Counter(`backend_${name}_200`),
    limited: new Counter(`backend_${name}_429`),
    other: new Counter(`backend_${name}_other`),
  };
}

function scenario(rate, exec) {
  return {
    executor: 'constant-arrival-rate',
    rate,
    timeUnit: '1s',
    duration,
    preAllocatedVUs: Math.max(20, rate),
    maxVUs: Math.max(100, rate * 4),
    exec,
  };
}

export const options = {
  scenarios: {
    java_order: scenario(endpointDefs.java_order.rate, 'javaOrder'),
    java_common: scenario(endpointDefs.java_common.rate, 'javaCommon'),
    go_ordrer: scenario(endpointDefs.go_ordrer.rate, 'goOrdrer'),
    go_common: scenario(endpointDefs.go_common.rate, 'goCommon'),
    python_order: scenario(endpointDefs.python_order.rate, 'pythonOrder'),
    python_search: scenario(endpointDefs.python_search.rate, 'pythonSearch'),
  },
};

function hit(name) {
  const def = endpointDefs[name];
  const res = http.get(`${target}${def.path}`);
  const m = endpointMetrics[name];
  m.total.add(1);

  if (res.status === 200) m.ok.add(1);
  else if (res.status === 429) m.limited.add(1);
  else m.other.add(1);

  const backend = res.headers['X-Backend-Instance'];
  const bm = backendMetrics[backend];
  if (bm) {
    bm.total.add(1);
    if (res.status === 200) bm.ok.add(1);
    else if (res.status === 429) bm.limited.add(1);
    else bm.other.add(1);
  }
}

export function javaOrder() { hit('java_order'); }
export function javaCommon() { hit('java_common'); }
export function goOrdrer() { hit('go_ordrer'); }
export function goCommon() { hit('go_common'); }
export function pythonOrder() { hit('python_order'); }
export function pythonSearch() { hit('python_search'); }

function count(data, metric) {
  return data.metrics[metric]?.values?.count || 0;
}

function row(cols) {
  return cols.map((v, i) => String(v).padEnd(i === 0 ? 24 : 12)).join('') + '\n';
}

export function handleSummary(data) {
  let out = '\n=== BY ENDPOINT ===\n';
  out += row(['endpoint', 'total', '200', '429', 'other']);

  for (const [name, def] of Object.entries(endpointDefs)) {
    out += row([
      def.path,
      count(data, `${name}_total`),
      count(data, `${name}_200`),
      count(data, `${name}_429`),
      count(data, `${name}_other`),
    ]);
  }

  const policyGroups = {
    'demo:order': ['java_order', 'go_ordrer', 'python_order'],
    'demo:common': ['java_common', 'go_common'],
    'demo:search': ['python_search'],
  };

  out += '\n=== BY SHARED RATE-LIMIT KEY ===\n';
  out += row(['policy', 'total', '200', '429', 'other']);
  for (const [policy, names] of Object.entries(policyGroups)) {
    const total = names.reduce((s, n) => s + count(data, `${n}_total`), 0);
    const ok = names.reduce((s, n) => s + count(data, `${n}_200`), 0);
    const limited = names.reduce((s, n) => s + count(data, `${n}_429`), 0);
    const other = names.reduce((s, n) => s + count(data, `${n}_other`), 0);
    out += row([policy, total, ok, limited, other]);
  }

  out += '\n=== BY BACKEND INSTANCE ===\n';
  out += row(['backend', 'total', '200', '429', 'other']);
  for (const backend of backendNames) {
    out += row([
      backend,
      count(data, `backend_${backend}_total`),
      count(data, `backend_${backend}_200`),
      count(data, `backend_${backend}_429`),
      count(data, `backend_${backend}_other`),
    ]);
  }

  return { stdout: out };
}
