# Load test

CPU: AMD Ryzen7 4750U
Ram: 16GB
OS: Debian 11 (container)
Date: 2024-08-11

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
     http_req_blocked...............: avg=14.5µs   min=0s         med=2.42µs  max=127.79ms p(75)=3µs     p(95)=4.96µs   p(99)=7.97µs   count=3269372
     http_req_connecting............: avg=8.42µs   min=0s         med=0s      max=116.98ms p(75)=0s      p(95)=0s       p(99)=0s       count=3269372
     http_req_duration..............: avg=16.88ms  min=251.88µs   med=14.21ms max=169.28ms p(75)=20.26ms p(95)=37.48ms  p(99)=66.34ms  count=3269372
       { expected_response:true }...: avg=16.88ms  min=251.88µs   med=14.21ms max=169.28ms p(75)=20.26ms p(95)=37.48ms  p(99)=66.34ms  count=3269372
     http_req_failed................: 0.00%   ✓ 0            ✗ 3269372
     http_req_receiving.............: avg=550.12µs min=-1918006ns med=33.38µs max=140.78ms p(75)=37.22µs p(95)=210.84µs p(99)=19.79ms  count=3269372
     http_req_sending...............: avg=58.1µs   min=-1932243ns med=17.3µs  max=123.74ms p(75)=19.42µs p(95)=32.85µs  p(99)=204.66µs count=3269372
     http_req_tls_handshaking.......: avg=0s       min=0s         med=0s      max=0s       p(75)=0s      p(95)=0s       p(99)=0s       count=3269372
     http_req_waiting...............: avg=16.27ms  min=206.34µs   med=14.11ms max=117.86ms p(75)=20.09ms p(95)=36.05ms  p(99)=51.36ms  count=3269372
     http_reqs......................: 3269372 27245.545656/s
     iteration_duration.............: avg=18.16ms  min=396.29µs   med=15ms    max=232.11ms p(75)=21.48ms p(95)=41.47ms  p(99)=73.41ms  count=3269372
     iterations.....................: 3269372 27245.545656/s
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


     data_received..................: 2.7 GB  23 MB/s
     data_sent......................: 1.0 GB  8.7 MB/s
     http_req_blocked...............: avg=35.25µs  min=691ns      med=2.56µs  max=310.57ms p(75)=3.25µs  p(95)=5.25µs   p(99)=8.63µs   count=2963802
     http_req_connecting............: avg=8.18µs   min=0s         med=0s      max=156.72ms p(75)=0s      p(95)=0s       p(99)=0s       count=2963802
     http_req_duration..............: avg=18.48ms  min=90.98µs    med=15.38ms max=187.12ms p(75)=22.98ms p(95)=41.9ms   p(99)=72.28ms  count=2963802
       { expected_response:true }...: avg=18.48ms  min=90.98µs    med=15.38ms max=187.12ms p(75)=22.98ms p(95)=41.9ms   p(99)=72.28ms  count=2963802
     http_req_failed................: 0.00%   ✓ 0            ✗ 2963802
     http_req_receiving.............: avg=602.47µs min=-2524932ns med=34.67µs max=137.94ms p(75)=38.96µs p(95)=228.67µs p(99)=22.18ms  count=2963802
     http_req_sending...............: avg=67.82µs  min=6.6µs      med=17.86µs max=149.08ms p(75)=20.15µs p(95)=35.74µs  p(99)=214.14µs count=2963802
     http_req_tls_handshaking.......: avg=17.8µs   min=0s         med=0s      max=212.29ms p(75)=0s      p(95)=0s       p(99)=0s       count=2963802
     http_req_waiting...............: avg=17.81ms  min=0s         med=15.25ms max=126.3ms  p(75)=22.77ms p(95)=40.14ms  p(99)=56.41ms  count=2963802
     http_reqs......................: 2963802 24698.728138/s
     iteration_duration.............: avg=20.04ms  min=438.07µs   med=16.44ms max=332.02ms p(75)=24.51ms p(95)=46.55ms  p(99)=80.57ms  count=2963802
     iterations.....................: 2963802 24698.728138/s
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
     data_sent......................: 470 MB  3.9 MB/s
     http_req_blocked...............: avg=29.1µs   min=130ns      med=370ns   max=301.47ms p(75)=391ns   p(95)=531ns    p(99)=751ns    count=2138118
     http_req_connecting............: avg=9.1µs    min=0s         med=0s      max=199.68ms p(75)=0s      p(95)=0s       p(99)=0s       count=2138118
     http_req_duration..............: avg=26.67ms  min=408.9µs    med=22.3ms  max=335.94ms p(75)=32.74ms p(95)=61.02ms  p(99)=95.44ms  count=2138118
       { expected_response:true }...: avg=26.67ms  min=408.9µs    med=22.3ms  max=335.94ms p(75)=32.74ms p(95)=61.02ms  p(99)=95.44ms  count=2138118
     http_req_failed................: 0.00%   ✓ 0            ✗ 2138118
     http_req_receiving.............: avg=10ms     min=-942503ns  med=6.6ms   max=250.68ms p(75)=12.97ms p(95)=31.11ms  p(99)=58.73ms  count=2138118
     http_req_sending...............: avg=458.81µs min=-2537454ns med=57.47µs max=217.03ms p(75)=65.05µs p(95)=188.95µs p(99)=9.35ms   count=2138118
     http_req_tls_handshaking.......: avg=18.51µs  min=0s         med=0s      max=211.31ms p(75)=0s      p(95)=0s       p(99)=0s       count=2138118
     http_req_waiting...............: avg=16.21ms  min=0s         med=13.83ms max=327.75ms p(75)=19.86ms p(95)=36.43ms  p(99)=61.11ms  count=2138118
     http_reqs......................: 2138118 17816.978552/s
     iteration_duration.............: avg=27.86ms  min=529.93µs   med=23.13ms max=336.1ms  p(75)=33.89ms p(95)=64.26ms  p(99)=100.77ms count=2138118
     iterations.....................: 2138118 17816.978552/s
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


     data_received..................: 1.4 GB  12 MB/s
     data_sent......................: 361 MB  3.0 MB/s
     http_req_blocked...............: avg=17.67µs  min=120ns      med=380ns   max=244.25ms p(75)=401ns   p(95)=561ns    p(99)=822ns    count=1642663
     http_req_connecting............: avg=6.62µs   min=0s         med=0s      max=139.54ms p(75)=0s      p(95)=0s       p(99)=0s       count=1642663
     http_req_duration..............: avg=35.99ms  min=93.28µs    med=31.98ms max=3.06s    p(75)=43.07ms p(95)=75.14ms  p(99)=109.98ms count=1642663
       { expected_response:true }...: avg=35.82ms  min=93.28µs    med=31.98ms max=218.12ms p(75)=43.06ms p(95)=75.11ms  p(99)=109.88ms count=1642568
     http_req_failed................: 0.00%   ✓ 95           ✗ 1642568
     http_req_receiving.............: avg=7.15ms   min=-2244013ns med=3.52ms  max=3s       p(75)=8.12ms  p(95)=26.85ms  p(99)=57ms     count=1642663
     http_req_sending...............: avg=647.99µs min=-2762513ns med=60.6µs  max=135.4ms  p(75)=72.29µs p(95)=222.08µs p(99)=24.85ms  count=1642663
     http_req_tls_handshaking.......: avg=9.22µs   min=0s         med=0s      max=118.03ms p(75)=0s      p(95)=0s       p(99)=0s       count=1642663
     http_req_waiting...............: avg=28.19ms  min=0s         med=26.29ms max=3.06s    p(75)=34.61ms p(95)=53.29ms  p(99)=80.68ms  count=1642663
     http_reqs......................: 1642663 13430.901036/s
     iteration_duration.............: avg=36.45ms  min=799.93µs   med=32.3ms  max=3.06s    p(75)=43.45ms p(95)=76.45ms  p(99)=112.29ms count=1642663
     iterations.....................: 1642663 13430.901036/s
     vus............................: 1       min=1          max=500  
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


     data_received..................: 2.6 GB  22 MB/s
     data_sent......................: 927 MB  7.7 MB/s
     http_req_blocked...............: avg=26.74µs  min=0s         med=2.96µs  max=163.86ms p(75)=3.94µs  p(95)=5.85µs   p(99)=10.99µs  count=2801558
     http_req_connecting............: avg=17.72µs  min=0s         med=0s      max=163.77ms p(75)=0s      p(95)=0s       p(99)=0s       count=2801558
     http_req_duration..............: avg=17.22ms  min=69.16µs    med=13.64ms max=318.44ms p(75)=22.55ms p(95)=42.68ms  p(99)=75.39ms  count=2801558
       { expected_response:true }...: avg=17.22ms  min=69.16µs    med=13.64ms max=318.44ms p(75)=22.55ms p(95)=42.68ms  p(99)=75.39ms  count=2801558
     http_req_failed................: 0.00%   ✓ 0            ✗ 2801558
     http_req_receiving.............: avg=753.44µs min=-2372012ns med=38.13µs max=165.28ms p(75)=43.99µs p(95)=267.91µs p(99)=26.98ms  count=2801558
     http_req_sending...............: avg=138.43µs min=-1660663ns med=19.3µs  max=219.32ms p(75)=22.02µs p(95)=56.01µs  p(99)=260.35µs count=2801558
     http_req_tls_handshaking.......: avg=0s       min=0s         med=0s      max=0s       p(75)=0s      p(95)=0s       p(99)=0s       count=2801558
     http_req_waiting...............: avg=16.33ms  min=0s         med=13.44ms max=160.38ms p(75)=22.15ms p(95)=40.36ms  p(99)=56.14ms  count=2801558
     http_reqs......................: 2801558 23346.706851/s
     iteration_duration.............: avg=20.7ms   min=309.3µs    med=16.31ms max=344.71ms p(75)=26.95ms p(95)=52.53ms  p(99)=89.2ms   count=2801558
     iterations.....................: 2801558 23346.706851/s
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


     data_received..................: 2.6 GB  22 MB/s
     data_sent......................: 969 MB  8.1 MB/s
     http_req_blocked...............: avg=42.37µs  min=712ns      med=2.98µs  max=275.26ms p(75)=4.03µs  p(95)=5.86µs   p(99)=9.15µs   count=2741855
     http_req_connecting............: avg=13.68µs  min=0s         med=0s      max=176.79ms p(75)=0s      p(95)=0s       p(99)=0s       count=2741855
     http_req_duration..............: avg=17.76ms  min=241.06µs   med=14.28ms max=213.57ms p(75)=23.5ms  p(95)=43.6ms   p(99)=70.71ms  count=2741855
       { expected_response:true }...: avg=17.76ms  min=241.06µs   med=14.28ms max=213.57ms p(75)=23.5ms  p(95)=43.6ms   p(99)=70.71ms  count=2741855
     http_req_failed................: 0.00%   ✓ 0            ✗ 2741855
     http_req_receiving.............: avg=594.59µs min=-1945695ns med=38.86µs max=187.46ms p(75)=44.4µs  p(95)=248.45µs p(99)=20.19ms  count=2741855
     http_req_sending...............: avg=95.25µs  min=-2501151ns med=19.49µs max=186.1ms  p(75)=22.17µs p(95)=43.53µs  p(99)=239.42µs count=2741855
     http_req_tls_handshaking.......: avg=20.36µs  min=0s         med=0s      max=263.2ms  p(75)=0s      p(95)=0s       p(99)=0s       count=2741855
     http_req_waiting...............: avg=17.07ms  min=187.35µs   med=14.09ms max=131.8ms  p(75)=23.12ms p(95)=41.8ms   p(99)=58.32ms  count=2741855
     http_reqs......................: 2741855 22848.866354/s
     iteration_duration.............: avg=21.21ms  min=349.42µs   med=16.98ms max=348.38ms p(75)=27.9ms  p(95)=52.9ms   p(99)=84.53ms  count=2741855
     iterations.....................: 2741855 22848.866354/s
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

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 4.8 GB  40 MB/s
     data_sent......................: 1.9 GB  16 MB/s
     http_req_blocked...............: avg=9.16µs   min=621ns      med=2.24µs  max=156.65ms p(75)=2.92µs  p(95)=4.69µs   p(99)=7.81µs   count=5613197
     http_req_connecting............: avg=2.87µs   min=0s         med=0s      max=71.44ms  p(75)=0s      p(95)=0s       p(99)=0s       count=5613197
     http_req_duration..............: avg=9.39ms   min=45.25µs    med=7.63ms  max=293.32ms p(75)=11.64ms p(95)=22.01ms  p(99)=47.33ms  count=5613197
       { expected_response:true }...: avg=9.39ms   min=45.25µs    med=7.63ms  max=293.32ms p(75)=11.64ms p(95)=22.01ms  p(99)=47.33ms  count=5613197
     http_req_failed................: 0.00%   ✓ 0            ✗ 5613197
     http_req_receiving.............: avg=478.04µs min=-3300978ns med=30.23µs max=155.5ms  p(75)=33.87µs p(95)=127.13µs p(99)=18.96ms  count=5613197
     http_req_sending...............: avg=82.74µs  min=-3310837ns med=16.08µs max=154.35ms p(75)=18.02µs p(95)=28.71µs  p(99)=206.93µs count=5613197
     http_req_tls_handshaking.......: avg=0s       min=0s         med=0s      max=0s       p(75)=0s      p(95)=0s       p(99)=0s       count=5613197
     http_req_waiting...............: avg=8.83ms   min=0s         med=7.53ms  max=293.22ms p(75)=11.49ms p(95)=21.12ms  p(99)=31.46ms  count=5613197
     http_reqs......................: 5613197 46778.233424/s
     iteration_duration.............: avg=10.42ms  min=109.05µs   med=8.25ms  max=318.76ms p(75)=12.54ms p(95)=25.37ms  p(99)=54.88ms  count=5613197
     iterations.....................: 5613197 46778.233424/s
     vus............................: 500     min=500        max=500  
     vus_max........................: 500     min=500        max=500 
```
