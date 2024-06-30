# Benchmark

这是在 Apple M1 的机器 （16 GB Ram) 上跑, 仅供参考

Client: k6

## Bifrost

### http2

  execution: local
     script: vus.js
     output: -

  scenarios: (100.00%) 1 scenario, 500 max VUs, 40s max duration (incl. graceful stop):
           * contacts: 500 looping VUs for 10s (gracefulStop: 30s)

     data_received..................: 313 MB 31 MB/s
     data_sent......................: 80 MB  7.9 MB/s
     http_req_blocked...............: avg=171.86µs min=41ns     med=84ns    max=306.41ms p(90)=125ns   p(95)=208ns  
     http_req_connecting............: avg=24.58µs  min=0s       med=0s      max=45.83ms  p(90)=0s      p(95)=0s     
     http_req_duration..............: avg=13.34ms  min=131.75µs med=11.65ms max=254.96ms p(90)=23.26ms p(95)=28.48ms
       { expected_response:true }...: avg=13.34ms  min=131.75µs med=11.65ms max=254.96ms p(90)=23.26ms p(95)=28.48ms
     http_req_failed................: 0.00%  ✓ 0            ✗ 360353
     http_req_receiving.............: avg=4.41ms   min=4.95µs   med=2.89ms  max=194.56ms p(90)=10.01ms p(95)=13.66ms
     http_req_sending...............: avg=235.47µs min=7.33µs   med=12.2µs  max=42.87ms  p(90)=39µs    p(95)=70.79µs
     http_req_tls_handshaking.......: avg=146.74µs min=0s       med=0s      max=277.84ms p(90)=0s      p(95)=0s     
     http_req_waiting...............: avg=8.69ms   min=0s       med=7.52ms  max=246.86ms p(90)=15.58ms p(95)=19.21ms
     http_reqs......................: 360353 36017.525783/s
     iteration_duration.............: avg=13.81ms  min=166.25µs med=11.87ms max=335.5ms  p(90)=23.8ms  p(95)=29.37ms
     iterations.....................: 360353 36017.525783/s
     vus............................: 500    min=500        max=500 
     vus_max........................: 500    min=500        max=500

### https1.1

    execution: local
     script: vus.js
     output: -

  scenarios: (100.00%) 1 scenario, 500 max VUs, 40s max duration (incl. graceful stop):
           * contacts: 500 looping VUs for 10s (gracefulStop: 30s)

     data_received..................: 455 MB 46 MB/s
     data_sent......................: 176 MB 18 MB/s
     http_req_blocked...............: avg=199.99µs min=291ns   med=417ns  max=436.29ms p(90)=1.25µs  p(95)=1.66µs 
     http_req_connecting............: avg=16.2µs   min=0s      med=0s     max=67.75ms  p(90)=0s      p(95)=0s     
     http_req_duration..............: avg=9.02ms   min=64.08µs med=7.36ms max=337.79ms p(90)=16.16ms p(95)=21.16ms
       { expected_response:true }...: avg=9.02ms   min=64.08µs med=7.36ms max=337.79ms p(90)=16.16ms p(95)=21.16ms
     http_req_failed................: 0.00%  ✓ 0            ✗ 498937
     http_req_receiving.............: avg=70.17µs  min=3.79µs  med=7.04µs max=67.85ms  p(90)=21.7µs  p(95)=63.08µs
     http_req_sending...............: avg=26.71µs  min=2.16µs  med=3.37µs max=63.23ms  p(90)=9.29µs  p(95)=23.66µs
     http_req_tls_handshaking.......: avg=180.58µs min=0s      med=0s     max=433.75ms p(90)=0s      p(95)=0s     
     http_req_waiting...............: avg=8.92ms   min=56.58µs med=7.31ms max=333.99ms p(90)=15.99ms p(95)=20.83ms
     http_reqs......................: 498937 49878.069644/s
     iteration_duration.............: avg=9.87ms   min=86.16µs med=7.78ms max=459.23ms p(90)=17.56ms p(95)=23.24ms
     iterations.....................: 498937 49878.069644/s
     vus............................: 500    min=500        max=500 
     vus_max........................: 500    min=500        max=500 

