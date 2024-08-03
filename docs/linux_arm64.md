# Benchmark

这是在 Apple M1 的机器 （16 GB Ram) 上跑, 仅供参考

Client: k6

## Bifrost

1. http1.1

```sh
  execution: local
     script: vus.js
     output: -

  scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
           * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 5.8 GB  48 MB/s
     data_sent......................: 2.2 GB  18 MB/s
     http_req_blocked...............: avg=1.99µs  min=250ns     med=416ns  max=83.84ms  p(75)=584ns   p(95)=1.5µs   p(99)=3.16µs  count=6514917
     http_req_connecting............: avg=266ns   min=0s        med=0s     max=20.12ms  p(75)=0s      p(95)=0s      p(99)=0s      count=6514917
     http_req_duration..............: avg=8.88ms  min=67.37µs   med=7.51ms max=258.29ms p(75)=10.32ms p(95)=18.1ms  p(99)=31.47ms count=6514917
       { expected_response:true }...: avg=8.88ms  min=67.37µs   med=7.51ms max=258.29ms p(75)=10.32ms p(95)=18.1ms  p(99)=31.47ms count=6514917
     http_req_failed................: 0.00%   ✓ 0            ✗ 6514917
     http_req_receiving.............: avg=32.65µs min=-926014ns med=6.62µs max=170.56ms p(75)=8.25µs  p(95)=27.12µs p(99)=119.2µs count=6514917
     http_req_sending...............: avg=16.38µs min=2µs       med=3.08µs max=147.19ms p(75)=4.16µs  p(95)=11.25µs p(99)=55.7µs  count=6514917
     http_req_tls_handshaking.......: avg=0s      min=0s        med=0s     max=0s       p(75)=0s      p(95)=0s      p(99)=0s      count=6514917
     http_req_waiting...............: avg=8.84ms  min=54.54µs   med=7.48ms max=258.28ms p(75)=10.28ms p(95)=17.96ms p(99)=31.1ms  count=6514917
     http_reqs......................: 6514917 54288.037206/s
     iteration_duration.............: avg=9.16ms  min=102.37µs  med=7.65ms max=277ms    p(75)=10.53ms p(95)=18.95ms p(99)=33.75ms count=6514917
     iterations.....................: 6514917 54288.037206/s
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


     data_received..................: 3.9 GB  32 MB/s
     data_sent......................: 986 MB  8.2 MB/s
     http_req_blocked...............: avg=12.02µs  min=0s         med=84ns    max=293.69ms p(75)=125ns   p(95)=209ns   p(99)=333ns   count=4483250
     http_req_connecting............: avg=2.53µs   min=0s         med=0s      max=75.1ms   p(75)=0s      p(95)=0s      p(99)=0s      count=4483250
     http_req_duration..............: avg=13.13ms  min=99.16µs    med=11.33ms max=392.47ms p(75)=15.79ms p(95)=26.82ms p(99)=42.29ms count=4483250
       { expected_response:true }...: avg=13.13ms  min=99.16µs    med=11.33ms max=392.47ms p(75)=15.79ms p(95)=26.82ms p(99)=42.29ms count=4483250
     http_req_failed................: 0.00%   ✓ 0            ✗ 4483250
     http_req_receiving.............: avg=4ms      min=-2593764ns med=2.61ms  max=323.04ms p(75)=5.29ms  p(95)=11.71ms p(99)=22ms    count=4483250
     http_req_sending...............: avg=135.98µs min=7.08µs     med=11.79µs max=182.63ms p(75)=14.25µs p(95)=49.5µs  p(99)=4.52ms  count=4483250
     http_req_tls_handshaking.......: avg=9µs      min=0s         med=0s      max=263.6ms  p(75)=0s      p(95)=0s      p(99)=0s      count=4483250
     http_req_waiting...............: avg=8.99ms   min=0s         med=7.66ms  max=359.76ms p(75)=10.94ms p(95)=19.14ms p(99)=29.76ms count=4483250
     http_reqs......................: 4483250 37359.250929/s
     iteration_duration.............: avg=13.34ms  min=126.37µs   med=11.48ms max=392.55ms p(75)=15.98ms p(95)=27.26ms p(99)=43.43ms count=4483250
     iterations.....................: 4483250 37359.250929/s
     vus............................: 500     min=500        max=500  
     vus_max........................: 500     min=500        max=500 
```

1. http2 (tls), upstream http2

```sh
     data_received..................: 3.4 GB  28 MB/s
     data_sent......................: 860 MB  7.2 MB/s
     http_req_blocked...............: avg=23.53µs  min=0s        med=84ns    max=430.02ms p(75)=125ns   p(95)=208ns   p(99)=292ns   count=3909909
     http_req_connecting............: avg=534ns    min=0s        med=0s      max=18.91ms  p(75)=0s      p(95)=0s      p(99)=0s      count=3909909
     http_req_duration..............: avg=15.19ms  min=15.5µs    med=13.44ms max=1s       p(75)=18.23ms p(95)=29.22ms p(99)=45.47ms count=3909909
       { expected_response:true }...: avg=15.18ms  min=15.5µs    med=13.44ms max=418.91ms p(75)=18.23ms p(95)=29.22ms p(99)=45.47ms count=3909893
     http_req_failed................: 0.00%   ✓ 16           ✗ 3909893
     http_req_receiving.............: avg=2.56ms   min=-984348ns med=1.49ms  max=996.97ms p(75)=3.17ms  p(95)=7.91ms  p(99)=17.07ms count=3909909
     http_req_sending...............: avg=119.41µs min=-635097ns med=11.54µs max=224ms    p(75)=13.66µs p(95)=46.91µs p(99)=2.71ms  count=3909909
     http_req_tls_handshaking.......: avg=22.75µs  min=0s        med=0s      max=417.83ms p(75)=0s      p(95)=0s      p(99)=0s      count=3909909
     http_req_waiting...............: avg=12.51ms  min=0s        med=11.15ms max=1s       p(75)=15.4ms  p(95)=24.14ms p(99)=35.57ms count=3909909
     http_reqs......................: 3909909 32580.264844/s
     iteration_duration.............: avg=15.32ms  min=153.08µs  med=13.52ms max=1s       p(75)=18.33ms p(95)=29.46ms p(99)=46.3ms  count=3909909
     iterations.....................: 3909909 32580.264844/s
     vus............................: 500     min=500        max=500  
     vus_max........................: 500     min=500        max=500  
```


