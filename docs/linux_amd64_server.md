# Load test

因為環境與參數的不同對壓測結果有很大的影響，數據僅供參考

Server
```
CPU: 8 vCPUs
Ram: 16GB
OS: Redhat 9 (5.14.0-427.37.1.el9_4.x86_64)
Date: 2024-10-19
Golang: 1.23
```

client:
```
32 vCPUs
65GB RAM
```

CPU INFO
```
processor	: 1
vendor_id	: AuthenticAMD
cpu family	: 25
model		: 17
model name	: AMD EPYC-Genoa Processor
stepping	: 0
microcode	: 0x1000065
cpu MHz		: 3249.990
cache size	: 1024 KB
physical id	: 0
siblings	: 8
core id		: 0
cpu cores	: 4
apicid		: 1
initial apicid	: 1
fpu		: yes
fpu_exception	: yes
cpuid level	: 13
wp		: yes
flags		: fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ht syscall nx mmxext fxsr_opt pdpe1gb rdtscp lm rep_good nopl cpuid extd_apicid tsc_known_freq pni pclmulqdq ssse3 fma cx16 pcid sse4_1 sse4_2 x2apic movbe popcnt aes xsave avx f16c rdrand hypervisor lahf_lm cmp_legacy cr8_legacy abm sse4a misalignsse 3dnowprefetch osvw topoext perfctr_core ssbd ibrs ibpb stibp vmmcall fsgsbase bmi1 avx2 smep bmi2 erms invpcid avx512f avx512dq avx512ifma clflushopt clwb avx512cd sha_ni avx512bw avx512vl xsaveopt xsaves avx512_bf16 clzero xsaveerptr wbnoinvd arat avx512vbmi umip pku ospke avx512_vbmi2 gfni vaes vpclmulqdq avx512_vnni avx512_bitalg avx512_vpopcntdq la57 rdpid fsrm
bugs		: sysret_ss_attrs spectre_v1 spectre_v2 spec_store_bypass srso
bogomips	: 6499.98
TLB size	: 1024 4K pages
clflush size	: 64
cache_alignment	: 64
address sizes	: 40 bits physical, 57 bits virtual
power management:
```



## Bifrost

### VUS
1. http1.1, upstream http1.1

```sh
     execution: local
        script: vue.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 1m0s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 30s (gracefulStop: 30s)


     data_received..................: 2.4 GB  81 MB/s
     data_sent......................: 911 MB  30 MB/s
     http_req_blocked...............: avg=4.9µs    min=1.04µs   med=2.47µs  max=34.54ms p(75)=2.98µs  p(95)=5.23µs   p(99)=23.26µs  count=2736057
     http_req_connecting............: avg=357ns    min=0s       med=0s      max=9.06ms  p(75)=0s      p(95)=0s       p(99)=0s       count=2736057
     http_req_duration..............: avg=5.22ms   min=386.12µs med=4.89ms  max=81.15ms p(75)=5.64ms  p(95)=8.75ms   p(99)=15.22ms  count=2736057
       { expected_response:true }...: avg=5.22ms   min=386.12µs med=4.89ms  max=81.15ms p(75)=5.64ms  p(95)=8.75ms   p(99)=15.22ms  count=2736057
     http_req_failed................: 0.00%   0 out of 2736057
     http_req_receiving.............: avg=142.66µs min=6.98µs   med=25.86µs max=54.73ms p(75)=36.49µs p(95)=249.11µs p(99)=3.8ms    count=2736057
     http_req_sending...............: avg=27.74µs  min=4.69µs   med=10.55µs max=52.54ms p(75)=12.36µs p(95)=46.42µs  p(99)=183.77µs count=2736057
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s      p(75)=0s      p(95)=0s       p(99)=0s       count=2736057
     http_req_waiting...............: avg=5.05ms   min=345.83µs med=4.83ms  max=74.99ms p(75)=5.58ms  p(95)=8.35ms   p(99)=12.22ms  count=2736057
     http_reqs......................: 2736057 91190.715428/s
     iteration_duration.............: avg=5.4ms    min=511.33µs med=4.99ms  max=81.27ms p(75)=5.75ms  p(95)=9.31ms   p(99)=15.94ms  count=2736057
     iterations.....................: 2736057 91190.715428/s
     vus............................: 500     min=500          max=500
     vus_max........................: 500     min=500          max=500

```


