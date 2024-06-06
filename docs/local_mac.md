## Bifrost

access_log: off

```sh
     execution: local
        script: create_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 1m1s max duration (incl. graceful stop):
              * default: Up to 500 looping VUs for 31s over 2 stages (gracefulRampDown: 30s, gracefulStop: 30s)


     data_received..................: 1.1 GB  35 MB/s
     data_sent......................: 399 MB  13 MB/s
     http_req_blocked...............: avg=8.96µs  min=0s       med=0s     max=120.43ms p(90)=1µs     p(95)=1µs
     http_req_connecting............: avg=877ns   min=0s       med=0s     max=39.81ms  p(90)=0s      p(95)=0s
     http_req_duration..............: avg=11.25ms min=72µs     med=9.47ms max=208.16ms p(90)=17.5ms  p(95)=25.95ms
       { expected_response:true }...: avg=11.25ms min=72µs     med=9.47ms max=208.16ms p(90)=17.5ms  p(95)=25.95ms
     http_req_failed................: 0.00%   ✓ 0            ✗ 1206360
     http_req_receiving.............: avg=1.04ms  min=2µs      med=6µs    max=167.87ms p(90)=26µs    p(95)=157µs
     http_req_sending...............: avg=55.22µs min=1µs      med=3µs    max=154.57ms p(90)=6µs     p(95)=14µs
     http_req_tls_handshaking.......: avg=0s      min=0s       med=0s     max=0s       p(90)=0s      p(95)=0s
     http_req_waiting...............: avg=10.15ms min=65µs     med=9.41ms max=138.56ms p(90)=16.92ms p(95)=20.94ms
     http_reqs......................: 1206360 38908.402507/s
     iteration_duration.............: avg=12.22ms min=101.29µs med=9.86ms max=208.19ms p(90)=19.27ms p(95)=31.66ms
     iterations.....................: 1206360 38908.402507/s
     vus............................: 500     min=500        max=500
     vus_max........................: 500     min=500        max=500
```




