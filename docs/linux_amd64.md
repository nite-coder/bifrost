# Load test

## Bifrost

1. http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 3.5 GB  29 MB/s
     data_sent......................: 1.2 GB  10 MB/s
     http_req_blocked...............: avg=8.08µs   min=611ns      med=2.38µs  max=170.66ms p(75)=2.89µs  p(95)=5.1µs    p(99)=8.41µs   count=3708266
     http_req_connecting............: avg=2.11µs   min=0s         med=0s      max=170.54ms p(75)=0s      p(95)=0s       p(99)=0s       count=3708266
     http_req_duration..............: avg=15.66ms  min=219.96µs   med=14.06ms max=160.46ms p(75)=18.11ms p(95)=28.85ms  p(99)=55.61ms  count=3708266
       { expected_response:true }...: avg=15.66ms  min=219.96µs   med=14.06ms max=160.46ms p(75)=18.11ms p(95)=28.85ms  p(99)=55.61ms  count=3708266
     http_req_failed................: 0.00%   ✓ 0            ✗ 3708266
     http_req_receiving.............: avg=678.54µs min=-1276568ns med=34.8µs  max=108.36ms p(75)=39.59µs p(95)=279.73µs p(99)=23.9ms   count=3708266
     http_req_sending...............: avg=67.09µs  min=-2070129ns med=16.78µs max=141.37ms p(75)=18.79µs p(95)=31.97µs  p(99)=214.46µs count=3708266
     http_req_tls_handshaking.......: avg=0s       min=0s         med=0s      max=0s       p(75)=0s      p(95)=0s       p(99)=0s       count=3708266
     http_req_waiting...............: avg=14.91ms  min=168.83µs   med=13.96ms max=127.26ms p(75)=17.96ms p(95)=26.93ms  p(99)=35.83ms  count=3708266
     http_reqs......................: 3708266 30901.430774/s
     iteration_duration.............: avg=16.09ms  min=308.82µs   med=14.31ms max=196.52ms p(75)=18.43ms p(95)=30.33ms  p(99)=58.34ms  count=3708266
     iterations.....................: 3708266 30901.430774/s
     vus............................: 500     min=500        max=500  
     vus_max........................: 500     min=500        max=500  
```

## openresty

1. HTTP1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 2.8 GB  24 MB/s
     data_sent......................: 992 MB  8.3 MB/s
     http_req_blocked...............: avg=19.04µs  min=662ns      med=2.43µs  max=168.59ms p(75)=3.01µs  p(95)=5.04µs   p(99)=8.92µs   count=3015654
     http_req_connecting............: avg=10.98µs  min=0s         med=0s      max=136.82ms p(75)=0s      p(95)=0s       p(99)=0s       count=3015654
     http_req_duration..............: avg=19.41ms  min=289.58µs   med=17.79ms max=128.43ms p(75)=21.38ms p(95)=33.26ms  p(99)=66.77ms  count=3015654
       { expected_response:true }...: avg=19.41ms  min=289.58µs   med=17.79ms max=128.43ms p(75)=21.38ms p(95)=33.26ms  p(99)=66.77ms  count=3015654
     http_req_failed................: 0.00%   ✓ 0            ✗ 3015654
     http_req_receiving.............: avg=910.03µs min=-1191062ns med=35.1µs  max=100.75ms p(75)=43.53µs p(95)=351.77µs p(99)=30.62ms  count=3015654
     http_req_sending...............: avg=80.68µs  min=5.55µs     med=16.42µs max=89.03ms  p(75)=18.58µs p(95)=32.91µs  p(99)=228.78µs count=3015654
     http_req_tls_handshaking.......: avg=0s       min=0s         med=0s      max=0s       p(75)=0s      p(95)=0s       p(99)=0s       count=3015654
     http_req_waiting...............: avg=18.42ms  min=258.49µs   med=17.69ms max=125.86ms p(75)=21.23ms p(95)=30.47ms  p(99)=40.66ms  count=3015654
     http_reqs......................: 3015654 25127.363598/s
     iteration_duration.............: avg=19.83ms  min=399.47µs   med=17.99ms max=234.11ms p(75)=21.65ms p(95)=34.89ms  p(99)=69.2ms   count=3015654
     iterations.....................: 3015654 25127.363598/s
     vus............................: 500     min=500        max=500  
     vus_max........................: 500     min=500        max=500  
```

## Yarp

```sh
     execution: local
        script: create_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 40s max duration (incl. graceful stop):
              * default: 500 looping VUs for 10s (gracefulStop: 30s)


     data_received..................: 211 MB 21 MB/s
     data_sent......................: 77 MB  7.7 MB/s
     http_req_blocked...............: avg=48.67µs min=641ns    med=2.48µs  max=171.91ms p(90)=4.88µs   p(95)=6.25µs 
     http_req_connecting............: avg=8.72µs  min=0s       med=0s      max=111.65ms p(90)=0s       p(95)=0s     
     http_req_duration..............: avg=20.29ms min=283.59µs med=16.8ms  max=192.7ms  p(90)=35.34ms  p(95)=48.96ms
       { expected_response:true }...: avg=20.29ms min=283.59µs med=16.8ms  max=192.7ms  p(90)=35.34ms  p(95)=48.96ms
     http_req_failed................: 0.00%  ✓ 0            ✗ 233036
     http_req_receiving.............: avg=1.76ms  min=12.66µs  med=35.67µs max=105.92ms p(90)=356.78µs p(95)=15.1ms 
     http_req_sending...............: avg=144.6µs min=5.35µs   med=17.13µs max=171.24ms p(90)=36.4µs   p(95)=122.7µs
     http_req_tls_handshaking.......: avg=0s      min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s     
     http_req_waiting...............: avg=18.38ms min=233.49µs med=16.62ms max=140.01ms p(90)=30.58ms  p(95)=36.61ms
     http_reqs......................: 233036 23280.636299/s
     iteration_duration.............: avg=21.15ms min=422.57µs med=17.23ms max=195.74ms p(90)=37.65ms  p(95)=51.98ms
     iterations.....................: 233036 23280.636299/s
     vus............................: 500    min=500        max=500 
     vus_max........................: 500    min=500        max=500 
```
