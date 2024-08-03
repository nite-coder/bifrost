# Load test

## Bifrost

1. http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 3.1 GB  26 MB/s
     data_sent......................: 1.2 GB  9.7 MB/s
     http_req_blocked...............: avg=7.57µs   min=601ns      med=2.35µs  max=98.66ms  p(75)=2.9µs   p(95)=4.89µs   p(99)=8.48µs   count=3526127
     http_req_connecting............: avg=834ns    min=0s         med=0s      max=88.37ms  p(75)=0s      p(95)=0s       p(99)=0s       count=3526127
     http_req_duration..............: avg=16.49ms  min=203.75µs   med=14.94ms max=136.09ms p(75)=18.64ms p(95)=29.95ms  p(99)=58.74ms  count=3526127
       { expected_response:true }...: avg=16.49ms  min=203.75µs   med=14.94ms max=136.09ms p(75)=18.64ms p(95)=29.95ms  p(99)=58.74ms  count=3526127
     http_req_failed................: 0.00%   ✓ 0            ✗ 3526127
     http_req_receiving.............: avg=698.84µs min=-2494719ns med=33.59µs max=108.55ms p(75)=39.25µs p(95)=261.91µs p(99)=26.03ms  count=3526127
     http_req_sending...............: avg=69.8µs   min=5.43µs     med=16.92µs max=73.99ms  p(75)=19.2µs  p(95)=34.16µs  p(99)=212.06µs count=3526127
     http_req_tls_handshaking.......: avg=0s       min=0s         med=0s      max=0s       p(75)=0s      p(95)=0s       p(99)=0s       count=3526127
     http_req_waiting...............: avg=15.72ms  min=182.68µs   med=14.84ms max=88.29ms  p(75)=18.5ms  p(95)=28ms     p(99)=37.54ms  count=3526127
     http_reqs......................: 3526127 29384.272193/s
     iteration_duration.............: avg=16.94ms  min=371.59µs   med=15.17ms max=142.29ms p(75)=18.97ms p(95)=31.49ms  p(99)=60.87ms  count=3526127
     iterations.....................: 3526127 29384.272193/s
     vus............................: 500     min=500        max=500  
     vus_max........................: 500     min=500        max=500 
```

1. http2 (tls), upstream http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 2.0 GB  17 MB/s
     data_sent......................: 513 MB  4.3 MB/s
     http_req_blocked...............: avg=8.78µs   min=120ns      med=340ns   max=218.8ms  p(75)=361ns   p(95)=491ns    p(99)=711ns   count=2332433
     http_req_connecting............: avg=1.2µs    min=0s         med=0s      max=78.45ms  p(75)=0s      p(95)=0s       p(99)=0s      count=2332433
     http_req_duration..............: avg=25.14ms  min=357.9µs    med=22.46ms max=226.16ms p(75)=30.43ms p(95)=50.92ms  p(99)=76.37ms count=2332433
       { expected_response:true }...: avg=25.14ms  min=357.9µs    med=22.46ms max=226.16ms p(75)=30.43ms p(95)=50.92ms  p(99)=76.37ms count=2332433
     http_req_failed................: 0.00%   ✓ 0            ✗ 2332433
     http_req_receiving.............: avg=6.46ms   min=-1965692ns med=4.17ms  max=159.95ms p(75)=8.02ms  p(95)=20.03ms  p(99)=41.09ms count=2332433
     http_req_sending...............: avg=548.51µs min=-2545608ns med=55.32µs max=144.14ms p(75)=62.4µs  p(95)=202.05µs p(99)=18.68ms count=2332433
     http_req_tls_handshaking.......: avg=6.75µs   min=0s         med=0s      max=214.73ms p(75)=0s      p(95)=0s       p(99)=0s      count=2332433
     http_req_waiting...............: avg=18.12ms  min=0s         med=16.34ms max=221.56ms p(75)=22.45ms p(95)=36.08ms  p(99)=54.84ms count=2332433
     http_reqs......................: 2332433 19435.908606/s
     iteration_duration.............: avg=25.62ms  min=469.78µs   med=22.77ms max=369.31ms p(75)=30.82ms p(95)=52.22ms  p(99)=79.15ms count=2332433
     iterations.....................: 2332433 19435.908606/s
     vus............................: 500     min=500        max=500  
     vus_max........................: 500     min=500        max=500  
```

