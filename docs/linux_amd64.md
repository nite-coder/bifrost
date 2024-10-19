# Load test

因為環境與參數的不同對壓測結果有很大的影響，數據僅供參考

CPU: AMD Ryzen7 4750U
Ram: 16GB
OS: Debian 12 (docker)
Date: 2024-09-01
Golang: 1.23

client:

```sh
taskset -c 0,1,2,3,4,5,6,7 k6 run vus.js
```

## Bifrost

1. http1.1, upstream http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 2.9 GB  24 MB/s
     data_sent......................: 1.1 GB  9.0 MB/s
     http_req_blocked...............: avg=8.53µs   min=601ns      med=2.42µs  max=220.41ms p(75)=3.02µs  p(95)=5.02µs   p(99)=8.42µs  count=3271825
     http_req_connecting............: avg=2.17µs   min=0s         med=0s      max=141.49ms p(75)=0s      p(95)=0s       p(99)=0s      count=3271825
     http_req_duration..............: avg=16.79ms  min=274.7µs    med=14.19ms max=296.72ms p(75)=20.41ms p(95)=37.15ms  p(99)=64.13ms count=3271825
       { expected_response:true }...: avg=16.79ms  min=274.7µs    med=14.19ms max=296.72ms p(75)=20.41ms p(95)=37.15ms  p(99)=64.13ms count=3271825
     http_req_failed................: 0.00%   ✓ 0            ✗ 3271825
     http_req_receiving.............: avg=543.32µs min=-4213154ns med=32.34µs max=147.56ms p(75)=36.54µs p(95)=215.81µs p(99)=20.23ms count=3271825
     http_req_sending...............: avg=59.74µs  min=-1631134ns med=17.15µs max=274.64ms p(75)=19.31µs p(95)=36.38µs  p(99)=196.6µs count=3271825
     http_req_tls_handshaking.......: avg=0s       min=0s         med=0s      max=0s       p(75)=0s      p(95)=0s       p(99)=0s      count=3271825
     http_req_waiting...............: avg=16.19ms  min=238.52µs   med=14.09ms max=272.91ms p(75)=20.24ms p(95)=35.52ms  p(99)=49.64ms count=3271825
     http_reqs......................: 3271825 27265.437027/s
     iteration_duration.............: avg=18.14ms  min=354.08µs   med=15.06ms max=345.17ms p(75)=21.75ms p(95)=41.19ms  p(99)=71.91ms count=3271825
     iterations.....................: 3271825 27265.437027/s
     vus............................: 500     min=500        max=500  
     vus_max........................: 500     min=500        max=500 