```sh
     execution: local
        script: vue.js
        output: -

     scenarios: (100.00%) 1 scenario, 100 max VUs, 1m0s max duration (incl. graceful stop):
              * contacts: 100 looping VUs for 30s (gracefulStop: 30s)


     data_received..................: 2.2 GB  74 MB/s
     data_sent......................: 831 MB  28 MB/s
     http_req_blocked...............: avg=3.88µs  min=1.09µs   med=2.54µs   max=10.18ms p(75)=3.04µs  p(95)=5.02µs  p(99)=22.62µs  count=2495452
     http_req_connecting............: avg=39ns    min=0s       med=0s       max=4.87ms  p(75)=0s      p(95)=0s      p(99)=0s       count=2495452
     http_req_duration..............: avg=1.09ms  min=403.35µs med=1.02ms   max=27.14ms p(75)=1.16ms  p(95)=1.51ms  p(99)=2.96ms   count=2495452
       { expected_response:true }...: avg=1.09ms  min=403.35µs med=1.02ms   max=27.14ms p(75)=1.16ms  p(95)=1.51ms  p(99)=2.96ms   count=2495452
     http_req_failed................: 0.00%   0 out of 2495452
     http_req_receiving.............: avg=44.69µs min=8.21µs   med=25.83µs  max=12.11ms p(75)=33.26µs p(95)=75.96µs p(99)=373.25µs count=2495452
     http_req_sending...............: avg=19.83µs min=4.82µs   med=10.53µs  max=12.52ms p(75)=12.12µs p(95)=35.38µs p(99)=179.29µs count=2495452
     http_req_tls_handshaking.......: avg=0s      min=0s       med=0s       max=0s      p(75)=0s      p(95)=0s      p(99)=0s       count=2495452
     http_req_waiting...............: avg=1.02ms  min=367.38µs med=975.35µs max=27.11ms p(75)=1.1ms   p(95)=1.4ms   p(99)=2.52ms   count=2495452
     http_reqs......................: 2495452 83178.685447/s
     iteration_duration.............: avg=1.18ms  min=483.73µs med=1.1ms    max=27.3ms  p(75)=1.24ms  p(95)=1.67ms  p(99)=3.33ms   count=2495452
     iterations.....................: 2495452 83178.685447/s
     vus............................: 100     min=100          max=100
     vus_max........................: 100     min=100          max=100
```



1. http1.1 (tls), upstream http1.1

```sh
     execution: local
        script: vue.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 1m0s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 30s (gracefulStop: 30s)


     data_received..................: 2.4 GB  79 MB/s
     data_sent......................: 926 MB  31 MB/s
     http_req_blocked...............: avg=17.11µs  min=1.09µs   med=2.53µs  max=149.88ms p(75)=3.06µs  p(95)=5.16µs   p(99)=23.95µs  count=2607547
     http_req_connecting............: avg=630ns    min=0s       med=0s      max=18.81ms  p(75)=0s      p(95)=0s       p(99)=0s       count=2607547
     http_req_duration..............: avg=5.48ms   min=388.13µs med=4.84ms  max=100.73ms p(75)=7.12ms  p(95)=11.83ms  p(99)=16.99ms  count=2607547
       { expected_response:true }...: avg=5.48ms   min=388.13µs med=4.84ms  max=100.73ms p(75)=7.12ms  p(95)=11.83ms  p(99)=16.99ms  count=2607547
     http_req_failed................: 0.00%   0 out of 2607547
     http_req_receiving.............: avg=122.73µs min=8.1µs    med=26.56µs max=36.99ms  p(75)=36.83µs p(95)=221.99µs p(99)=3.38ms   count=2607547
     http_req_sending...............: avg=24.78µs  min=4.92µs   med=10.78µs max=36.91ms  p(75)=12.59µs p(95)=49.44µs  p(99)=182.57µs count=2607547
     http_req_tls_handshaking.......: avg=12.14µs  min=0s       med=0s      max=149.14ms p(75)=0s      p(95)=0s       p(99)=0s       count=2607547
     http_req_waiting...............: avg=5.34ms   min=366.43µs med=4.77ms  max=96.03ms  p(75)=7.04ms  p(95)=11.26ms  p(99)=15.08ms  count=2607547
     http_reqs......................: 2607547 86909.954969/s
     iteration_duration.............: avg=5.68ms   min=516.65µs med=4.98ms  max=170ms    p(75)=7.27ms  p(95)=12.21ms  p(99)=17.65ms  count=2607547
     iterations.....................: 2607547 86909.954969/s
     vus............................: 500     min=500          max=500
     vus_max........................: 500     min=500          max=500
```


