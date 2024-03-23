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


     data_received..................: 90 MB  45 MB/s
     data_sent......................: 28 MB  14 MB/s
     http_req_blocked...............: avg=8.49µs   min=601ns   med=2.2µs    max=19.48ms p(90)=4.1µs   p(95)=5.51µs  
     http_req_connecting............: avg=291ns    min=0s      med=0s       max=3.88ms  p(90)=0s      p(95)=0s      
     http_req_duration..............: avg=1.61ms   min=41.34µs med=940.17µs max=45.84ms p(90)=3.55ms  p(95)=5.22ms  
       { expected_response:true }...: avg=1.61ms   min=41.34µs med=940.17µs max=45.84ms p(90)=3.55ms  p(95)=5.22ms  
     http_req_failed................: 0.00%  ✓ 0            ✗ 100000
     http_req_receiving.............: avg=116.96µs min=7.68µs  med=28.2µs   max=27.94ms p(90)=49.79µs p(95)=114.21µs
     http_req_sending...............: avg=38.18µs  min=4.39µs  med=14.07µs  max=27.37ms p(90)=22.13µs p(95)=32.8µs  
     http_req_tls_handshaking.......: avg=0s       min=0s      med=0s       max=0s      p(90)=0s      p(95)=0s      
     http_req_waiting...............: avg=1.46ms   min=24.83µs med=872.63µs max=30.15ms p(90)=3.34ms  p(95)=4.77ms  
     http_reqs......................: 100000 49177.172323/s
     iteration_duration.............: avg=1.98ms   min=88.94µs med=1.09ms   max=45.97ms p(90)=4.21ms  p(95)=7.08ms  
     iterations.....................: 100000 49177.172323/s
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