```

1. http1.1 (tls), upstream http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 2.7 GB  22 MB/s
     data_sent......................: 1.0 GB  8.7 MB/s
     http_req_blocked...............: avg=42.72µs  min=631ns      med=2.52µs  max=401.02ms p(75)=3.19µs  p(95)=5.14µs   p(99)=8.3µs    count=2949700
     http_req_connecting............: avg=8.36µs   min=0s         med=0s      max=106.12ms p(75)=0s      p(95)=0s       p(99)=0s       count=2949700
     http_req_duration..............: avg=18.7ms   min=300.33µs   med=15.57ms max=264.64ms p(75)=23.2ms  p(95)=42.14ms  p(99)=73.06ms  count=2949700
       { expected_response:true }...: avg=18.7ms   min=300.33µs   med=15.57ms max=264.64ms p(75)=23.2ms  p(95)=42.14ms  p(99)=73.06ms  count=2949700
     http_req_failed................: 0.00%   ✓ 0            ✗ 2949700
     http_req_receiving.............: avg=679.48µs min=-736381ns  med=33.47µs max=172.66ms p(75)=37.83µs p(95)=240.95µs p(99)=25.22ms  count=2949700
     http_req_sending...............: avg=68.09µs  min=-1849143ns med=17.47µs max=173.7ms  p(75)=19.61µs p(95)=35.38µs  p(99)=203.45µs count=2949700
     http_req_tls_handshaking.......: avg=27.88µs  min=0s         med=0s      max=288.39ms p(75)=0s      p(95)=0s       p(99)=0s       count=2949700
     http_req_waiting...............: avg=17.95ms  min=258.75µs   med=15.45ms max=197.8ms  p(75)=22.98ms p(95)=39.94ms  p(99)=55.29ms  count=2949700
     http_reqs......................: 2949700 24580.603728/s
     iteration_duration.............: avg=20.15ms  min=425.18µs   med=16.57ms max=453.91ms p(75)=24.57ms p(95)=46.68ms  p(99)=80.45ms  count=2949700
     iterations.....................: 2949700 24580.603728/s
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


     data_received..................: 1.7 GB  14 MB/s
     data_sent......................: 435 MB  3.6 MB/s
     http_req_blocked...............: avg=58.52µs  min=120ns      med=371ns   max=547.87ms p(75)=391ns   p(95)=491ns    p(99)=751ns    count=1977558
     http_req_connecting............: avg=13.53µs  min=0s         med=0s      max=118.37ms p(75)=0s      p(95)=0s       p(99)=0s       count=1977558
     http_req_duration..............: avg=29.02ms  min=387.75µs   med=24.28ms max=292.79ms p(75)=35.82ms p(95)=66.09ms  p(99)=103.34ms count=1977558
       { expected_response:true }...: avg=29.02ms  min=387.75µs   med=24.28ms max=292.79ms p(75)=35.82ms p(95)=66.09ms  p(99)=103.34ms count=1977558
     http_req_failed................: 0.00%   ✓ 0            ✗ 1977558
     http_req_receiving.............: avg=10.51ms  min=12.66µs    med=6.76ms  max=257.17ms p(75)=13.66ms p(95)=33.05ms  p(99)=63.76ms  count=1977558
     http_req_sending...............: avg=547.18µs min=-1442297ns med=63.71µs max=164.15ms p(75)=72.33µs p(95)=205.95µs p(99)=11.21ms  count=1977558
     http_req_tls_handshaking.......: avg=43.83µs  min=0s         med=0s      max=444.59ms p(75)=0s      p(95)=0s       p(99)=0s       count=1977558
     http_req_waiting...............: avg=17.95ms  min=0s         med=15.35ms max=290.92ms p(75)=22.28ms p(95)=40.06ms  p(99)=65.1ms   count=1977558
     http_reqs......................: 1977558 16478.431928/s
     iteration_duration.............: avg=30.17ms  min=482.32µs   med=25.08ms max=557.07ms p(75)=36.9ms  p(95)=69.36ms  p(99)=107.86ms count=1977558
     iterations.....................: 1977558 16478.431928/s
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


     data_received..................: 1.5 GB  13 MB/s
     data_sent......................: 390 MB  3.2 MB/s
     http_req_blocked...............: avg=27.27µs  min=120ns      med=351ns   max=226.33ms p(75)=371ns   p(95)=461ns    p(99)=722ns   count=1774737
     http_req_connecting............: avg=4.45µs   min=0s         med=0s      max=194.56ms p(75)=0s      p(95)=0s       p(99)=0s      count=1774737
     http_req_duration..............: avg=33.45ms  min=102.52µs   med=31.49ms max=3.02s    p(75)=39.03ms p(95)=56.67ms  p(99)=76.64ms count=1774737
       { expected_response:true }...: avg=33.18ms  min=102.52µs   med=31.49ms max=193.4ms  p(75)=39.03ms p(95)=56.65ms  p(99)=76.53ms count=1774576
     http_req_failed................: 0.00%   ✓ 161          ✗ 1774576
     http_req_receiving.............: avg=3.36ms   min=-1031633ns med=1.72ms  max=2.98s    p(75)=3.81ms  p(95)=11.79ms  p(99)=28.51ms count=1774737
     http_req_sending...............: avg=355.22µs min=-2353325ns med=60.82µs max=82.19ms  p(75)=69.53µs p(95)=175.25µs p(99)=10.96ms count=1774737
     http_req_tls_handshaking.......: avg=21.31µs  min=0s         med=0s      max=214.8ms  p(75)=0s      p(95)=0s       p(99)=0s      count=1774737
     http_req_waiting...............: avg=29.72ms  min=0s         med=28.65ms max=3.02s    p(75)=35.32ms p(95)=48.38ms  p(99)=62.03ms count=1774737
     http_reqs......................: 1774737 14431.768257/s
     iteration_duration.............: avg=33.75ms  min=818.25µs   med=31.7ms  max=3.02s    p(75)=39.29ms p(95)=57.28ms  p(99)=78.14ms count=1774737
     iterations.....................: 1774737 14431.768257/s
     vus............................: 1       min=1          max=500  
     vus_max........................: 500     min=500        max=500 
```