# Openresty

1. http1.1

```sh
  execution: local
     script: vus.js
     output: -

  scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
           * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 7.4 GB  62 MB/s
     data_sent......................: 2.6 GB  22 MB/s
     http_req_blocked...............: avg=3.71µs  min=250ns     med=416ns  max=69.44ms  p(75)=625ns  p(95)=1.5µs   p(99)=3.29µs   count=7915474
     http_req_connecting............: avg=1.76µs  min=0s        med=0s     max=69.41ms  p(75)=0s     p(95)=0s      p(99)=0s       count=7915474
     http_req_duration..............: avg=6.92ms  min=36.5µs    med=5.51ms max=448.58ms p(75)=8.53ms p(95)=16.35ms p(99)=30.28ms  count=7915474
       { expected_response:true }...: avg=6.92ms  min=36.5µs    med=5.51ms max=448.58ms p(75)=8.53ms p(95)=16.35ms p(99)=30.28ms  count=7915474
     http_req_failed................: 0.00%   ✓ 0           ✗ 7915474
     http_req_receiving.............: avg=42.12µs min=-504847ns med=6.79µs max=399.82ms p(75)=8.25µs p(95)=29.87µs p(99)=137.87µs count=7915474
     http_req_sending...............: avg=20.15µs min=2.04µs    med=3.16µs max=403.55ms p(75)=4.12µs p(95)=12.33µs p(99)=61.54µs  count=7915474
     http_req_tls_handshaking.......: avg=0s      min=0s        med=0s     max=0s       p(75)=0s     p(95)=0s      p(99)=0s       count=7915474
     http_req_waiting...............: avg=6.85ms  min=25.62µs   med=5.48ms max=448.56ms p(75)=8.48ms p(95)=16.18ms p(99)=29.72ms  count=7915474
     http_reqs......................: 7915474 65959.51766/s
     iteration_duration.............: avg=7.44ms  min=53.54µs   med=5.86ms max=464.45ms p(75)=9.04ms p(95)=17.89ms p(99)=33.92ms  count=7915474
     iterations.....................: 7915474 65959.51766/s
     vus............................: 500     min=500       max=500  
     vus_max........................: 500     min=500       max=500  
```


1. http2 (tls), upstream http1.1

```sh
  execution: local
     script: vus.js
     output: -

  scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
           * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 4.6 GB  38 MB/s
     data_sent......................: 1.1 GB  9.2 MB/s
     http_req_blocked...............: avg=18.1µs   min=0s         med=84ns    max=198.24ms p(75)=125ns   p(95)=208ns   p(99)=416ns   count=5031220
     http_req_connecting............: avg=4.57µs   min=0s         med=0s      max=142.26ms p(75)=0s      p(95)=0s      p(99)=0s      count=5031220
     http_req_duration..............: avg=11.46ms  min=58.54µs    med=8.83ms  max=403.06ms p(75)=13.59ms p(95)=27.8ms  p(99)=54.52ms count=5031220
       { expected_response:true }...: avg=11.46ms  min=58.54µs    med=8.83ms  max=403.06ms p(75)=13.59ms p(95)=27.8ms  p(99)=54.52ms count=5031220
     http_req_failed................: 0.00%   ✓ 0            ✗ 5031220
     http_req_receiving.............: avg=3.31ms   min=-1566098ns med=1.71ms  max=296.36ms p(75)=4.11ms  p(95)=11.07ms p(99)=23.12ms count=5031220
     http_req_sending...............: avg=144.29µs min=-61514ns   med=12.29µs max=197.31ms p(75)=15.91µs p(95)=60.45µs p(99)=2.23ms  count=5031220
     http_req_tls_handshaking.......: avg=12.89µs  min=0s         med=0s      max=135.6ms  p(75)=0s      p(95)=0s      p(99)=0s      count=5031220
     http_req_waiting...............: avg=8ms      min=0s         med=6.15ms  max=352.26ms p(75)=9.62ms  p(95)=19.34ms p(99)=39.27ms count=5031220
     http_reqs......................: 5031220 41913.378484/s
     iteration_duration.............: avg=11.86ms  min=77.54µs    med=9.08ms  max=457.79ms p(75)=13.97ms p(95)=28.83ms p(99)=57.26ms count=5031220
     iterations.....................: 5031220 41913.378484/s
     vus............................: 500     min=500        max=500  
     vus_max........................: 500     min=500        max=500  
```
