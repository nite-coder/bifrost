# Benchmark

因為環境與參數的不同隊壓測結果有很大的影響，數據僅供參考

CPU: Apple M1
Ram: 16GB
OS: Debian 12 (docker)
Date: 2024-09-01
Golang: 1.23

Client: k6

```sh
taskset -c 0,1,2,3 k6 run vus.js
```

## Bifrost

1. http1.1, upstream http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 5.1 GB  42 MB/s
     data_sent......................: 1.9 GB  16 MB/s
     http_req_blocked...............: avg=3.12µs   min=208ns      med=375ns  max=76.87ms  p(75)=542ns   p(95)=1.33µs  p(99)=2.66µs  count=5714532
     http_req_connecting............: avg=2.09µs   min=0s         med=0s     max=49.19ms  p(75)=0s      p(95)=0s      p(99)=0s      count=5714532
     http_req_duration..............: avg=10.17ms  min=11.2µs     med=8.32ms max=288.32ms p(75)=11.91ms p(95)=22.92ms p(99)=38.97ms count=5714532
       { expected_response:true }...: avg=10.17ms  min=11.2µs     med=8.32ms max=288.32ms p(75)=11.91ms p(95)=22.92ms p(99)=38.97ms count=5714532
     http_req_failed................: 0.00%   ✓ 0            ✗ 5714532
     http_req_receiving.............: avg=155.95µs min=-1009014ns med=6.62µs max=161.44ms p(75)=8µs     p(95)=30.87µs p(99)=4.15ms  count=5714532
     http_req_sending...............: avg=14.36µs  min=1.91µs     med=3µs    max=220.61ms p(75)=3.95µs  p(95)=9.58µs  p(99)=53.66µs count=5714532
     http_req_tls_handshaking.......: avg=0s       min=0s         med=0s     max=0s       p(75)=0s      p(95)=0s      p(99)=0s      count=5714532
     http_req_waiting...............: avg=10ms     min=0s         med=8.29ms max=234.05ms p(75)=11.86ms p(95)=22.24ms p(99)=35.28ms count=5714532
     http_reqs......................: 5714532 47618.108772/s
     iteration_duration.............: avg=10.46ms  min=111µs      med=8.53ms max=301.04ms p(75)=12.17ms p(95)=23.68ms p(99)=40.28ms count=5714532
     iterations.....................: 5714532 47618.108772/s
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


     data_received..................: 5.3 GB  44 MB/s
     data_sent......................: 2.0 GB  17 MB/s
     http_req_blocked...............: avg=11.21µs  min=208ns      med=375ns  max=388.52ms p(75)=541ns   p(95)=1.2µs   p(99)=2.33µs  count=5776275
     http_req_connecting............: avg=2.33µs   min=0s         med=0s     max=67.4ms   p(75)=0s      p(95)=0s      p(99)=0s      count=5776275
     http_req_duration..............: avg=10.01ms  min=86.79µs    med=8.48ms max=276.05ms p(75)=11.97ms p(95)=21.97ms p(99)=36.53ms count=5776275
       { expected_response:true }...: avg=10.01ms  min=86.79µs    med=8.48ms max=276.05ms p(75)=11.97ms p(95)=21.97ms p(99)=36.53ms count=5776275
     http_req_failed................: 0.00%   ✓ 0            ✗ 5776275
     http_req_receiving.............: avg=149.48µs min=-1672180ns med=6.7µs  max=174.14ms p(75)=7.87µs  p(95)=29.66µs p(99)=3.9ms   count=5776275
     http_req_sending...............: avg=12.47µs  min=-1450723ns med=3.08µs max=177.9ms  p(75)=3.79µs  p(95)=8.7µs   p(99)=54.04µs count=5776275
     http_req_tls_handshaking.......: avg=7.86µs   min=0s         med=0s     max=386.75ms p(75)=0s      p(95)=0s      p(99)=0s      count=5776275
     http_req_waiting...............: avg=9.85ms   min=76.75µs    med=8.46ms max=231.73ms p(75)=11.92ms p(95)=21.5ms  p(99)=33.03ms count=5776275
     http_reqs......................: 5776275 48133.575947/s
     iteration_duration.............: avg=10.35ms  min=113.87µs   med=8.73ms max=421.86ms p(75)=12.27ms p(95)=22.83ms p(99)=38.03ms count=5776275
     iterations.....................: 5776275 48133.575947/s
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


     data_received..................: 3.5 GB  29 MB/s
     data_sent......................: 888 MB  7.4 MB/s
     http_req_blocked...............: avg=16.48µs  min=0s         med=125ns   max=249.08ms p(75)=125ns   p(95)=208ns   p(99)=333ns   count=4037859
     http_req_connecting............: avg=3.43µs   min=0s         med=0s      max=50.08ms  p(75)=0s      p(95)=0s      p(99)=0s      count=4037859
     http_req_duration..............: avg=14.54ms  min=111.16µs   med=11.96ms max=459.05ms p(75)=17.08ms p(95)=32.46ms p(99)=53.83ms count=4037859
       { expected_response:true }...: avg=14.54ms  min=111.16µs   med=11.96ms max=459.05ms p(75)=17.08ms p(95)=32.46ms p(99)=53.83ms count=4037859
     http_req_failed................: 0.00%   ✓ 0            ✗ 4037859
     http_req_receiving.............: avg=4.84ms   min=-1738972ns med=3.28ms  max=352.73ms p(75)=6.15ms  p(95)=13.97ms p(99)=28.43ms count=4037859
     http_req_sending...............: avg=106.46µs min=-116306ns  med=11.83µs max=206.95ms p(75)=14.33µs p(95)=39.7µs  p(99)=710µs   count=4037859
     http_req_tls_handshaking.......: avg=12.75µs  min=0s         med=0s      max=210.02ms p(75)=0s      p(95)=0s      p(99)=0s      count=4037859
     http_req_waiting...............: avg=9.59ms   min=0s         med=7.73ms  max=342.75ms p(75)=11.04ms p(95)=22.55ms p(99)=40.17ms count=4037859
     http_reqs......................: 4037859 33617.167923/s
     iteration_duration.............: avg=14.83ms  min=137.7µs    med=12.17ms max=459.12ms p(75)=17.32ms p(95)=33.11ms p(99)=55.16ms count=4037859
     iterations.....................: 4037859 33617.167923/s
     vus............................: 500     min=500        max=500  
     vus_max........................: 500     min=500        max=500  
