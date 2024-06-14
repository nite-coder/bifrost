## openresty

access_log: off

```sh
     execution: local
        script: create_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 40s max duration (incl. graceful stop):
              * default: 500 looping VUs for 10s (gracefulStop: 30s)


     data_received..................: 239 MB 24 MB/s
     data_sent......................: 83 MB  8.3 MB/s
     http_req_blocked...............: avg=28.69µs  min=731ns    med=2.5µs   max=112.38ms p(90)=4.81µs   p(95)=5.99µs  
     http_req_connecting............: avg=13.43µs  min=0s       med=0s      max=112.27ms p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=18.19ms  min=281.03µs med=14.46ms max=139.24ms p(90)=32.05ms  p(95)=56.61ms 
       { expected_response:true }...: avg=18.19ms  min=281.03µs med=14.46ms max=139.24ms p(90)=32.05ms  p(95)=56.61ms 
     http_req_failed................: 0.00%  ✓ 0            ✗ 253600
     http_req_receiving.............: avg=2.03ms   min=11.54µs  med=35µs    max=90.71ms  p(90)=336.26µs p(95)=20.48ms 
     http_req_sending...............: avg=149.05µs min=6.19µs   med=16.72µs max=122.64ms p(90)=29.91µs  p(95)=107.49µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=16.01ms  min=246.96µs med=14.28ms max=94.05ms  p(90)=29.3ms   p(95)=36.26ms 
     http_reqs......................: 253600 25324.655308/s
     iteration_duration.............: avg=19.46ms  min=376.17µs med=15.06ms max=139.4ms  p(90)=36.27ms  p(95)=60.84ms 
     iterations.....................: 253600 25324.655308/s
     vus............................: 500    min=500        max=500 
     vus_max........................: 500    min=500        max=500
```

access_log: on

```sh
     execution: local
        script: create_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 40s max duration (incl. graceful stop):
              * default: 500 looping VUs for 10s (gracefulStop: 30s)


     data_received..................: 235 MB 23 MB/s
     data_sent......................: 82 MB  8.2 MB/s
     http_req_blocked...............: avg=34.64µs  min=712ns    med=2.52µs  max=176.74ms p(90)=4.85µs   p(95)=5.98µs  
     http_req_connecting............: avg=14.73µs  min=0s       med=0s      max=113.66ms p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=18.37ms  min=265.31µs med=14.35ms max=185.63ms p(90)=32.91ms  p(95)=58.87ms 
       { expected_response:true }...: avg=18.37ms  min=265.31µs med=14.35ms max=185.63ms p(90)=32.91ms  p(95)=58.87ms 
     http_req_failed................: 0.00%  ✓ 0            ✗ 249426
     http_req_receiving.............: avg=2.28ms   min=11.2µs   med=35.24µs max=135.33ms p(90)=331.28µs p(95)=21.34ms 
     http_req_sending...............: avg=150.55µs min=6.56µs   med=16.86µs max=84.42ms  p(90)=30.27µs  p(95)=111.13µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=15.94ms  min=228.96µs med=14.16ms max=122.4ms  p(90)=29.82ms  p(95)=36.72ms 
     http_reqs......................: 249426 24900.971368/s
     iteration_duration.............: avg=19.78ms  min=399.22µs med=15.04ms max=241.86ms p(90)=37.53ms  p(95)=65.33ms 
     iterations.....................: 249426 24900.971368/s
     vus............................: 500    min=500        max=500 
     vus_max........................: 500    min=500        max=500 
```

## Bifrost

access_log: off

```sh
     execution: local
        script: create_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 40s max duration (incl. graceful stop):
              * default: 500 looping VUs for 10s (gracefulStop: 30s)


     data_received..................: 243 MB 24 MB/s
     data_sent......................: 90 MB  9.0 MB/s
     http_req_blocked...............: avg=27.72µs  min=661ns    med=2.33µs  max=77.55ms  p(90)=4.64µs   p(95)=5.93µs  
     http_req_connecting............: avg=12.47µs  min=0s       med=0s      max=72.93ms  p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=17.24ms  min=234.25µs med=14.09ms max=289.63ms p(90)=29.91ms  p(95)=44.17ms 
       { expected_response:true }...: avg=17.24ms  min=234.25µs med=14.09ms max=289.63ms p(90)=29.91ms  p(95)=44.17ms 
     http_req_failed................: 0.00%  ✓ 0            ✗ 273255
     http_req_receiving.............: avg=1.42ms   min=9.91µs   med=32.47µs max=79.81ms  p(90)=354.62µs p(95)=13.1ms  
     http_req_sending...............: avg=127.15µs min=5.71µs   med=16.26µs max=79.46ms  p(90)=30.01µs  p(95)=121.45µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=15.69ms  min=186.65µs med=13.93ms max=289.49ms p(90)=26.45ms  p(95)=31.46ms 
     http_reqs......................: 273255 27314.907118/s
     iteration_duration.............: avg=18.07ms  min=347.16µs med=14.56ms max=337.9ms  p(90)=32.18ms  p(95)=46.67ms 
     iterations.....................: 273255 27314.907118/s
     vus............................: 500    min=500        max=500 
     vus_max........................: 500    min=500        max=500 
```


access_log: on

```sh
     execution: local
        script: create_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 40s max duration (incl. graceful stop):
              * default: 500 looping VUs for 10s (gracefulStop: 30s)


     data_received..................: 223 MB 22 MB/s
     data_sent......................: 83 MB  8.3 MB/s
     http_req_blocked...............: avg=79.24µs  min=621ns    med=2.37µs  max=85.99ms  p(90)=4.78µs   p(95)=6.1µs   
     http_req_connecting............: avg=65.68µs  min=0s       med=0s      max=85.28ms  p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=18.71ms  min=239.42µs med=15.26ms max=155.23ms p(90)=34.37ms  p(95)=49.84ms 
       { expected_response:true }...: avg=18.71ms  min=239.42µs med=15.26ms max=155.23ms p(90)=34.37ms  p(95)=49.84ms 
     http_req_failed................: 0.00%  ✓ 0            ✗ 251121
     http_req_receiving.............: avg=1.64ms   min=10.18µs  med=32.79µs max=99.75ms  p(90)=338.46µs p(95)=14.41ms 
     http_req_sending...............: avg=145.95µs min=5.64µs   med=16.51µs max=83.65ms  p(90)=30.09µs  p(95)=116.62µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=16.92ms  min=190.77µs med=15.07ms max=121.58ms p(90)=30.07ms  p(95)=35.89ms 
     http_reqs......................: 251121 25077.406201/s
     iteration_duration.............: avg=19.73ms  min=364.7µs  med=15.8ms  max=237.3ms  p(90)=37.25ms  p(95)=52.59ms 
     iterations.....................: 251121 25077.406201/s
     vus............................: 500    min=500        max=500 
     vus_max........................: 500    min=500        max=500 
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