### QPS


1. http1.1, upstream http1.1

PUB2
```sh
     execution: local
        script: gps.js
        output: -

     scenarios: (100.00%) 1 scenario, 100 max VUs, 1m0s max duration (incl. graceful stop):
              * constant_request_rate: 50000.00 iterations/s for 30s (maxVUs: 100, gracefulStop: 30s)

WARN[0000] Insufficient VUs, reached 100 active VUs and cannot initialize more  executor=constant-arrival-rate scenario=constant_request_rate

     data_received..................: 1.3 GB  43 MB/s
     data_sent......................: 485 MB  16 MB/s
     dropped_iterations.............: 43325   1444.123015/s
     http_req_blocked...............: avg=4.06µs   min=832ns    med=3.03µs   max=8.14ms  p(75)=3.62µs   p(95)=5.22µs   p(99)=19.82µs  count=1456678
     http_req_connecting............: avg=75ns     min=0s       med=0s       max=4.75ms  p(75)=0s       p(95)=0s       p(99)=0s       count=1456678
     http_req_duration..............: avg=692.35µs min=362.06µs med=604.52µs max=31.16ms p(75)=683.73µs p(95)=1.11ms   p(99)=2.67ms   count=1456678
       { expected_response:true }...: avg=692.35µs min=362.06µs med=604.52µs max=31.16ms p(75)=683.73µs p(95)=1.11ms   p(99)=2.67ms   count=1456678
     http_req_failed................: 0.00%   0 out of 1456678
     http_req_receiving.............: avg=56.81µs  min=7.13µs   med=23.03µs  max=6.07ms  p(75)=28.6µs   p(95)=63.04µs  p(99)=1.58ms   count=1456678
     http_req_sending...............: avg=30.21µs  min=4.48µs   med=10.37µs  max=9.35ms  p(75)=12.88µs  p(95)=58.1µs   p(99)=295.63µs count=1456678
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s       max=0s      p(75)=0s       p(95)=0s       p(99)=0s       count=1456678
     http_req_waiting...............: avg=605.31µs min=335.27µs med=563.75µs max=31.13ms p(75)=635.82µs p(95)=834.49µs p(99)=1.34ms   count=1456678
     http_reqs......................: 1456678 48554.465689/s
     iteration_duration.............: avg=778.84µs min=419.44µs med=678.84µs max=31.23ms p(75)=762.84µs p(95)=1.32ms   p(99)=2.89ms   count=1456678
     iterations.....................: 1456678 48554.465689/s
     vus............................: 34      min=24           max=87 
     vus_max........................: 100     min=100          max=100

```

