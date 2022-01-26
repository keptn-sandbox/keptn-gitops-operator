import http from 'k6/http';
import { check } from 'k6';
import { Rate } from 'k6/metrics';

export let errorRate = new Rate('errors');
export let options = {
  thresholds: {
    errors: ['rate<0.1'], // <10% errors
    http_req_duration: ['p(95)<500']
  },
};

export default function () {
  const res = http.get('http://podtato-${__ENV.SERVICE}-preview.podtatohead-${__ENV.STAGE}.svc.cluster.local:8080/${__ENV.SUBPATH}');
  const result = check(res, {
    'status is 200': (r) => r.status == 200,
  });

  errorRate.add(!result);
}
