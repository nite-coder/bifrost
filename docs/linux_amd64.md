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

# Bifrost

## HTTP1.1

```sh
     execution: local
        script: vus.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 3.1 GB  26 MB/s
     data_sent......................: 1.1 GB  9.6 MB/s
     http_req_blocked...............: avg=7.68µs   min=641ns     med=2.36µs  max=148.32ms p(75)=2.91µs  p(95)=5.13µs   p(99)=8.78µs   count=3471728
     http_req_connecting............: avg=1.37µs   min=0s        med=0s      max=143.77ms p(75)=0s      p(95)=0s       p(99)=0s       count=3471728
     http_req_duration..............: avg=16.7ms   min=166.37µs  med=14.86ms max=301.65ms p(75)=19.46ms p(95)=32.42ms  p(99)=57.96ms  count=3471728
       { expected_response:true }...: avg=16.7ms   min=166.37µs  med=14.86ms max=301.65ms p(75)=19.46ms p(95)=32.42ms  p(99)=57.96ms  count=3471728
     http_req_failed................: 0.00%   ✓ 0            ✗ 3471728
     http_req_receiving.............: avg=717.32µs min=-777013ns med=33.1µs  max=254.59ms p(75)=38.22µs p(95)=289.58µs p(99)=24.62ms  count=3471728
     http_req_sending...............: avg=66.41µs  min=-188007ns med=16.59µs max=158.9ms  p(75)=18.72µs p(95)=34µs     p(99)=214.58µs count=3471728
     http_req_tls_handshaking.......: avg=0s       min=0s        med=0s      max=0s       p(75)=0s      p(95)=0s       p(99)=0s       count=3471728
     http_req_waiting...............: avg=15.92ms  min=105.86µs  med=14.75ms max=151.72ms p(75)=19.3ms  p(95)=29.4ms   p(99)=39.71ms  count=3471728
     http_reqs......................: 3471728 28925.976585/s
     iteration_duration.............: avg=17.17ms  min=341.89µs  med=15.11ms max=307.71ms p(75)=19.81ms p(95)=34.03ms  p(99)=60.71ms  count=3471728
     iterations.....................: 3471728 28925.976585/s
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