```

1. http2 (tls), upstream http2

```sh
     data_received..................: 3.1 GB  25 MB/s
     data_sent......................: 779 MB  6.3 MB/s
     http_req_blocked...............: avg=17.72µs  min=0s        med=125ns   max=203.73ms p(75)=125ns   p(95)=167ns   p(99)=291ns    count=3542766
     http_req_connecting............: avg=7.85µs   min=0s        med=0s      max=125.88ms p(75)=0s      p(95)=0s      p(99)=0s       count=3542766
     http_req_duration..............: avg=16.76ms  min=13.08µs   med=14.31ms max=3.02s    p(75)=20.12ms p(95)=35.71ms p(99)=55.55ms  count=3542766
       { expected_response:true }...: avg=16.67ms  min=13.08µs   med=14.3ms  max=189.77ms p(75)=20.12ms p(95)=35.7ms  p(99)=55.51ms  count=3542655
     http_req_failed................: 0.00%   ✓ 111          ✗ 3542655
     http_req_receiving.............: avg=4.01ms   min=0s        med=2.44ms  max=3s       p(75)=5.16ms  p(95)=12.29ms p(99)=25.45ms  count=3542766
     http_req_sending...............: avg=113.81µs min=-247972ns med=11.62µs max=96.29ms  p(75)=14.2µs  p(95)=36.58µs p(99)=725.02µs count=3542766
     http_req_tls_handshaking.......: avg=9.2µs    min=0s        med=0s      max=134.7ms  p(75)=0s      p(95)=0s      p(99)=0s       count=3542766
     http_req_waiting...............: avg=12.64ms  min=0s        med=10.78ms max=3.02s    p(75)=15.28ms p(95)=26.85ms p(99)=44.01ms  count=3542766
     http_reqs......................: 3542766 28819.459296/s
     iteration_duration.............: avg=16.91ms  min=193.79µs  med=14.4ms  max=3.02s    p(75)=20.24ms p(95)=36ms    p(99)=56.24ms  count=3542766
     iterations.....................: 3542766 28819.459296/s
     vus............................: 3       min=3          max=500  
     vus_max........................: 500     min=500        max=500  