1. http2 (tls), upstream http2

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 2.3 GB  20 MB/s
     data_sent......................: 871 MB  7.3 MB/s
     http_req_blocked...............: avg=5.99µs   min=641ns      med=2.32µs  max=116.41ms p(75)=2.88µs  p(95)=4.89µs   p(99)=8.03µs  count=2632560
     http_req_connecting............: avg=1.23µs   min=0s         med=0s      max=96.92ms  p(75)=0s      p(95)=0s       p(99)=0s      count=2632560
     http_req_duration..............: avg=22.53ms  min=624.44µs   med=20.92ms max=288.54ms p(75)=26.96ms p(95)=40.14ms  p(99)=53.79ms count=2632560
       { expected_response:true }...: avg=22.53ms  min=624.44µs   med=20.92ms max=288.54ms p(75)=26.96ms p(95)=40.14ms  p(99)=53.79ms count=2632560
     http_req_failed................: 0.00%   ✓ 0           ✗ 2632560
     http_req_receiving.............: avg=378.07µs min=12.24µs    med=33.12µs max=74.01ms  p(75)=40.23µs p(95)=271.23µs p(99)=12.92ms count=2632560
     http_req_sending...............: avg=41.25µs  min=-3165983ns med=16.97µs max=114.58ms p(75)=19.54µs p(95)=37.81µs  p(99)=200.7µs count=2632560
     http_req_tls_handshaking.......: avg=0s       min=0s         med=0s      max=0s       p(75)=0s      p(95)=0s       p(99)=0s      count=2632560
     http_req_waiting...............: avg=22.11ms  min=580.6µs    med=20.82ms max=288.37ms p(75)=26.7ms  p(95)=38.13ms  p(99)=48.65ms count=2632560
     http_reqs......................: 2632560 21935.62693/s
     iteration_duration.............: avg=22.75ms  min=728.73µs   med=21.1ms  max=389.65ms p(75)=27.18ms p(95)=40.59ms  p(99)=54.43ms count=2632560
     iterations.....................: 2632560 21935.62693/s
     vus............................: 500     min=500       max=500  
     vus_max........................: 500     min=500       max=500
```

## openresty

1. HTTP1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 4.1 GB  34 MB/s
     data_sent......................: 1.4 GB  12 MB/s
     http_req_blocked...............: avg=11.79µs  min=0s         med=2.37µs  max=110.95ms p(75)=3.06µs  p(95)=5.34µs   p(99)=10.07µs  count=4376116
     http_req_connecting............: avg=1.82µs   min=0s         med=0s      max=99.14ms  p(75)=0s      p(95)=0s       p(99)=0s       count=4376116
     http_req_duration..............: avg=12.68ms  min=81.39µs    med=10.06ms max=168.33ms p(75)=15.94ms p(95)=29.54ms  p(99)=65.53ms  count=4376116
       { expected_response:true }...: avg=12.68ms  min=81.39µs    med=10.06ms max=168.33ms p(75)=15.94ms p(95)=29.54ms  p(99)=65.53ms  count=4376116
     http_req_failed................: 0.00%   ✓ 0            ✗ 4376116
     http_req_receiving.............: avg=771.64µs min=-2473719ns med=35.39µs max=123.44ms p(75)=40.34µs p(95)=211.48µs p(99)=31.01ms  count=4376116
     http_req_sending...............: avg=99.63µs  min=-2476375ns med=17.33µs max=119.54ms p(75)=19.7µs  p(95)=34.15µs  p(99)=228.76µs count=4376116
     http_req_tls_handshaking.......: avg=0s       min=0s         med=0s      max=0s       p(75)=0s      p(95)=0s       p(99)=0s       count=4376116
     http_req_waiting...............: avg=11.81ms  min=0s         med=9.92ms  max=100.72ms p(75)=15.74ms p(95)=27.92ms  p(99)=39.07ms  count=4376116
     http_reqs......................: 4376116 36468.037704/s
     iteration_duration.............: avg=13.57ms  min=214.74µs   med=10.61ms max=187.54ms p(75)=16.72ms p(95)=32.33ms  p(99)=71.4ms   count=4376116
     iterations.....................: 4376116 36468.037704/s
     vus............................: 500     min=500        max=500  
     vus_max........................: 500     min=500        max=500  
```

1. http2 (tls), upstream http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 2.9 GB  24 MB/s
     data_sent......................: 695 MB  5.8 MB/s
     http_req_blocked...............: avg=24.29µs  min=120ns      med=340ns   max=205.06ms p(75)=361ns   p(95)=491ns    p(99)=711ns   count=3151787
     http_req_connecting............: avg=3.21µs   min=0s         med=0s      max=84.77ms  p(75)=0s      p(95)=0s       p(99)=0s      count=3151787
     http_req_duration..............: avg=18.38ms  min=-844131ns  med=15.21ms max=194.76ms p(75)=22.5ms  p(95)=42.21ms  p(99)=70.87ms count=3151787
       { expected_response:true }...: avg=18.38ms  min=-844131ns  med=15.21ms max=194.76ms p(75)=22.5ms  p(95)=42.21ms  p(99)=70.87ms count=3151787
     http_req_failed................: 0.00%   ✓ 0            ✗ 3151787
     http_req_receiving.............: avg=4.12ms   min=-2896971ns med=1.67ms  max=164.25ms p(75)=4.66ms  p(95)=15.83ms  p(99)=36.28ms count=3151787
     http_req_sending...............: avg=562.01µs min=-2617373ns med=54.55µs max=155.24ms p(75)=61.14µs p(95)=212.23µs p(99)=18.38ms count=3151787
     http_req_tls_handshaking.......: avg=19.34µs  min=0s         med=0s      max=197.15ms p(75)=0s      p(95)=0s       p(99)=0s      count=3151787
     http_req_waiting...............: avg=13.69ms  min=0s         med=11.99ms max=190.26ms p(75)=17.18ms p(95)=29.5ms   p(99)=43.94ms count=3151787
     http_reqs......................: 3151787 26263.310618/s
     iteration_duration.............: avg=18.93ms  min=331.39µs   med=15.59ms max=229.06ms p(75)=23.03ms p(95)=43.8ms   p(99)=73.59ms count=3151787
     iterations.....................: 3151787 26263.310618/s
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
