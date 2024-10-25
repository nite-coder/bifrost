# Load test

因為環境與參數的不同對壓測結果有很大的影響，數據僅供參考

CPU: AMD Ryzen7 4750U
Ram: 16GB
OS: Debian 12 (docker)
Date: 2024-10-22
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

     scenarios: (100.00%) 1 scenario, 100 max VUs, 1m0s max duration (incl. graceful stop):
              * contacts: 100 looping VUs for 30s (gracefulStop: 30s)


     data_received..................: 614 MB 21 MB/s
     data_sent......................: 229 MB 7.6 MB/s
     http_req_blocked...............: avg=7.48µs   min=671ns      max=37.67ms p(50)=2.51µs  p(75)=3.14µs  p(95)=5.37µs   p(99)=10.14µs  count=690871
     http_req_connecting............: avg=645ns    min=0s         max=16.64ms p(50)=0s      p(75)=0s      p(95)=0s       p(99)=0s       count=690871
     http_req_duration..............: avg=4.01ms   min=57.92µs    max=60.52ms p(50)=3.11ms  p(75)=4.66ms  p(95)=9.57ms   p(99)=23.16ms  count=690871
       { expected_response:true }...: avg=4.01ms   min=57.92µs    max=60.52ms p(50)=3.11ms  p(75)=4.66ms  p(95)=9.57ms   p(99)=23.16ms  count=690871
     http_req_failed................: 0.00%  ✓ 0            ✗ 690871
     http_req_receiving.............: avg=247.27µs min=-2243361ns max=54.56ms p(50)=32.06µs p(75)=38.53µs p(95)=138.94µs p(99)=9.48ms   count=690871
     http_req_sending...............: avg=57.5µs   min=5.21µs     max=38.05ms p(50)=17.25µs p(75)=20.09µs p(95)=37.16µs  p(99)=197.53µs count=690871
     http_req_tls_handshaking.......: avg=0s       min=0s         max=0s      p(50)=0s      p(75)=0s      p(95)=0s       p(99)=0s       count=690871
     http_req_waiting...............: avg=3.71ms   min=0s         max=45.29ms p(50)=3.03ms  p(75)=4.56ms  p(95)=9.05ms   p(99)=14.95ms  count=690871
     http_reqs......................: 690871 23029.874815/s
     iteration_duration.............: avg=4.3ms    min=286µs      max=60.68ms p(50)=3.27ms  p(75)=4.88ms  p(95)=10.77ms  p(99)=24.45ms  count=690871
     iterations.....................: 690871 23029.874815/s
     vus............................: 100    min=100        max=100 
     vus_max........................: 100    min=100        max=100
```

1. http1.1 (tls), upstream http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 100 max VUs, 1m0s max duration (incl. graceful stop):
              * contacts: 100 looping VUs for 30s (gracefulStop: 30s)


     data_received..................: 403 MB 13 MB/s
     data_sent......................: 102 MB 3.4 MB/s
     http_req_blocked...............: avg=8.01µs   min=110ns      max=70.36ms  p(50)=381ns   p(75)=401ns   p(95)=571ns    p(99)=862ns   count=464701
     http_req_connecting............: avg=512ns    min=0s         max=8.48ms   p(50)=0s      p(75)=0s      p(95)=0s       p(99)=0s      count=464701
     http_req_duration..............: avg=6.19ms   min=364.42µs   max=103.35ms p(50)=4.87ms  p(75)=7.48ms  p(95)=15.15ms  p(99)=26.78ms count=464701
       { expected_response:true }...: avg=6.19ms   min=364.42µs   max=103.35ms p(50)=4.87ms  p(75)=7.48ms  p(95)=15.15ms  p(99)=26.78ms count=464701
     http_req_failed................: 0.00%  ✓ 0            ✗ 464701
     http_req_receiving.............: avg=2.66ms   min=-430936ns  max=73.91ms  p(50)=1.7ms   p(75)=3.22ms  p(95)=7.94ms   p(99)=18.69ms count=464701
     http_req_sending...............: avg=269.31µs min=-2586024ns max=54.11ms  p(50)=64.08µs p(75)=77.23µs p(95)=196.09µs p(99)=8.31ms  count=464701
     http_req_tls_handshaking.......: avg=6.64µs   min=0s         max=69.74ms  p(50)=0s      p(75)=0s      p(95)=0s       p(99)=0s      count=464701
     http_req_waiting...............: avg=3.26ms   min=0s         max=94.74ms  p(50)=2.6ms   p(75)=4.08ms  p(95)=7.48ms   p(99)=12.93ms count=464701
     http_reqs......................: 464701 15489.847174/s
     iteration_duration.............: avg=6.42ms   min=488.42µs   max=104.01ms p(50)=5.04ms  p(75)=7.69ms  p(95)=15.66ms  p(99)=27.57ms count=464701
     iterations.....................: 464701 15489.847174/s
     vus............................: 100    min=100        max=100 
     vus_max........................: 100    min=100        max=100
```