```


# Openresty

1. http1.1, upstream http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 5.9 GB  49 MB/s
     data_sent......................: 2.1 GB  17 MB/s
     http_req_blocked...............: avg=10.35µs  min=208ns     med=375ns  max=305.08ms p(75)=542ns   p(95)=1.2µs   p(99)=2.58µs  count=6250183
     http_req_connecting............: avg=9.18µs   min=0s        med=0s     max=165.29ms p(75)=0s      p(95)=0s      p(99)=0s      count=6250183
     http_req_duration..............: avg=9.24ms   min=50.08µs   med=7.73ms max=896.04ms p(75)=10.89ms p(95)=20.75ms p(99)=36.04ms count=6250183
       { expected_response:true }...: avg=9.24ms   min=50.08µs   med=7.73ms max=896.04ms p(75)=10.89ms p(95)=20.75ms p(99)=36.04ms count=6250183
     http_req_failed................: 0.00%   ✓ 0            ✗ 6250183
     http_req_receiving.............: avg=138.64µs min=-360889ns med=6.95µs max=412.59ms p(75)=8.33µs  p(95)=39.58µs p(99)=3.48ms  count=6250183
     http_req_sending...............: avg=16.87µs  min=1.95µs    med=3.08µs max=535.16ms p(75)=3.95µs  p(95)=9.37µs  p(99)=55.95µs count=6250183
     http_req_tls_handshaking.......: avg=0s       min=0s        med=0s     max=0s       p(75)=0s      p(95)=0s      p(99)=0s      count=6250183
     http_req_waiting...............: avg=9.09ms   min=40.62µs   med=7.69ms max=868.85ms p(75)=10.83ms p(95)=20.24ms p(99)=32.46ms count=6250183
     http_reqs......................: 6250183 52083.545231/s
     iteration_duration.............: avg=9.56ms   min=80.83µs   med=7.93ms max=1.01s    p(75)=11.14ms p(95)=21.72ms p(99)=38.01ms count=6250183
     iterations.....................: 6250183 52083.545231/s
     vus............................: 500     min=500        max=500  
     vus_max........................: 500     min=500        max=500  
```


1. http1 (tls), upstream http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 4.5 GB  38 MB/s
     data_sent......................: 1.1 GB  9.2 MB/s
     http_req_blocked...............: avg=21.22µs min=0s         med=125ns   max=118.54ms p(75)=125ns   p(95)=167ns   p(99)=333ns    count=4985338
     http_req_connecting............: avg=8.3µs   min=0s         med=0s      max=61.09ms  p(75)=0s      p(95)=0s      p(99)=0s       count=4985338
     http_req_duration..............: avg=11.87ms min=110.45µs   med=10.26ms max=293.37ms p(75)=13.93ms p(95)=25.47ms p(99)=41.55ms  count=4985338
       { expected_response:true }...: avg=11.87ms min=110.45µs   med=10.26ms max=293.37ms p(75)=13.93ms p(95)=25.47ms p(99)=41.55ms  count=4985338
     http_req_failed................: 0.00%   ✓ 0            ✗ 4985338
     http_req_receiving.............: avg=4.58ms  min=2.65µs     med=3.93ms  max=228.56ms p(75)=5.92ms  p(95)=10.22ms p(99)=22.91ms  count=4985338
     http_req_sending...............: avg=83.59µs min=-1326639ns med=13.58µs max=228.14ms p(75)=18.7µs  p(95)=59.62µs p(99)=975.63µs count=4985338
     http_req_tls_handshaking.......: avg=12.55µs min=0s         med=0s      max=104.4ms  p(75)=0s      p(95)=0s      p(99)=0s       count=4985338
     http_req_waiting...............: avg=7.2ms   min=0s         med=6.16ms  max=217.71ms p(75)=8.44ms  p(95)=16.66ms p(99)=29.5ms   count=4985338
     http_reqs......................: 4985338 41543.206731/s
     iteration_duration.............: avg=12.01ms min=131.91µs   med=10.33ms max=293.41ms p(75)=14.03ms p(95)=26.02ms p(99)=42.45ms  count=4985338
     iterations.....................: 4985338 41543.206731/s
     vus............................: 500     min=500        max=500  
     vus_max........................: 500     min=500        max=500  
```