### http1.1

```sh
  execution: local
    script: vus.js
    output: -

  scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
          * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


    data_received..................: 5.5 GB  46 MB/s
    data_sent......................: 2.0 GB  17 MB/s
    http_req_blocked...............: avg=2.59µs  min=250ns   med=417ns  max=228.61ms p(75)=666ns   p(95)=1.62µs  p(99)=3.79µs   count=6144653
    http_req_connecting............: avg=402ns   min=0s      med=0s     max=38.47ms  p(75)=0s      p(95)=0s      p(99)=0s       count=6144653
    http_req_duration..............: avg=9.41ms  min=68.58µs med=7.89ms max=339.52ms p(75)=10.93ms p(95)=20.17ms p(99)=34.7ms   count=6144653
      { expected_response:true }...: avg=9.41ms  min=68.58µs med=7.89ms max=339.52ms p(75)=10.93ms p(95)=20.17ms p(99)=34.7ms   count=6144653
    http_req_failed................: 0.00%   ✓ 0            ✗ 6144653
    http_req_receiving.............: avg=39.26µs min=3.66µs  med=7.29µs max=240.51ms p(75)=9.58µs  p(95)=29.54µs p(99)=132.41µs count=6144653
    http_req_sending...............: avg=17.02µs min=2.04µs  med=3.29µs max=245.91ms p(75)=4.62µs  p(95)=11.91µs p(99)=58.91µs  count=6144653
    http_req_tls_handshaking.......: avg=0s      min=0s      med=0s     max=0s       p(75)=0s      p(95)=0s      p(99)=0s       count=6144653
    http_req_waiting...............: avg=9.35ms  min=55.33µs med=7.86ms max=337.02ms p(75)=10.89ms p(95)=19.99ms p(99)=34.28ms  count=6144653
    http_reqs......................: 6144653 51202.737824/s
    iteration_duration.............: avg=9.71ms  min=92.7µs  med=8.04ms max=339.56ms p(75)=11.18ms p(95)=21.18ms p(99)=36.78ms  count=6144653
    iterations.....................: 6144653 51202.737824/s
    vus............................: 500     min=500        max=500  
    vus_max........................: 500     min=500        max=500 
  ```

# Openresty

## http2

  scenarios: (100.00%) 1 scenario, 500 max VUs, 40s max duration (incl. graceful stop):
           * contacts: 500 looping VUs for 10s (gracefulStop: 30s)

     data_received..................: 346 MB 35 MB/s
     data_sent......................: 84 MB  8.4 MB/s
     http_req_blocked...............: avg=137.89µs min=0s       med=84ns     max=277.64ms p(90)=125ns   p(95)=250ns  
     http_req_connecting............: avg=17.84µs  min=0s       med=0s       max=32.93ms  p(90)=0s      p(95)=0s     
     http_req_duration..............: avg=12.77ms  min=156.25µs med=11.16ms  max=130.29ms p(90)=21.08ms p(95)=26.14ms
       { expected_response:true }...: avg=12.77ms  min=156.25µs med=11.16ms  max=130.29ms p(90)=21.08ms p(95)=26.14ms
     http_req_failed................: 0.00%  ✓ 0           ✗ 379453
     http_req_receiving.............: avg=2ms      min=4.58µs   med=661.58µs max=55.16ms  p(90)=5.27ms  p(95)=8.98ms 
     http_req_sending...............: avg=211.28µs min=7.33µs   med=11.37µs  max=36.48ms  p(90)=35µs    p(95)=64.37µs
     http_req_tls_handshaking.......: avg=119.56µs min=0s       med=0s       max=277.43ms p(90)=0s      p(95)=0s     
     http_req_waiting...............: avg=10.56ms  min=0s       med=9.61ms   max=130.09ms p(90)=16.92ms p(95)=19.93ms
     http_reqs......................: 379453 37914.88829/s
     iteration_duration.............: avg=13.09ms  min=191.54µs med=11.29ms  max=296.96ms p(90)=21.44ms p(95)=26.63ms
     iterations.....................: 379453 37914.88829/s
     vus............................: 500    min=500       max=500 
     vus_max........................: 500    min=500       max=500 

