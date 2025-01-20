// import http from 'k6/http';
// import { sleep, check } from 'k6';
// import { Rate } from 'k6/metrics';

// const errorRate = new Rate('errors');

// export const options = {
//   stages: [
//     { duration: '30s', target: 20 },
//     { duration: '1m', target: 20 },
//     { duration: '20s', target: 50 },
//     { duration: '30s', target: 0 },
//   ],
// };

// const ENDPOINTS = [
//   'https://httpbin.org/get',
//   'https://httpbin.org/headers',
//   'https://httpbin.org/json',
//   'http://httpbin.org/delay/1',
//   'http://httpbin.org/bytes/50000',
//   'https://example.com',
//   'https://httpbin.org/get',
//   'http://example.com',
//   'http://neverssl.com',
//   'https://neverssl.com'
// ];

// export default function () {
//   const url = ENDPOINTS[Math.floor(Math.random() * ENDPOINTS.length)];
  
//   // THIS IS THE KEY CHANGE - proper proxy config
//   const params = {
//     proxy: {
//       url: 'http://localhost:8080',
//     }
//   };

//   const response = http.get(url, params);
  
//   check(response, {
//     'status is 200': (r) => r.status === 200,
//     'response time < 2s': (r) => r.timings.duration < 2000,
//   });

//   sleep(Math.random() * 0.9 + 0.1);
// }

import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
    vus: 50,  // Number of virtual users
    duration: '30s',  // Load test duration
};

export default function () {
    const targetUrl = 'http://www.example.com';

    // Using environment variable to set proxy
    let proxyUrl = __ENV.HTTP_PROXY || 'http://localhost:8080';

    let params = {
        headers: {
            'Host': 'www.example.com',
        },
        timeout: '60s',  // Increase timeout if needed
    };

    let res = http.get(targetUrl, params);

    check(res, {
        'is status 200': (r) => r.status === 200,
    });

    sleep(1);
}