import http from 'k6/http';
export const options = {
  scenarios: { ramp: { executor: 'ramping-arrival-rate', startRate: 100, timeUnit: '1s', preAllocatedVUs: 2000, maxVUs: 8000,
    stages: [{ target: 2000, duration: '30s' }, { target: 10000, duration: '30s' }, { target: 10000, duration: '3m' }, { target: 15000, duration: '30s' }] } },
  thresholds: { http_req_duration: ['p(95)<10', 'p(99)<15'], 'http_req_failed{status:500}': ['rate==0'] },
};
export default () => http.get('http://localhost:8080/', { headers: { 'X-API-Key': `k${__VU}` } });