## openresty

1. HTTP1.1, upstream http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 2.8 GB  23 MB/s
     data_sent......................: 978 MB  8.1 MB/s
     http_req_blocked...............: avg=21.72µs  min=631ns      med=2.81µs  max=202.28ms p(75)=3.83µs  p(95)=5.68µs   p(99)=9.84µs  count=2954063
     http_req_connecting............: avg=12.42µs  min=0s         med=0s      max=201.07ms p(75)=0s      p(95)=0s       p(99)=0s      count=2954063
     http_req_duration..............: avg=16.51ms  min=167.97µs   med=13.11ms max=218.46ms p(75)=21.8ms  p(95)=40.75ms  p(99)=70.55ms count=2954063
       { expected_response:true }...: avg=16.51ms  min=167.97µs   med=13.11ms max=218.46ms p(75)=21.8ms  p(95)=40.75ms  p(99)=70.55ms count=2954063
     http_req_failed................: 0.00%   ✓ 0            ✗ 2954063
     http_req_receiving.............: avg=705.73µs min=-725442ns  med=36.67µs max=161.72ms p(75)=42.1µs  p(95)=234.59µs p(99)=25.43ms count=2954063
     http_req_sending...............: avg=130.63µs min=-1969621ns med=18.5µs  max=198ms    p(75)=21.1µs  p(95)=44.55µs  p(99)=229.9µs count=2954063
     http_req_tls_handshaking.......: avg=0s       min=0s         med=0s      max=0s       p(75)=0s      p(95)=0s       p(99)=0s      count=2954063
     http_req_waiting...............: avg=15.67ms  min=123.45µs   med=12.91ms max=196.62ms p(75)=21.41ms p(95)=38.48ms  p(99)=52.87ms count=2954063
     http_reqs......................: 2954063 24614.649678/s
     iteration_duration.............: avg=19.68ms  min=265.58µs   med=15.56ms max=247.25ms p(75)=25.8ms  p(95)=49.84ms  p(99)=83.22ms count=2954063
     iterations.....................: 2954063 24614.649678/s
     vus............................: 500     min=500        max=500  
     vus_max........................: 500     min=500        max=500  