## openresty

version: openresty/1.27.1.1

1. HTTP1.1, upstream http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 100 max VUs, 1m0s max duration (incl. graceful stop):
              * contacts: 100 looping VUs for 30s (gracefulStop: 30s)


     data_received..................: 540 MB 18 MB/s
     data_sent......................: 190 MB 6.3 MB/s
     http_req_blocked...............: avg=12.56µs  min=792ns      max=41.86ms p(50)=2.94µs  p(75)=3.75µs  p(95)=6.45µs   p(99)=14.39µs count=573733
     http_req_connecting............: avg=1.52µs   min=0s         max=41.76ms p(50)=0s      p(75)=0s      p(95)=0s       p(99)=0s      count=573733
     http_req_duration..............: avg=4.74ms   min=-1151782ns max=76.48ms p(50)=3.04ms  p(75)=5.54ms  p(95)=14.36ms  p(99)=32.43ms count=573733
       { expected_response:true }...: avg=4.74ms   min=-1151782ns max=76.48ms p(50)=3.04ms  p(75)=5.54ms  p(95)=14.36ms  p(99)=32.43ms count=573733
     http_req_failed................: 0.00%  ✓ 0            ✗ 573733
     http_req_receiving.............: avg=395.53µs min=-2144842ns max=61.16ms p(50)=38.98µs p(75)=46.76µs p(95)=186.45µs p(99)=16.14ms count=573733
     http_req_sending...............: avg=87.81µs  min=7.22µs     max=52.4ms  p(50)=19.24µs p(75)=22.62µs p(95)=51.21µs  p(99)=281µs   count=573733
     http_req_tls_handshaking.......: avg=0s       min=0s         max=0s      p(50)=0s      p(75)=0s      p(95)=0s       p(99)=0s      count=573733
     http_req_waiting...............: avg=4.26ms   min=0s         max=48.59ms p(50)=2.93ms  p(75)=5.36ms  p(95)=12.93ms  p(99)=21.2ms  count=573733
     http_reqs......................: 573733 19121.793807/s
     iteration_duration.............: avg=5.17ms   min=288.94µs   max=86.46ms p(50)=3.25ms  p(75)=5.87ms  p(95)=16.71ms  p(99)=34.28ms count=573733
     iterations.....................: 573733 19121.793807/s
     vus............................: 100    min=100        max=100 
     vus_max........................: 100    min=100        max=100 
