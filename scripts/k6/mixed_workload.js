import http from 'k6/http';
import { check } from 'k6';
import { SharedArray } from 'k6/data';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const PATH_PATTERN = __ENV.PATH_PATTERN || '/kvs/%s';
const DURATION = __ENV.DURATION || '1m';
const RATE = Number(__ENV.RATE) || 200; // RPS
const VUS = Number(__ENV.VUS) || 50;
const READ_RATIO = Number(__ENV.READ_RATIO) || 0.9; // 90% reads, 10% writes
const KEYS = Number(__ENV.KEYS) || 50000;
const VALUE_SIZE = Number(__ENV.VALUE_SIZE) || 128;
const TTL_RATIO = Number(__ENV.TTL_RATIO) || 0;
const TTL_MS = Number(__ENV.TTL_MS) || 0;

export const options = {
  scenarios: {
    mixed: {
      executor: 'constant-arrival-rate',
      rate: RATE,
      timeUnit: '1s',
      duration: DURATION,
      preAllocatedVUs: VUS,
      maxVUs: VUS * 2,
    },
  },
  systemTags: [ 'method', 'status', 'name', 'group', 'check'],
  thresholds: {
    'checks{check:no_5xx}': ['rate==1.0'], // 0 5xx errors
    'http_req_failed{name:PUT /kvs/:key}': ['rate<0.01'], // <1% errors
    http_req_duration: ['p(95)<50'], // 95% of requests should be below 50ms
  },
};

const keys = new SharedArray('keys', () => {
  const arr = [];
  for (let i = 0; i < KEYS; i++) {
    arr.push('k' + String(i).padStart(6, '0'));
  }
  return arr;
});

function pathFor(key) {
  return PATH_PATTERN.replace('%s', key);
}

const value = 'x'.repeat(VALUE_SIZE);

export default function () {
  const key = keys[Math.floor(Math.random() * keys.length)];
  if (Math.random() < READ_RATIO) {
    const res = http.get(`${BASE_URL}${pathFor(key)}`, {
      tags: { name: 'GET /kvs/:key' },
    });
    check(res, {
      '200/404': (r) => r.status === 200 || r.status === 404,
      'no_5xx': (r) => r.status < 500,
    });
    return;
  }
  const body = { value };
  if (TTL_RATIO > 0 && Math.random() < TTL_RATIO) {
    body.ttl = TTL_MS;
  }
  const res = http.put(`${BASE_URL}${pathFor(key)}`, JSON.stringify(body), {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'PUT /kvs/:key', op: 'set'},
  });
  check(res, {
    '2xx': (r) => r.status >= 200 && r.status < 300,
    'no_5xx': (r) => r.status < 500,
  });
}