PL2
```sh
     execution: local
        script: gps.js
        output: -

     scenarios: (100.00%) 1 scenario, 1000 max VUs, 1m0s max duration (incl. graceful stop):
              * constant_request_rate: 10000.00 iterations/s for 30s (maxVUs: 1000, gracefulStop: 30s)

WARN[0023] Insufficient VUs, reached 1000 active VUs and cannot initialize more  executor=constant-arrival-rate scenario=constant_request_rate

     data_received..................: 266 MB 8.9 MB/s
     data_sent......................: 100 MB 3.3 MB/s
     dropped_iterations.............: 958    31.930987/s
     http_req_blocked...............: avg=6.69µs   min=962ns    med=2.99µs   max=15.68ms  p(75)=3.5µs    p(95)=4.76µs   p(99)=14.51µs  count=299043
     http_req_connecting............: avg=2.81µs   min=0s       med=0s       max=10.24ms  p(75)=0s       p(95)=0s       p(99)=0s       count=299043
     http_req_duration..............: avg=577.02µs min=354.21µs med=502.88µs max=209.62ms p(75)=546.47µs p(95)=654.07µs p(99)=2.23ms   count=299043
       { expected_response:true }...: avg=577.02µs min=354.21µs med=502.88µs max=209.62ms p(75)=546.47µs p(95)=654.07µs p(99)=2.23ms   count=299043
     http_req_failed................: 0.00%  0 out of 299043
     http_req_receiving.............: avg=39.16µs  min=6.77µs   med=24.94µs  max=185.3ms  p(75)=30.52µs  p(95)=49.13µs  p(99)=169.15µs count=299043
     http_req_sending...............: avg=22.61µs  min=5.13µs   med=11.76µs  max=185.16ms p(75)=17.23µs  p(95)=49.35µs  p(99)=131.69µs count=299043
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s       max=0s       p(75)=0s       p(95)=0s       p(99)=0s       count=299043
     http_req_waiting...............: avg=515.24µs min=331.76µs med=457.92µs max=185.52ms p(75)=497.72µs p(95)=586.21µs p(99)=1.92ms   count=299043
     http_reqs......................: 299043 9967.367705/s
     iteration_duration.............: avg=693.87µs min=401.56µs med=573.55µs max=210.98ms p(75)=622.05µs p(95)=739.43µs p(99)=2.68ms   count=299043
     iterations.....................: 299043 9967.367705/s
     vus............................: 8      min=4           max=11  
     vus_max........................: 1000   min=1000        max=1000
```


PL4
```sh
     execution: local
        script: gps.js
        output: -

     scenarios: (100.00%) 1 scenario, 1000 max VUs, 1m0s max duration (incl. graceful stop):
              * constant_request_rate: 20000.00 iterations/s for 30s (maxVUs: 1000, gracefulStop: 30s)

WARN[0005] Insufficient VUs, reached 1000 active VUs and cannot initialize more  executor=constant-arrival-rate scenario=constant_request_rate

     data_received..................: 525 MB 18 MB/s
     data_sent......................: 197 MB 6.6 MB/s
     dropped_iterations.............: 9412   313.713132/s
     http_req_blocked...............: avg=20.61µs  min=891ns    med=2.94µs   max=281.32ms p(75)=3.54µs   p(95)=5.04µs   p(99)=16.55µs  count=590590
     http_req_connecting............: avg=12.26µs  min=0s       med=0s       max=31.66ms  p(75)=0s       p(95)=0s       p(99)=0s       count=590590
     http_req_duration..............: avg=728.44µs min=354.1µs  med=515.38µs max=289.77ms p(75)=565.64µs p(95)=732.6µs  p(99)=7.9ms    count=590590
       { expected_response:true }...: avg=728.44µs min=354.1µs  med=515.38µs max=289.77ms p(75)=565.64µs p(95)=732.6µs  p(99)=7.9ms    count=590590
     http_req_failed................: 0.00%  0 out of 590590
     http_req_receiving.............: avg=40.89µs  min=7.18µs   med=23.33µs  max=282.01ms p(75)=28.76µs  p(95)=47.51µs  p(99)=199µs    count=590590
     http_req_sending...............: avg=36.88µs  min=5.2µs    med=11.54µs  max=281.88ms p(75)=20.95µs  p(95)=57.47µs  p(99)=162.58µs count=590590
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s       max=0s       p(75)=0s       p(95)=0s       p(99)=0s       count=590590
     http_req_waiting...............: avg=650.65µs min=329.65µs med=472.44µs max=285.32ms p(75)=517.87µs p(95)=623.87µs p(99)=7.26ms   count=590590
     http_reqs......................: 590590 19685.065727/s
     iteration_duration.............: avg=832.53µs min=401.14µs med=584.31µs max=293.73ms p(75)=638.59µs p(95)=839.07µs p(99)=8.6ms    count=590590
     iterations.....................: 590590 19685.065727/s
     vus............................: 13     min=10          max=56  
     vus_max........................: 1000   min=1000        max=1000
```


