raw

```sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 100 max VUs, 10m30s max duration (incl. graceful stop):
              * default: 100000 iterations shared among 100 VUs (maxDuration: 10m0s, gracefulStop: 30s)


     data_received..................: 90 MB  44 MB/s
     data_sent......................: 28 MB  13 MB/s
     http_req_blocked...............: avg=8.64µs  min=621ns   med=2.23µs   max=13.94ms p(90)=4.14µs p(95)=5.64µs  
     http_req_connecting............: avg=590ns   min=0s      med=0s       max=3ms     p(90)=0s     p(95)=0s      
     http_req_duration..............: avg=1.67ms  min=56.21µs med=1.03ms   max=35.48ms p(90)=3.67ms p(95)=5.24ms  
       { expected_response:true }...: avg=1.67ms  min=56.21µs med=1.03ms   max=35.48ms p(90)=3.67ms p(95)=5.24ms  
     http_req_failed................: 0.00%  ✓ 0            ✗ 100000
     http_req_receiving.............: avg=90.42µs min=7.2µs   med=28.46µs  max=31.71ms p(90)=51µs   p(95)=118.83µs
     http_req_sending...............: avg=38.43µs min=4.31µs  med=14.23µs  max=26.79ms p(90)=22.4µs p(95)=35.52µs 
     http_req_tls_handshaking.......: avg=0s      min=0s      med=0s       max=0s      p(90)=0s     p(95)=0s      
     http_req_waiting...............: avg=1.54ms  min=28.9µs  med=969.85µs max=29.81ms p(90)=3.49ms p(95)=4.9ms   
     http_reqs......................: 100000 48304.366462/s
     iteration_duration.............: avg=2.02ms  min=106.1µs med=1.2ms    max=41.54ms p(90)=4.3ms  p(95)=6.7ms   
     iterations.....................: 100000 48304.366462/s
     vus............................: 100    min=100        max=100 
     vus_max........................: 100    min=100        max=100 
```

fiber

```sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 100 max VUs, 10m30s max duration (incl. graceful stop):
              * default: 100000 iterations shared among 100 VUs (maxDuration: 10m0s, gracefulStop: 30s)


     data_received..................: 25 MB  10 MB/s
     data_sent......................: 28 MB  12 MB/s
     http_req_blocked...............: avg=12.79µs  min=721ns    med=2.5µs   max=21.78ms p(90)=4.63µs  p(95)=6.28µs  
     http_req_connecting............: avg=3.33µs   min=0s       med=0s      max=15ms    p(90)=0s      p(95)=0s      
     http_req_duration..............: avg=1.9ms    min=63.29µs  med=1.2ms   max=47.34ms p(90)=4.14ms  p(95)=5.82ms  
       { expected_response:true }...: avg=1.9ms    min=63.29µs  med=1.2ms   max=47.34ms p(90)=4.14ms  p(95)=5.82ms  
     http_req_failed................: 0.00%  ✓ 0            ✗ 100000
     http_req_receiving.............: avg=104.68µs min=8.34µs   med=29.19µs max=41.87ms p(90)=69.43µs p(95)=186.23µs
     http_req_sending...............: avg=48.77µs  min=4.85µs   med=15.24µs max=31.53ms p(90)=25.97µs p(95)=70.76µs 
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s      p(90)=0s      p(95)=0s      
     http_req_waiting...............: avg=1.75ms   min=34.48µs  med=1.12ms  max=25.04ms p(90)=3.93ms  p(95)=5.44ms  
     http_reqs......................: 100000 41696.023603/s
     iteration_duration.............: avg=2.31ms   min=120.98µs med=1.39ms  max=50.14ms p(90)=4.82ms  p(95)=7.48ms  
     iterations.....................: 100000 41696.023603/s
     vus............................: 100    min=100        max=100 
     vus_max........................: 100    min=100        max=100 
```

Hertz

``` sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 100 max VUs, 10m30s max duration (incl. graceful stop):
              * default: 100000 iterations shared among 100 VUs (maxDuration: 10m0s, gracefulStop: 30s)


     data_received..................: 90 MB  26 MB/s
     data_sent......................: 28 MB  8.2 MB/s
     http_req_blocked...............: avg=6.69µs   min=671ns    med=2.47µs  max=14.74ms p(90)=4.4µs   p(95)=5.7µs  
     http_req_connecting............: avg=168ns    min=0s       med=0s      max=1.81ms  p(90)=0s      p(95)=0s     
     http_req_duration..............: avg=3.11ms   min=154.94µs med=2.55ms  max=33.48ms p(90)=5.77ms  p(95)=7.6ms  
       { expected_response:true }...: avg=3.11ms   min=154.94µs med=2.55ms  max=33.48ms p(90)=5.77ms  p(95)=7.6ms  
     http_req_failed................: 0.00%  ✓ 0            ✗ 100000
     http_req_receiving.............: avg=111.08µs min=10.19µs  med=32.12µs max=28.92ms p(90)=65.35µs p(95)=166.6µs
     http_req_sending...............: avg=38.8µs   min=4.92µs   med=15.51µs max=17.52ms p(90)=24.09µs p(95)=49.9µs 
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s      p(90)=0s      p(95)=0s     
     http_req_waiting...............: avg=2.96ms   min=114.33µs med=2.46ms  max=23.65ms p(90)=5.53ms  p(95)=7.13ms 
     http_reqs......................: 100000 29154.140074/s
     iteration_duration.............: avg=3.38ms   min=247.43µs med=2.72ms  max=35.54ms p(90)=6.25ms  p(95)=8.52ms 
     iterations.....................: 100000 29154.140074/s
     vus............................: 100    min=100        max=100 
     vus_max........................: 100    min=100        max=100
```

openresty

``` sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 100 max VUs, 10m30s max duration (incl. graceful stop):
              * default: 100000 iterations shared among 100 VUs (maxDuration: 10m0s, gracefulStop: 30s)


     data_received..................: 94 MB  19 MB/s
     data_sent......................: 27 MB  5.6 MB/s
     http_req_blocked...............: avg=13.59µs  min=801ns    med=2.81µs  max=21.18ms p(90)=5.17µs  p(95)=6.62µs  
     http_req_connecting............: avg=414ns    min=0s       med=0s      max=1.78ms  p(90)=0s      p(95)=0s      
     http_req_duration..............: avg=4.48ms   min=284.23µs med=3.63ms  max=60.88ms p(90)=8.49ms  p(95)=10.66ms 
       { expected_response:true }...: avg=4.48ms   min=284.23µs med=3.63ms  max=60.88ms p(90)=8.49ms  p(95)=10.66ms 
     http_req_failed................: 0.00%  ✓ 0            ✗ 100000
     http_req_receiving.............: avg=158.39µs min=14.08µs  med=37.84µs max=48.99ms p(90)=88.74µs p(95)=234.93µs
     http_req_sending...............: avg=61.52µs  min=6.51µs   med=16.9µs  max=35.69ms p(90)=28.23µs p(95)=64.73µs 
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s      p(90)=0s      p(95)=0s      
     http_req_waiting...............: avg=4.26ms   min=245.36µs med=3.51ms  max=34.47ms p(90)=8.16ms  p(95)=10.18ms 
     http_reqs......................: 100000 20351.729597/s
     iteration_duration.............: avg=4.85ms   min=385.02µs med=3.85ms  max=61ms    p(90)=9.15ms  p(95)=11.81ms 
     iterations.....................: 100000 20351.729597/s
     vus............................: 100    min=100        max=100 
     vus_max........................: 100    min=100        max=100 
```