## https1.1

  execution: local
     script: vus.js
     output: -

  scenarios: (100.00%) 1 scenario, 500 max VUs, 40s max duration (incl. graceful stop):
           * contacts: 500 looping VUs for 10s (gracefulStop: 30s)

     data_received..................: 432 MB 43 MB/s
     data_sent......................: 158 MB 16 MB/s
     http_req_blocked...............: avg=114.28µs min=291ns    med=417ns  max=211.47ms p(90)=1.29µs  p(95)=1.62µs 
     http_req_connecting............: avg=10.72µs  min=0s       med=0s     max=22.66ms  p(90)=0s      p(95)=0s     
     http_req_duration..............: avg=10.56ms  min=147.45µs med=9.19ms max=166.34ms p(90)=16.97ms p(95)=21.35ms
       { expected_response:true }...: avg=10.56ms  min=147.45µs med=9.19ms max=166.34ms p(90)=16.97ms p(95)=21.35ms
     http_req_failed................: 0.00%  ✓ 0            ✗ 447566
     http_req_receiving.............: avg=58.9µs   min=4.2µs    med=7.2µs  max=51.32ms  p(90)=22.79µs p(95)=58.98µs
     http_req_sending...............: avg=33.71µs  min=2.16µs   med=3.29µs max=43.76ms  p(90)=9.33µs  p(95)=20.04µs
     http_req_tls_handshaking.......: avg=100.09µs min=0s       med=0s     max=205.89ms p(90)=0s      p(95)=0s     
     http_req_waiting...............: avg=10.46ms  min=120.75µs med=9.15ms max=161.98ms p(90)=16.81ms p(95)=21.02ms
     http_reqs......................: 447566 44730.014698/s
     iteration_duration.............: avg=11.11ms  min=226.83µs med=9.42ms max=225.62ms p(90)=17.95ms p(95)=23.12ms
     iterations.....................: 447566 44730.014698/s
     vus............................: 500    min=500        max=500 
     vus_max........................: 500    min=500        max=500 

## http1.1

  execution: local
     script: vus.js
     output: -

  scenarios: (100.00%) 1 scenario, 500 max VUs, 40s max duration (incl. graceful stop):
           * default: 500 looping VUs for 10s (gracefulStop: 30s)

     data_received..................: 432 MB 43 MB/s
     data_sent......................: 152 MB 15 MB/s
     http_req_blocked...............: avg=19.61µs min=291ns    med=416ns  max=46.06ms p(90)=1.25µs  p(95)=1.66µs 
     http_req_connecting............: avg=16.48µs min=0s       med=0s     max=46.02ms p(90)=0s      p(95)=0s     
     http_req_duration..............: avg=10.37ms min=82.83µs  med=8.95ms max=79.5ms  p(90)=16.97ms p(95)=21.99ms
       { expected_response:true }...: avg=10.37ms min=82.83µs  med=8.95ms max=79.5ms  p(90)=16.97ms p(95)=21.99ms
     http_req_failed................: 0.00%  ✓ 0            ✗ 459495
     http_req_receiving.............: avg=64.84µs min=4.29µs   med=7.12µs max=56.12ms p(90)=22.79µs p(95)=61.58µs
     http_req_sending...............: avg=33.78µs min=2.04µs   med=3.08µs max=55.85ms p(90)=9.08µs  p(95)=22.25µs
     http_req_tls_handshaking.......: avg=0s      min=0s       med=0s     max=0s      p(90)=0s      p(95)=0s     
     http_req_waiting...............: avg=10.27ms min=69.29µs  med=8.91ms max=58.99ms p(90)=16.75ms p(95)=21.6ms 
     http_reqs......................: 459495 45921.249674/s
     iteration_duration.............: avg=10.78ms min=105.95µs med=9.16ms max=80.33ms p(90)=17.96ms p(95)=23.35ms
     iterations.....................: 459495 45921.249674/s
     vus............................: 500    min=500        max=500 
     vus_max........................: 500    min=500        max=500 