## openresty

nginx version: openresty/1.27.1.1


### VUE

1. HTTP1.1, upstream http1.1

```sh
     execution: local
        script: vue.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 1m0s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 30s (gracefulStop: 30s)


     data_received..................: 3.8 GB  126 MB/s
     data_sent......................: 1.3 GB  45 MB/s
     http_req_blocked...............: avg=7.43µs   min=1.03µs   med=2.54µs  max=172.84ms p(75)=3.11µs  p(95)=5.74µs   p(99)=33.88µs  count=4028764
     http_req_connecting............: avg=2.06µs   min=0s       med=0s      max=172.74ms p(75)=0s      p(95)=0s       p(99)=0s       count=4028764
     http_req_duration..............: avg=2.85ms   min=324.69µs med=2.22ms  max=280.97ms p(75)=3.14ms  p(95)=6.09ms   p(99)=14.87ms  count=4028764
       { expected_response:true }...: avg=2.85ms   min=324.69µs med=2.22ms  max=280.97ms p(75)=3.14ms  p(95)=6.09ms   p(99)=14.87ms  count=4028764
     http_req_failed................: 0.00%   0 out of 4028764
     http_req_receiving.............: avg=128.44µs min=7.11µs   med=25.54µs max=278.05ms p(75)=38.2µs  p(95)=138.73µs p(99)=2.59ms   count=4028764
     http_req_sending...............: avg=37.71µs  min=4.43µs   med=10.76µs max=143.64ms p(75)=12.74µs p(95)=48.88µs  p(99)=173.96µs count=4028764
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(75)=0s      p(95)=0s       p(99)=0s       count=4028764
     http_req_waiting...............: avg=2.69ms   min=305.35µs med=2.17ms  max=280.93ms p(75)=3.07ms  p(95)=5.79ms   p(99)=11.69ms  count=4028764
     http_reqs......................: 4028764 134284.501538/s
     iteration_duration.............: avg=3.29ms   min=430.44µs med=2.5ms   max=282.73ms p(75)=3.5ms   p(95)=7.52ms   p(99)=16.85ms  count=4028764
     iterations.....................: 4028764 134284.501538/s
     vus............................: 500     min=500          max=500
     vus_max........................: 500     min=500          max=500 
```


```sh
     execution: local
        script: vue.js
        output: -

     scenarios: (100.00%) 1 scenario, 100 max VUs, 1m0s max duration (incl. graceful stop):
              * contacts: 100 looping VUs for 30s (gracefulStop: 30s)


     data_received..................: 2.6 GB  86 MB/s
     data_sent......................: 922 MB  31 MB/s
     http_req_blocked...............: avg=5.58µs   min=1.02µs   med=2.52µs   max=447.29ms p(75)=3.03µs   p(95)=5.25µs  p(99)=27.01µs  count=2768599
     http_req_connecting............: avg=1.3µs    min=0s       med=0s       max=31.27ms  p(75)=0s       p(95)=0s      p(99)=0s       count=2768599
     http_req_duration..............: avg=940.64µs min=340.25µs med=739.76µs max=738.61ms p(75)=870.89µs p(95)=1.26ms  p(99)=2.75ms   count=2768599
       { expected_response:true }...: avg=940.64µs min=340.25µs med=739.76µs max=738.61ms p(75)=870.89µs p(95)=1.26ms  p(99)=2.75ms   count=2768599
     http_req_failed................: 0.00%   0 out of 2768599
     http_req_receiving.............: avg=59.29µs  min=8.58µs   med=26.66µs  max=737.83ms p(75)=35.82µs  p(95)=82.14µs p(99)=355.79µs count=2768599
     http_req_sending...............: avg=24.25µs  min=4.32µs   med=10.66µs  max=734.58ms p(75)=12.43µs  p(95)=39.56µs p(99)=170.91µs count=2768599
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s       max=0s       p(75)=0s       p(95)=0s      p(99)=0s       count=2768599
     http_req_waiting...............: avg=857.09µs min=312.14µs med=690.52µs max=738.26ms p(75)=817.68µs p(95)=1.14ms  p(99)=2.27ms   count=2768599
     http_reqs......................: 2768599 91498.924312/s
     iteration_duration.............: avg=1.07ms   min=401.26µs med=825.23µs max=743.85ms p(75)=961.22µs p(95)=1.44ms  p(99)=3.22ms   count=2768599
     iterations.....................: 2768599 91498.924312/s
     vus............................: 100     min=100          max=100
     vus_max........................: 100     min=100          max=100
```

