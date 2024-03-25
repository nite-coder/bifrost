raw

```sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 1024 max VUs, 10m30s max duration (incl. graceful stop):
              * default: 1000000 iterations shared among 1024 VUs (maxDuration: 10m0s, gracefulStop: 30s)


     data_received..................: 867 MB  35 MB/s
     data_sent......................: 281 MB  11 MB/s
     http_req_blocked...............: avg=31.22µs min=651ns    med=3.34µs  max=291.61ms p(90)=5.63µs  p(95)=6.6µs   
     http_req_connecting............: avg=13.97µs min=0s       med=0s      max=131.24ms p(90)=0s      p(95)=0s      
     http_req_duration..............: avg=18.11ms min=74.08µs  med=13.09ms max=373.28ms p(90)=36.35ms p(95)=53.15ms 
       { expected_response:true }...: avg=18.11ms min=74.08µs  med=13.09ms max=373.28ms p(90)=36.35ms p(95)=53.15ms 
     http_req_failed................: 0.00%   ✓ 0            ✗ 1000000
     http_req_receiving.............: avg=2.05ms  min=7.53µs   med=33µs    max=334.24ms p(90)=338.1µs p(95)=10.56ms 
     http_req_sending...............: avg=322.6µs min=5µs      med=16.95µs max=324.21ms p(90)=30.63µs p(95)=138.36µs
     http_req_tls_handshaking.......: avg=0s      min=0s       med=0s      max=0s       p(90)=0s      p(95)=0s      
     http_req_waiting...............: avg=15.74ms min=38.3µs   med=12.74ms max=206.07ms p(90)=32.75ms p(95)=40.6ms  
     http_reqs......................: 1000000 40386.429144/s
     iteration_duration.............: avg=22.66ms min=142.41µs med=16.48ms max=479.42ms p(90)=46.55ms p(95)=65.22ms 
     iterations.....................: 1000000 40386.429144/s
     vus............................: 1024    min=1024       max=1024 
     vus_max........................: 1024    min=1024       max=1024 
```

Hertz

``` sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 1024 max VUs, 10m30s max duration (incl. graceful stop):
              * default: 1000000 iterations shared among 1024 VUs (maxDuration: 10m0s, gracefulStop: 30s)


     data_received..................: 904 MB  26 MB/s
     data_sent......................: 281 MB  8.2 MB/s
     http_req_blocked...............: avg=74.66µs  min=671ns    med=2.68µs  max=200.43ms p(90)=5.61µs   p(95)=6.76µs  
     http_req_connecting............: avg=58.44µs  min=0s       med=0s      max=194.86ms p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=30.86ms  min=147.62µs med=24.94ms max=359.07ms p(90)=56.69ms  p(95)=80.34ms 
       { expected_response:true }...: avg=30.86ms  min=147.62µs med=24.94ms max=359.07ms p(90)=56.69ms  p(95)=80.34ms 
     http_req_failed................: 0.00%   ✓ 0            ✗ 1000000
     http_req_receiving.............: avg=2.86ms   min=8.84µs   med=36.38µs max=227.46ms p(90)=399.66µs p(95)=21.76ms 
     http_req_sending...............: avg=281.81µs min=6.16µs   med=16.88µs max=219.93ms p(90)=30.02µs  p(95)=114.32µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=27.72ms  min=119.7µs  med=24.56ms max=250.27ms p(90)=50ms     p(95)=59.09ms 
     http_reqs......................: 1000000 29057.004411/s
     iteration_duration.............: avg=34.05ms  min=323.74µs med=26.86ms max=359.17ms p(90)=64.04ms  p(95)=90.29ms 
     iterations.....................: 1000000 29057.004411/s
     vus............................: 1024    min=1024       max=1024 
     vus_max........................: 1024    min=1024       max=1024
```

openresty

``` sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 1024 max VUs, 10m30s max duration (incl. graceful stop):
              * default: 1000000 iterations shared among 1024 VUs (maxDuration: 10m0s, gracefulStop: 30s)


     data_received..................: 941 MB  20 MB/s
     data_sent......................: 276 MB  5.9 MB/s
     http_req_blocked...............: avg=105.31µs min=741ns    med=2.95µs  max=314.58ms p(90)=5.93µs   p(95)=7.17µs  
     http_req_connecting............: avg=85.59µs  min=0s       med=0s      max=310.77ms p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=44.06ms  min=320.5µs  med=38.26ms max=294.55ms p(90)=73.62ms  p(95)=103.57ms
       { expected_response:true }...: avg=44.06ms  min=320.5µs  med=38.26ms max=294.55ms p(90)=73.62ms  p(95)=103.57ms
     http_req_failed................: 0.00%   ✓ 0           ✗ 1000000
     http_req_receiving.............: avg=3.31ms   min=11.73µs  med=40.35µs max=207.11ms p(90)=485.01µs p(95)=26.95ms 
     http_req_sending...............: avg=235.82µs min=6.21µs   med=17.74µs max=180.86ms p(90)=32.96µs  p(95)=140.7µs 
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=40.52ms  min=262.88µs med=38ms    max=247.3ms  p(90)=64.6ms   p(95)=76.48ms 
     http_reqs......................: 1000000 21257.75979/s
     iteration_duration.............: avg=46.88ms  min=465.98µs med=39.29ms max=496.25ms p(90)=79.62ms  p(95)=114.68ms
     iterations.....................: 1000000 21257.75979/s
     vus............................: 1024    min=1024      max=1024 
     vus_max........................: 1024    min=1024      max=1024 
```
