raw

```sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 2048 max VUs, 10m30s max duration (incl. graceful stop):
              * default: 100000 iterations shared among 2048 VUs (maxDuration: 10m0s, gracefulStop: 30s)


     data_received..................: 87 MB  28 MB/s
     data_sent......................: 31 MB  9.7 MB/s
     http_req_blocked...............: avg=2.24ms   min=791ns    med=4.27µs  max=314.26ms p(90)=6.36µs  p(95)=8.35µs  
     http_req_connecting............: avg=2.13ms   min=0s       med=0s      max=314.16ms p(90)=0s      p(95)=0s      
     http_req_duration..............: avg=34.13ms  min=79.84µs  med=26.35ms max=265.6ms  p(90)=68.07ms p(95)=89.84ms 
       { expected_response:true }...: avg=34.13ms  min=79.84µs  med=26.35ms max=265.6ms  p(90)=68.07ms p(95)=89.84ms 
     http_req_failed................: 0.00%  ✓ 0            ✗ 100000
     http_req_receiving.............: avg=1.94ms   min=8.21µs   med=29.83µs max=143.8ms  p(90)=590.2µs p(95)=9.53ms  
     http_req_sending...............: avg=712.76µs min=6.74µs   med=16.61µs max=175.81ms p(90)=57.82µs p(95)=256.34µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s      p(95)=0s      
     http_req_waiting...............: avg=31.47ms  min=51.94µs  med=25.74ms max=186.24ms p(90)=62.49ms p(95)=78.78ms 
     http_reqs......................: 100000 31870.491357/s
     iteration_duration.............: avg=49.31ms  min=247.82µs med=37.09ms max=541.18ms p(90)=88.11ms p(95)=118.44ms
     iterations.....................: 100000 31870.491357/s
     vus............................: 2048   min=2048       max=2048
     vus_max........................: 2048   min=2048       max=2048
```

bifrost

``` sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 2048 max VUs, 10m30s max duration (incl. graceful stop):
              * default: 100000 iterations shared among 2048 VUs (maxDuration: 10m0s, gracefulStop: 30s)


     data_received..................: 90 MB  24 MB/s
     data_sent......................: 31 MB  8.0 MB/s
     http_req_blocked...............: avg=2.14ms   min=721ns    med=2.77µs  max=487.13ms p(90)=5.75µs   p(95)=7.6µs   
     http_req_connecting............: avg=2.03ms   min=0s       med=0s      max=394.08ms p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=62.38ms  min=288.86µs med=54.92ms max=337.47ms p(90)=106.13ms p(95)=134.93ms
       { expected_response:true }...: avg=62.38ms  min=288.86µs med=54.92ms max=337.47ms p(90)=106.13ms p(95)=134.93ms
     http_req_failed................: 0.00%  ✓ 0           ✗ 100000
     http_req_receiving.............: avg=4.43ms   min=13.17µs  med=34.6µs  max=231.88ms p(90)=8.53ms   p(95)=32.69ms 
     http_req_sending...............: avg=658.27µs min=6.11µs   med=16.33µs max=300.62ms p(90)=46.73µs  p(95)=212.54µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=57.29ms  min=243.39µs med=54.44ms max=236.57ms p(90)=88.91ms  p(95)=102.31ms
     http_reqs......................: 100000 26391.48058/s
     iteration_duration.............: avg=69.81ms  min=779.44µs med=57.6ms  max=615.05ms p(90)=125.04ms p(95)=154.54ms
     iterations.....................: 100000 26391.48058/s
     vus............................: 2048   min=2048      max=2048
     vus_max........................: 2048   min=2048      max=2048
```

openresty

``` sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 2048 max VUs, 10m30s max duration (incl. graceful stop):
              * default: 100000 iterations shared among 2048 VUs (maxDuration: 10m0s, gracefulStop: 30s)


     data_received..................: 94 MB  18 MB/s
     data_sent......................: 30 MB  5.8 MB/s
     http_req_blocked...............: avg=1.1ms    min=742ns    med=3.31µs  max=729.65ms p(90)=6.43µs   p(95)=8.81µs  
     http_req_connecting............: avg=333.52µs min=0s       med=0s      max=172.89ms p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=85.47ms  min=305.35µs med=76.6ms  max=496.04ms p(90)=138.94ms p(95)=172.16ms
       { expected_response:true }...: avg=85.47ms  min=305.35µs med=76.6ms  max=496.04ms p(90)=138.94ms p(95)=172.16ms
     http_req_failed................: 0.00%  ✓ 0            ✗ 100000
     http_req_receiving.............: avg=5.97ms   min=13.54µs  med=38.97µs max=310.36ms p(90)=8.6ms    p(95)=40.88ms 
     http_req_sending...............: avg=1.47ms   min=6.46µs   med=17.86µs max=485.27ms p(90)=66.64µs  p(95)=229.71µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=78.02ms  min=229.09µs med=75.71ms max=466.31ms p(90)=124.68ms p(95)=142.9ms 
     http_reqs......................: 100000 19121.888236/s
     iteration_duration.............: avg=96ms     min=587.43µs med=80.83ms max=901.19ms p(90)=161.63ms p(95)=199.82ms
     iterations.....................: 100000 19121.888236/s
     vus............................: 2048   min=2048       max=2048
     vus_max........................: 2048   min=2048       max=2048 
```

1. 如果想做動態 reload hertz, 可以查 `engine.transport`, 替換 ondata