1. http1.1 (tls), upstream http1.1

```sh
     execution: local
        script: vue.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 1m0s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 30s (gracefulStop: 30s)


     data_received..................: 2.0 GB  67 MB/s
     data_sent......................: 489 MB  16 MB/s
     http_req_blocked...............: avg=16.5µs   min=120ns    med=381ns   max=1.45s    p(75)=411ns    p(95)=481ns    p(99)=631ns   count=2217965
     http_req_connecting............: avg=6.01µs   min=0s       med=0s      max=1.45s    p(75)=0s       p(95)=0s       p(99)=0s      count=2217965
     http_req_duration..............: avg=6.45ms   min=457.82µs med=4.03ms  max=763.6ms  p(75)=6.29ms   p(95)=18.73ms  p(99)=30.54ms count=2217965
       { expected_response:true }...: avg=6.45ms   min=457.82µs med=4.03ms  max=763.6ms  p(75)=6.29ms   p(95)=18.73ms  p(99)=30.54ms count=2217965
     http_req_failed................: 0.00%   0 out of 2217965
     http_req_receiving.............: avg=635.55µs min=9.2µs    med=85.88µs max=553.96ms p(75)=350.88µs p(95)=2.81ms   p(99)=8.66ms  count=2217965
     http_req_sending...............: avg=171.56µs min=14.16µs  med=48.08µs max=553.88ms p(75)=66.86µs  p(95)=177.05µs p(99)=2.49ms  count=2217965
     http_req_tls_handshaking.......: avg=9.33µs   min=0s       med=0s      max=531.8ms  p(75)=0s       p(95)=0s       p(99)=0s      count=2217965
     http_req_waiting...............: avg=5.65ms   min=0s       med=3.61ms  max=760.69ms p(75)=5.6ms    p(95)=15.9ms   p(99)=27.96ms count=2217965
     http_reqs......................: 2217965 73900.105212/s
     iteration_duration.............: avg=6.68ms   min=520.66µs med=4.18ms  max=1.46s    p(75)=6.5ms    p(95)=19.3ms   p(99)=31.02ms count=2217965
     iterations.....................: 2217965 73900.105212/s
     vus............................: 500     min=500          max=500
     vus_max........................: 500     min=500          max=500
```


### QPS

1. HTTP1.1, upstream http1.1

PUB2
```sh
     execution: local
        script: gps.js
        output: -

     scenarios: (100.00%) 1 scenario, 100 max VUs, 1m0s max duration (incl. graceful stop):
              * constant_request_rate: 50000.00 iterations/s for 30s (maxVUs: 100, gracefulStop: 30s)

WARN[0000] Insufficient VUs, reached 100 active VUs and cannot initialize more  executor=constant-arrival-rate scenario=constant_request_rate

     data_received..................: 1.4 GB  46 MB/s
     data_sent......................: 485 MB  16 MB/s
     dropped_iterations.............: 44813   1493.732519/s
     http_req_blocked...............: avg=4.95µs   min=852ns    med=3.03µs   max=29.35ms  p(75)=3.62µs   p(95)=5.28µs   p(99)=21.25µs  count=1455187
     http_req_connecting............: avg=571ns    min=0s       med=0s       max=28.23ms  p(75)=0s       p(95)=0s       p(99)=0s       count=1455187
     http_req_duration..............: avg=570.97µs min=323.41µs med=487.28µs max=112.56ms p(75)=541.8µs  p(95)=913.53µs p(99)=2.61ms   count=1455187
       { expected_response:true }...: avg=570.97µs min=323.41µs med=487.28µs max=112.56ms p(75)=541.8µs  p(95)=913.53µs p(99)=2.61ms   count=1455187
     http_req_failed................: 0.00%   0 out of 1455187
     http_req_receiving.............: avg=56.87µs  min=7.84µs   med=23.66µs  max=19.72ms  p(75)=29.54µs  p(95)=64.48µs  p(99)=1.51ms   count=1455187
     http_req_sending...............: avg=36.07µs  min=4.61µs   med=10.36µs  max=110.67ms p(75)=12.74µs  p(95)=55µs     p(99)=430.22µs count=1455187
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s       max=0s       p(75)=0s       p(95)=0s       p(99)=0s       count=1455187
     http_req_waiting...............: avg=478.02µs min=295.62µs med=447.3µs  max=112.38ms p(75)=496.2µs  p(95)=637.72µs p(99)=1.09ms   count=1455187
     http_reqs......................: 1455187 48505.124461/s
     iteration_duration.............: avg=660.49µs min=371.05µs med=560.94µs max=112.73ms p(75)=620.99µs p(95)=1.15ms   p(99)=2.85ms   count=1455187
     iterations.....................: 1455187 48505.124461/s
     vus............................: 26      min=24           max=94 
     vus_max........................: 100     min=100          max=100
```