```

1. http1.1 (tls), upstream http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 2.1 GB  18 MB/s
     data_sent......................: 519 MB  4.3 MB/s
     http_req_blocked...............: avg=46.78µs  min=100ns      med=351ns   max=364.63ms p(75)=371ns    p(95)=451ns    p(99)=682ns    count=2355898
     http_req_connecting............: avg=20.62µs  min=0s         med=0s      max=234ms    p(75)=0s       p(95)=0s       p(99)=0s       count=2355898
     http_req_duration..............: avg=22.85ms  min=318.82µs   med=18.62ms max=294.57ms p(75)=29.56ms  p(95)=54.21ms  p(99)=90.83ms  count=2355898
       { expected_response:true }...: avg=22.85ms  min=318.82µs   med=18.62ms max=294.57ms p(75)=29.56ms  p(95)=54.21ms  p(99)=90.83ms  count=2355898
     http_req_failed................: 0.00%   ✓ 0            ✗ 2355898
     http_req_receiving.............: avg=10.72ms  min=-595499ns  med=7.4ms   max=254.39ms p(75)=15.23ms  p(95)=30.97ms  p(99)=58.58ms  count=2355898
     http_req_sending...............: avg=508.98µs min=-1486386ns med=73.9µs  max=236.47ms p(75)=123.59µs p(95)=269.97µs p(99)=8.57ms   count=2355898
     http_req_tls_handshaking.......: avg=20.68µs  min=0s         med=0s      max=276.09ms p(75)=0s       p(95)=0s       p(99)=0s       count=2355898
     http_req_waiting...............: avg=11.62ms  min=0s         med=9.61ms  max=256.37ms p(75)=15.35ms  p(95)=27.03ms  p(99)=48.82ms  count=2355898
     http_reqs......................: 2355898 19631.870906/s
     iteration_duration.............: avg=25.08ms  min=418.92µs   med=20.25ms max=410.88ms p(75)=32.27ms  p(95)=61.12ms  p(99)=100.25ms count=2355898
     iterations.....................: 2355898 19631.870906/s
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


     data_received..................: 1.9 GB  16 MB/s
     data_sent......................: 471 MB  3.9 MB/s
     http_req_blocked...............: avg=36.07µs  min=130ns      med=391ns   max=258.8ms  p(75)=411ns    p(95)=521ns    p(99)=781ns    count=2134506
     http_req_connecting............: avg=14.91µs  min=0s         med=0s      max=130.66ms p(75)=0s       p(95)=0s       p(99)=0s       count=2134506
     http_req_duration..............: avg=25ms     min=274.57µs   med=20.19ms max=321.65ms p(75)=31.74ms  p(95)=60.8ms   p(99)=105.53ms count=2134506
       { expected_response:true }...: avg=25ms     min=274.57µs   med=20.19ms max=321.65ms p(75)=31.74ms  p(95)=60.8ms   p(99)=105.53ms count=2134506
     http_req_failed................: 0.00%   ✓ 0            ✗ 2134506
     http_req_receiving.............: avg=11.04ms  min=-109344ns  med=6.66ms  max=294.21ms p(75)=15.43ms  p(95)=34.26ms  p(99)=64.33ms  count=2134506
     http_req_sending...............: avg=679.41µs min=-2035083ns med=73.97µs max=268.99ms p(75)=125.84µs p(95)=334.13µs p(99)=13.41ms  count=2134506
     http_req_tls_handshaking.......: avg=19.34µs  min=0s         med=0s      max=143.81ms p(75)=0s       p(95)=0s       p(99)=0s       count=2134506
     http_req_waiting...............: avg=13.27ms  min=0s         med=11.03ms max=285.98ms p(75)=17.73ms  p(95)=31.12ms  p(99)=54.95ms  count=2134506
     http_reqs......................: 2134506 17787.535063/s
     iteration_duration.............: avg=27.63ms  min=379.97µs   med=22.08ms max=323.51ms p(75)=35.11ms  p(95)=69.06ms  p(99)=114.26ms count=2134506
     iterations.....................: 2134506 17787.535063/s
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

## test server (raw)

1. http1.1, upstream http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 100 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 100 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 3.9 GB  33 MB/s
     data_sent......................: 1.5 GB  13 MB/s
     http_req_blocked...............: avg=2.74µs  min=571ns      med=2.09µs  max=30.54ms p(75)=2.47µs p(95)=3.68µs  p(99)=5.8µs    count=4615937
     http_req_connecting............: avg=54ns    min=0s         med=0s      max=21.59ms p(75)=0s     p(95)=0s      p(99)=0s       count=4615937
     http_req_duration..............: avg=2.46ms  min=-1287734ns med=1.78ms  max=73.49ms p(75)=3.34ms p(95)=6.75ms  p(99)=10.66ms  count=4615937
       { expected_response:true }...: avg=2.46ms  min=-1287734ns med=1.78ms  max=73.49ms p(75)=3.34ms p(95)=6.75ms  p(99)=10.66ms  count=4615937
     http_req_failed................: 0.00%   ✓ 0            ✗ 4615937
     http_req_receiving.............: avg=62.34µs min=-2235372ns med=26.64µs max=35.92ms p(75)=30.5µs p(95)=43.11µs p(99)=226.59µs count=4615937
     http_req_sending...............: avg=21.84µs min=-2233279ns med=14.54µs max=34.8ms  p(75)=16.6µs p(95)=21.47µs p(99)=71.93µs  count=4615937
     http_req_tls_handshaking.......: avg=0s      min=0s         med=0s      max=0s      p(75)=0s     p(95)=0s      p(99)=0s       count=4615937
     http_req_waiting...............: avg=2.38ms  min=0s         med=1.73ms  max=73.41ms p(75)=3.28ms p(95)=6.65ms  p(99)=10.03ms  count=4615937
     http_reqs......................: 4615937 38468.293826/s
     iteration_duration.............: avg=2.58ms  min=102.69µs   med=1.88ms  max=73.6ms  p(75)=3.46ms p(95)=6.93ms  p(99)=11.24ms  count=4615937
     iterations.....................: 4615937 38468.293826/s
     vus............................: 100     min=100        max=100  
     vus_max........................: 100     min=100        max=100 
```