```

1. http1.1 (tls), upstream http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 100 max VUs, 1m0s max duration (incl. graceful stop):
              * contacts: 100 looping VUs for 30s (gracefulStop: 30s)


     data_received..................: 420 MB 14 MB/s
     data_sent......................: 102 MB 3.4 MB/s
     http_req_blocked...............: avg=12.23µs  min=130ns      max=64.31ms  p(50)=391ns   p(75)=421ns    p(95)=631ns    p(99)=952ns   count=461268
     http_req_connecting............: avg=1.52µs   min=0s         max=15.96ms  p(50)=0s      p(75)=0s       p(95)=0s       p(99)=0s      count=461268
     http_req_duration..............: avg=6.19ms   min=167.47µs   max=120.36ms p(50)=4.14ms  p(75)=7.18ms   p(95)=18.54ms  p(99)=32.25ms count=461268
       { expected_response:true }...: avg=6.19ms   min=167.47µs   max=120.36ms p(50)=4.14ms  p(75)=7.18ms   p(95)=18.54ms  p(99)=32.25ms count=461268
     http_req_failed................: 0.00%  ✓ 0           ✗ 461268
     http_req_receiving.............: avg=2.54ms   min=-1011782ns max=101.11ms p(50)=1.41ms  p(75)=3.01ms   p(95)=8.05ms   p(99)=20.41ms count=461268
     http_req_sending...............: avg=400.37µs min=-2566316ns max=80.98ms  p(50)=85.24µs p(75)=135.53µs p(95)=318.02µs p(99)=11.94ms count=461268
     http_req_tls_handshaking.......: avg=8.95µs   min=0s         max=63.57ms  p(50)=0s      p(75)=0s       p(95)=0s       p(99)=0s      count=461268
     http_req_waiting...............: avg=3.25ms   min=0s         max=91.08ms  p(50)=2.13ms  p(75)=3.67ms   p(95)=10.25ms  p(99)=18.94ms count=461268
     http_reqs......................: 461268 15374.25651/s
     iteration_duration.............: avg=6.46ms   min=447.77µs   max=120.47ms p(50)=4.34ms  p(75)=7.47ms   p(95)=19.16ms  p(99)=33.34ms count=461268
     iterations.....................: 461268 15374.25651/s
     vus............................: 100    min=100       max=100 
     vus_max........................: 100    min=100       max=100 
```

## test server (raw)

1. http1.1, upstream http1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 100 max VUs, 1m0s max duration (incl. graceful stop):
              * contacts: 100 looping VUs for 30s (gracefulStop: 30s)


     data_received..................: 757 MB 25 MB/s
     data_sent......................: 294 MB 9.8 MB/s
     http_req_blocked...............: avg=5.26µs   min=772ns    max=25.39ms p(50)=2.46µs  p(75)=3.02µs  p(95)=4.91µs  p(99)=8.94µs   count=887938
     http_req_connecting............: avg=343ns    min=0s       max=14.31ms p(50)=0s      p(75)=0s      p(95)=0s      p(99)=0s       count=887938
     http_req_duration..............: avg=3.13ms   min=49.54µs  max=49.36ms p(50)=2.2ms   p(75)=4.04ms  p(95)=8.6ms   p(99)=17.72ms  count=887938
       { expected_response:true }...: avg=3.13ms   min=49.54µs  max=49.36ms p(50)=2.2ms   p(75)=4.04ms  p(95)=8.6ms   p(99)=17.72ms  count=887938
     http_req_failed................: 0.00%  ✓ 0            ✗ 887938
     http_req_receiving.............: avg=161.23µs min=8.77µs   max=37.85ms p(50)=30.22µs p(75)=35.96µs p(95)=98.87µs p(99)=4.89ms   count=887938
     http_req_sending...............: avg=43.64µs  min=6.19µs   max=33.88ms p(50)=16.91µs p(75)=19.66µs p(95)=31.56µs p(99)=166.59µs count=887938
     http_req_tls_handshaking.......: avg=0s       min=0s       max=0s      p(50)=0s      p(75)=0s      p(95)=0s      p(99)=0s       count=887938
     http_req_waiting...............: avg=2.92ms   min=0s       max=35.01ms p(50)=2.13ms  p(75)=3.94ms  p(95)=8.32ms  p(99)=13.06ms  count=887938
     http_reqs......................: 887938 29596.936994/s
     iteration_duration.............: avg=3.35ms   min=131.37µs max=49.49ms p(50)=2.35ms  p(75)=4.22ms  p(95)=9.14ms  p(99)=19.55ms  count=887938
     iterations.....................: 887938 29596.936994/s
     vus............................: 100    min=100        max=100 
     vus_max........................: 100    min=100        max=100 
```