PL2
```sh
     execution: local
        script: gps.js
        output: -

     scenarios: (100.00%) 1 scenario, 1000 max VUs, 1m0s max duration (incl. graceful stop):
              * constant_request_rate: 10000.00 iterations/s for 30s (maxVUs: 1000, gracefulStop: 30s)


     data_received..................: 282 MB 9.4 MB/s
     data_sent......................: 100 MB 3.3 MB/s
     http_req_blocked...............: avg=5.94µs   min=1.02µs   med=3.07µs   max=6.18ms  p(75)=3.61µs   p(95)=4.95µs   p(99)=19.73µs  count=300001
     http_req_connecting............: avg=2.06µs   min=0s       med=0s       max=6.13ms  p(75)=0s       p(95)=0s       p(99)=0s       count=300001
     http_req_duration..............: avg=496.1µs  min=330.77µs med=472.09µs max=15.18ms p(75)=510.01µs p(95)=588.15µs p(99)=819.15µs count=300001
       { expected_response:true }...: avg=496.1µs  min=330.77µs med=472.09µs max=15.18ms p(75)=510.01µs p(95)=588.15µs p(99)=819.15µs count=300001
     http_req_failed................: 0.00%  0 out of 300001
     http_req_receiving.............: avg=30.74µs  min=7.79µs   med=26.23µs  max=2.64ms  p(75)=32.1µs   p(95)=50.24µs  p(99)=156.18µs count=300001
     http_req_sending...............: avg=19.71µs  min=5.07µs   med=12.36µs  max=4.16ms  p(75)=19.56µs  p(95)=48.45µs  p(99)=126.29µs count=300001
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s       max=0s      p(75)=0s       p(95)=0s       p(99)=0s       count=300001
     http_req_waiting...............: avg=445.63µs min=306.71µs med=426.44µs max=15.08ms p(75)=458.66µs p(95)=522.24µs p(99)=596.59µs count=300001
     http_reqs......................: 300001 9999.519272/s
     iteration_duration.............: avg=572.34µs min=378.41µs med=543.9µs  max=15.3ms  p(75)=587.27µs p(95)=675.57µs p(99)=1ms      count=300001
     iterations.....................: 300001 9999.519272/s
     vus............................: 6      min=4           max=7   
     vus_max........................: 1000   min=1000        max=1000
```



