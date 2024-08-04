# Load test

CPU: AMD Ryzen7 4750U
Ram: 16GB
OS: Debian 11

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


     data_received..................: 3.2 GB  27 MB/s
     data_sent......................: 1.2 GB  10 MB/s
     http_req_blocked...............: avg=12.28µs  min=0s         med=2.25µs  max=114.01ms p(75)=2.75µs  p(95)=4.59µs   p(99)=7.44µs   count=3628932
     http_req_connecting............: avg=7.6µs    min=0s         med=0s      max=113.41ms p(75)=0s      p(95)=0s       p(99)=0s       count=3628932
     http_req_duration..............: avg=15.27ms  min=198.65µs   med=13.17ms max=181.8ms  p(75)=18.2ms  p(95)=32.61ms  p(99)=58.58ms  count=3628932
       { expected_response:true }...: avg=15.27ms  min=198.65µs   med=13.17ms max=181.8ms  p(75)=18.2ms  p(95)=32.61ms  p(99)=58.58ms  count=3628932
     http_req_failed................: 0.00%   ✓ 0            ✗ 3628932
     http_req_receiving.............: avg=456.04µs min=-2303874ns med=31.56µs max=122.44ms p(75)=34.98µs p(95)=157.35µs p(99)=17.39ms  count=3628932
     http_req_sending...............: avg=46.58µs  min=-2305747ns med=16.26µs max=100.63ms p(75)=18.15µs p(95)=30.64µs  p(99)=189.95µs count=3628932
     http_req_tls_handshaking.......: avg=0s       min=0s         med=0s      max=0s       p(75)=0s      p(95)=0s       p(99)=0s       count=3628932
     http_req_waiting...............: avg=14.76ms  min=173.02µs   med=13.08ms max=181.63ms p(75)=18.07ms p(95)=31.73ms  p(99)=45.71ms  count=3628932
     http_reqs......................: 3628932 30241.520029/s
     iteration_duration.............: avg=16.35ms  min=281.81µs   med=13.83ms max=289.85ms p(75)=19.24ms p(95)=35.83ms  p(99)=64.32ms  count=3628932
     iterations.....................: 3628932 30241.520029/s
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


     data_received..................: 3.1 GB  26 MB/s
     data_sent......................: 1.2 GB  10 MB/s
     http_req_blocked...............: avg=27.94µs  min=651ns      med=2.31µs  max=369.46ms p(75)=2.84µs  p(95)=4.72µs   p(99)=7.48µs   count=3434052
     http_req_connecting............: avg=5.24µs   min=0s         med=0s      max=70.5ms   p(75)=0s      p(95)=0s       p(99)=0s       count=3434052
     http_req_duration..............: avg=16.07ms  min=269.57µs   med=13.76ms max=249.57ms p(75)=19.99ms p(95)=35.14ms  p(99)=61.2ms   count=3434052
       { expected_response:true }...: avg=16.07ms  min=269.57µs   med=13.76ms max=249.57ms p(75)=19.99ms p(95)=35.14ms  p(99)=61.2ms   count=3434033
     http_req_failed................: 0.00%   ✓ 19           ✗ 3434033
     http_req_receiving.............: avg=477.64µs min=-2107076ns med=32.81µs max=142.5ms  p(75)=36.36µs p(95)=166.97µs p(99)=17.59ms  count=3434052
     http_req_sending...............: avg=56.25µs  min=-1711830ns med=16.58µs max=98.7ms   p(75)=18.53µs p(95)=31.23µs  p(99)=193.99µs count=3434052
     http_req_tls_handshaking.......: avg=17.56µs  min=0s         med=0s      max=304.47ms p(75)=0s      p(95)=0s       p(99)=0s       count=3434052
     http_req_waiting...............: avg=15.53ms  min=218.1µs    med=13.66ms max=249.51ms p(75)=19.83ms p(95)=34.08ms  p(99)=48.03ms  count=3434052
     http_reqs......................: 3434052 28616.122293/s
     iteration_duration.............: avg=17.31ms  min=366.98µs   med=14.6ms  max=408.14ms p(75)=21.15ms p(95)=38.68ms  p(99)=68.69ms  count=3434052
     iterations.....................: 3434052 28616.122293/s
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


     data_received..................: 2.1 GB  18 MB/s
     data_sent......................: 531 MB  4.4 MB/s
     http_req_blocked...............: avg=43.73µs  min=100ns      med=341ns   max=414.15ms p(75)=361ns   p(95)=510ns   p(99)=721ns   count=2415818
     http_req_connecting............: avg=1.65µs   min=0s         med=0s      max=110.86ms p(75)=0s      p(95)=0s      p(99)=0s      count=2415818
     http_req_duration..............: avg=23.7ms   min=318.55µs   med=20.09ms max=200.02ms p(75)=29.02ms p(95)=52.75ms p(99)=83.24ms count=2415818
       { expected_response:true }...: avg=23.7ms   min=318.55µs   med=20.09ms max=200.02ms p(75)=29.02ms p(95)=52.75ms p(99)=83.24ms count=2415818
     http_req_failed................: 0.00%   ✓ 0            ✗ 2415818
     http_req_receiving.............: avg=9.03ms   min=-4423435ns med=6.21ms  max=181.23ms p(75)=11.72ms p(95)=27.08ms p(99)=52.5ms  count=2415818
     http_req_sending...............: avg=384.56µs min=-2182847ns med=54.23µs max=156.99ms p(75)=60.58µs p(95)=149.4µs p(99)=5.35ms  count=2415818
     http_req_tls_handshaking.......: avg=41.47µs  min=0s         med=0s      max=410.15ms p(75)=0s      p(95)=0s      p(99)=0s      count=2415818
     http_req_waiting...............: avg=14.28ms  min=0s         med=12.39ms max=172.05ms p(75)=17.55ms p(95)=30.89ms p(99)=51.73ms count=2415818
     http_reqs......................: 2415818 20132.205718/s
     iteration_duration.............: avg=24.68ms  min=519.33µs   med=20.78ms max=451.41ms p(75)=29.94ms p(95)=55.57ms p(99)=87.86ms count=2415818
     iterations.....................: 2415818 20132.205718/s
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


     data_received..................: 1.5 GB  12 MB/s
     data_sent......................: 373 MB  3.1 MB/s
     http_req_blocked...............: avg=72.68µs  min=120ns      med=350ns   max=410.94ms p(75)=371ns   p(95)=541ns    p(99)=821ns   count=1694787
     http_req_connecting............: avg=1.13µs   min=0s         med=0s      max=52.36ms  p(75)=0s      p(95)=0s       p(99)=0s      count=1694787
     http_req_duration..............: avg=35.06ms  min=199.33µs   med=32.94ms max=206.36ms p(75)=40.26ms p(95)=64.12ms  p(99)=93.74ms count=1694787
       { expected_response:true }...: avg=35.06ms  min=199.33µs   med=32.94ms max=206.36ms p(75)=40.26ms p(95)=64.12ms  p(99)=93.74ms count=1694787
     http_req_failed................: 0.00%   ✓ 0            ✗ 1694787
     http_req_receiving.............: avg=3.81ms   min=-2108000ns med=1.8ms   max=136.36ms p(75)=3.72ms  p(95)=13.78ms  p(99)=40.99ms count=1694787
     http_req_sending...............: avg=453.54µs min=-897499ns  med=55.6µs  max=106.31ms p(75)=64.25µs p(95)=146.31µs p(99)=17.07ms count=1694787
     http_req_tls_handshaking.......: avg=70.94µs  min=0s         med=0s      max=395.75ms p(75)=0s      p(95)=0s       p(99)=0s      count=1694787
     http_req_waiting...............: avg=30.78ms  min=0s         med=30.15ms max=205.16ms p(75)=36.4ms  p(95)=50.92ms  p(99)=71.3ms  count=1694787
     http_reqs......................: 1694787 14121.165885/s
     iteration_duration.............: avg=35.35ms  min=714.34µs   med=33.1ms  max=481.73ms p(75)=40.46ms p(95)=64.6ms   p(99)=94.91ms count=1694787
     iterations.....................: 1694787 14121.165885/s
     vus............................: 500     min=500        max=500  
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


     data_received..................: 3.4 GB  28 MB/s
     data_sent......................: 1.2 GB  9.9 MB/s
     http_req_blocked...............: avg=20.84µs  min=641ns      med=2.48µs  max=206.61ms p(75)=3.39µs  p(95)=5.09µs   p(99)=8.08µs   count=3575573
     http_req_connecting............: avg=15.38µs  min=0s         med=0s      max=89.48ms  p(75)=0s      p(95)=0s       p(99)=0s       count=3575573
     http_req_duration..............: avg=13.77ms  min=51.22µs    med=11.09ms max=167.71ms p(75)=18.31ms p(95)=33.12ms  p(99)=55.59ms  count=3575573
       { expected_response:true }...: avg=13.77ms  min=51.22µs    med=11.09ms max=167.71ms p(75)=18.31ms p(95)=33.12ms  p(99)=55.59ms  count=3575573
     http_req_failed................: 0.00%   ✓ 0            ✗ 3575573
     http_req_receiving.............: avg=433.62µs min=-2342898ns med=33.65µs max=145.57ms p(75)=38.44µs p(95)=151.42µs p(99)=15.06ms  count=3575573
     http_req_sending...............: avg=80.19µs  min=5.95µs     med=17.1µs  max=119.02ms p(75)=19.56µs p(95)=32.06µs  p(99)=195.53µs count=3575573
     http_req_tls_handshaking.......: avg=0s       min=0s         med=0s      max=0s       p(75)=0s      p(95)=0s       p(99)=0s       count=3575573
     http_req_waiting...............: avg=13.26ms  min=0s         med=10.95ms max=93.13ms  p(75)=18.05ms p(95)=32.18ms  p(99)=45.27ms  count=3575573
     http_reqs......................: 3575573 29797.146666/s
     iteration_duration.............: avg=16.29ms  min=227.77µs   med=13.08ms max=238.96ms p(75)=21.57ms p(95)=39.59ms  p(99)=65.78ms  count=3575573
     iterations.....................: 3575573 29797.146666/s
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


     data_received..................: 3.1 GB  26 MB/s
     data_sent......................: 1.1 GB  9.5 MB/s
     http_req_blocked...............: avg=42.94µs  min=682ns      med=2.6µs   max=237.29ms p(75)=3.63µs  p(95)=5.49µs   p(99)=8.5µs    count=3240469
     http_req_connecting............: avg=15.25µs  min=0s         med=0s      max=93.83ms  p(75)=0s      p(95)=0s       p(99)=0s       count=3240469
     http_req_duration..............: avg=15.2ms   min=-97013ns   med=12.14ms max=259.27ms p(75)=20.05ms p(95)=36.66ms  p(99)=61.57ms  count=3240469
       { expected_response:true }...: avg=15.2ms   min=-97013ns   med=12.14ms max=259.27ms p(75)=20.05ms p(95)=36.66ms  p(99)=61.57ms  count=3240469
     http_req_failed................: 0.00%   ✓ 0           ✗ 3240469
     http_req_receiving.............: avg=516.44µs min=-2055340ns med=35.24µs max=149.84ms p(75)=40.13µs p(95)=184.38µs p(99)=17.23ms  count=3240469
     http_req_sending...............: avg=96.91µs  min=-2047059ns med=17.69µs max=125.73ms p(75)=20.33µs p(95)=37.44µs  p(99)=214.91µs count=3240469
     http_req_tls_handshaking.......: avg=21.42µs  min=0s         med=0s      max=237.09ms p(75)=0s      p(95)=0s       p(99)=0s       count=3240469
     http_req_waiting...............: avg=14.59ms  min=0s         med=11.97ms max=210.17ms p(75)=19.73ms p(95)=35.45ms  p(99)=49.31ms  count=3240469
     http_reqs......................: 3240469 27004.83286/s
     iteration_duration.............: avg=18.02ms  min=279.94µs   med=14.33ms max=326.02ms p(75)=23.71ms p(95)=44.29ms  p(99)=73.24ms  count=3240469
     iterations.....................: 3240469 27004.83286/s
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


     data_received..................: 2.6 GB  22 MB/s
     data_sent......................: 631 MB  5.3 MB/s
     http_req_blocked...............: avg=23.08µs min=110ns      med=341ns   max=225.94ms p(75)=370ns   p(95)=481ns    p(99)=711ns   count=2860925
     http_req_connecting............: avg=6.9µs   min=0s         med=0s      max=177.55ms p(75)=0s      p(95)=0s       p(99)=0s      count=2860925
     http_req_duration..............: avg=18.71ms min=150.72µs   med=15.33ms max=180.69ms p(75)=24.44ms p(95)=43.26ms  p(99)=72.35ms count=2860925
       { expected_response:true }...: avg=18.71ms min=150.72µs   med=15.33ms max=180.69ms p(75)=24.44ms p(95)=43.26ms  p(99)=72.35ms count=2860925
     http_req_failed................: 0.00%   ✓ 0            ✗ 2860925
     http_req_receiving.............: avg=9.51ms  min=-2260837ns med=7.25ms  max=154.06ms p(75)=13.41ms p(95)=25.36ms  p(99)=48.82ms count=2860925
     http_req_sending...............: avg=315.9µs min=-3002317ns med=57.51µs max=143.17ms p(75)=82.68µs p(95)=165.21µs p(99)=3.6ms   count=2860925
     http_req_tls_handshaking.......: avg=14.42µs min=0s         med=0s      max=217.22ms p(75)=0s      p(95)=0s       p(99)=0s      count=2860925
     http_req_waiting...............: avg=8.88ms  min=0s         med=7.43ms  max=151.07ms p(75)=11.43ms p(95)=20.38ms  p(99)=39.15ms count=2860925
     http_reqs......................: 2860925 23841.840425/s
     iteration_duration.............: avg=20.65ms min=325.43µs   med=16.8ms  max=299.48ms p(75)=26.81ms p(95)=49.59ms  p(99)=80.42ms count=2860925
     iterations.....................: 2860925 23841.840425/s
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

1. http1.1

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