PL4
```sh
     execution: local
        script: gps.js
        output: -

     scenarios: (100.00%) 1 scenario, 1000 max VUs, 1m0s max duration (incl. graceful stop):
              * constant_request_rate: 20000.00 iterations/s for 30s (maxVUs: 1000, gracefulStop: 30s)

WARN[0000] Insufficient VUs, reached 1000 active VUs and cannot initialize more  executor=constant-arrival-rate scenario=constant_request_rate

     data_received..................: 565 MB 19 MB/s
     data_sent......................: 200 MB 6.7 MB/s
     dropped_iterations.............: 8      0.266646/s
     http_req_blocked...............: avg=17.45µs  min=812ns    med=2.96µs   max=72.93ms p(75)=3.58µs   p(95)=5.08µs   p(99)=16.78µs  count=599993
     http_req_connecting............: avg=10.65µs  min=0s       med=0s       max=33.12ms p(75)=0s       p(95)=0s       p(99)=0s       count=599993
     http_req_duration..............: avg=535.13µs min=325.7µs  med=457.73µs max=49.83ms p(75)=496.73µs p(95)=596.64µs p(99)=1.03ms   count=599993
       { expected_response:true }...: avg=535.13µs min=325.7µs  med=457.73µs max=49.83ms p(75)=496.73µs p(95)=596.64µs p(99)=1.03ms   count=599993
     http_req_failed................: 0.00%  0 out of 599993
     http_req_receiving.............: avg=33.18µs  min=7.83µs   med=24.49µs  max=25.75ms p(75)=30.18µs  p(95)=48.15µs  p(99)=195.82µs count=599993
     http_req_sending...............: avg=33.19µs  min=4.93µs   med=11.51µs  max=32.71ms p(75)=20.42µs  p(95)=55.04µs  p(99)=161.32µs count=599993
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s       max=0s      p(75)=0s       p(95)=0s       p(99)=0s       count=599993
     http_req_waiting...............: avg=468.75µs min=300.4µs  med=414.33µs max=28.8ms  p(75)=448.21µs p(95)=519.45µs p(99)=737.59µs count=599993
     http_reqs......................: 599993 19998.217735/s
     iteration_duration.............: avg=625.82µs min=375.59µs med=527.43µs max=76.23ms p(75)=571.63µs p(95)=687.81µs p(99)=1.29ms   count=599993
     iterations.....................: 599993 19998.217735/s
     vus............................: 11     min=9           max=14  
     vus_max........................: 1000   min=1000        max=1000
```


## test server (raw)

1. http1.1, upstream http1.1

```sh
     execution: local
        script: vue.js
        output: -

     scenarios: (100.00%) 1 scenario, 500 max VUs, 2m30s max duration (incl. graceful stop):
              * contacts: 500 looping VUs for 2m0s (gracefulStop: 30s)


     data_received..................: 15 GB    122 MB/s
     data_sent......................: 5.7 GB   48 MB/s
     http_req_blocked...............: avg=5.17µs  min=911ns    med=2.48µs  max=398.1ms  p(75)=3.06µs  p(95)=5.57µs  p(99)=32.95µs  count=17138788
     http_req_connecting............: avg=384ns   min=0s       med=0s      max=96.55ms  p(75)=0s      p(95)=0s      p(99)=0s       count=17138788
     http_req_duration..............: avg=2.2ms   min=199.59µs med=1.68ms  max=457.63ms p(75)=2.7ms   p(95)=4.95ms  p(99)=8.96ms   count=17138788
       { expected_response:true }...: avg=2.2ms   min=199.59µs med=1.68ms  max=457.63ms p(75)=2.7ms   p(95)=4.95ms  p(99)=8.96ms   count=17138788
     http_req_failed................: 0.00%    0 out of 17138788
     http_req_receiving.............: avg=71.06µs min=5.49µs   med=20.06µs max=452.54ms p(75)=25.41µs p(95)=69.88µs p(99)=263.29µs count=17138788
     http_req_sending...............: avg=27.57µs min=4.31µs   med=10.33µs max=452.35ms p(75)=12.12µs p(95)=40.77µs p(99)=122.9µs  count=17138788
     http_req_tls_handshaking.......: avg=0s      min=0s       med=0s      max=0s       p(75)=0s      p(95)=0s      p(99)=0s       count=17138788
     http_req_waiting...............: avg=2.1ms   min=172.63µs med=1.63ms  max=457.59ms p(75)=2.65ms  p(95)=4.85ms  p(99)=8.29ms   count=17138788
     http_reqs......................: 17138788 142814.902175/s
     iteration_duration.............: avg=2.66ms  min=264.15µs med=2.02ms  max=462.65ms p(75)=3.07ms  p(95)=5.76ms  p(99)=11.82ms  count=17138788
     iterations.....................: 17138788 142814.902175/s
     vus............................: 500      min=500           max=500
     vus_max........................: 500      min=500           max=500
```
