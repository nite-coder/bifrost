Architecture:            x86_64
  CPU op-mode(s):        32-bit, 64-bit
  Address sizes:         40 bits physical, 48 bits virtual
  Byte Order:            Little Endian
CPU(s):                  16
  On-line CPU(s) list:   0-15
Vendor ID:               GenuineIntel
  BIOS Vendor ID:        QEMU
  Model name:            Intel(R) Xeon(R) Platinum 8358 CPU @ 2.60GHz
    BIOS Model name:     pc-i440fx-6.1
    CPU family:          6
    Model:               106
    Thread(s) per core:  1
    Core(s) per socket:  16
    Socket(s):           1
    Stepping:            6
    BogoMIPS:            5187.80
    Flags:               fpu vme de pse tsc msr pae mce cx8 apic sep mtrr pge mca cmov pat pse36 clflush mmx fxsr sse sse2 ht syscall nx pdpe1gb rdtscp lm constant_tsc arch_perfmon rep_good nopl xtopology cpuid tsc_known_freq pni pclmulqdq v
                         mx ssse3 fma cx16 pcid sse4_1 sse4_2 x2apic movbe popcnt tsc_deadline_timer aes xsave avx f16c rdrand hypervisor lahf_lm abm 3dnowprefetch cpuid_fault invpcid_single pti ssbd ibrs ibpb tpr_shadow vnmi flexpriority ep
                         t vpid ept_ad fsgsbase bmi1 avx2 smep bmi2 erms invpcid avx512f avx512dq rdseed adx smap clflushopt clwb avx512cd avx512bw avx512vl xsaveopt xsavec xgetbv1 wbnoinvd arat avx512vbmi umip pku ospke avx512_vbmi2 gfni va
                         es vpclmulqdq avx512_vnni avx512_bitalg avx512_vpopcntdq
Virtualization features:
  Virtualization:        VT-x
  Hypervisor vendor:     KVM
  Virtualization type:   full
Caches (sum of all):
  L1d:                   512 KiB (16 instances)
  L1i:                   512 KiB (16 instances)
  L2:                    64 MiB (16 instances)
NUMA:
  NUMA node(s):          1
  NUMA node0 CPU(s):     0-15
Vulnerabilities:
  Itlb multihit:         KVM: Mitigation: VMX disabled
  L1tf:                  Mitigation; PTE Inversion; VMX conditional cache flushes, SMT disabled
  Mds:                   Vulnerable: Clear CPU buffers attempted, no microcode; SMT Host state unknown
  Meltdown:              Mitigation; PTI
  Mmio stale data:       Vulnerable: Clear CPU buffers attempted, no microcode; SMT Host state unknown
  Retbleed:              Not affected
  Spec store bypass:     Mitigation; Speculative Store Bypass disabled via prctl
  Spectre v1:            Mitigation; usercopy/swapgs barriers and __user pointer sanitization
  Spectre v2:            Mitigation; Retpolines, IBPB conditional, IBRS_FW, STIBP disabled, RSB filling, PBRSB-eIBRS Not affected
  Srbds:                 Not affected
  Tsx async abort:       Not affected

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

``` sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 2048 max VUs, 1m30s max duration (incl. graceful stop):
              * default: 2048 looping VUs for 1m0s (gracefulStop: 30s)


     data_received..................: 2.1 GB  34 MB/s
     data_sent......................: 801 MB  13 MB/s
     http_req_blocked...............: avg=51.72µs  min=621ns    med=1.86µs  max=285.34ms p(90)=2.64µs   p(95)=3.21µs  
     http_req_connecting............: avg=43.33µs  min=0s       med=0s      max=117.24ms p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=40.62ms  min=149.06µs med=33.88ms max=421.89ms p(90)=79.14ms  p(95)=101.68ms
       { expected_response:true }...: avg=40.62ms  min=149.06µs med=33.88ms max=421.89ms p(90)=79.14ms  p(95)=101.68ms
     http_req_failed................: 0.00%   ✓ 0           ✗ 2411119
     http_req_receiving.............: avg=2.67ms   min=6.52µs   med=18.05µs max=288.59ms p(90)=238.08µs p(95)=14.37ms 
     http_req_sending...............: avg=276.18µs min=4.4µs    med=10.01µs max=310.27ms p(90)=21.34µs  p(95)=106.88µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=37.67ms  min=125.72µs med=33.61ms max=289.5ms  p(90)=72.93ms  p(95)=86.86ms 
     http_reqs......................: 2411119 40176.77655/s
     iteration_duration.............: avg=48.19ms  min=205.15µs med=38.52ms max=491.05ms p(90)=94.3ms   p(95)=117.02ms
     iterations.....................: 2411119 40176.77655/s
     vus............................: 2048    min=2048      max=2048 
     vus_max........................: 2048    min=2048      max=2048 


```

bifrost

``` sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 2048 max VUs, 1m30s max duration (incl. graceful stop):
              * default: 2048 looping VUs for 1m0s (gracefulStop: 30s)


     data_received..................: 2.8 GB  47 MB/s
     data_sent......................: 987 MB  16 MB/s
     http_req_blocked...............: avg=50.53µs  min=538ns    med=1.35µs  max=176.16ms p(90)=2.17µs   p(95)=2.52µs 
     http_req_connecting............: avg=42.71µs  min=0s       med=0s      max=150.22ms p(90)=0s       p(95)=0s     
     http_req_duration..............: avg=37.74ms  min=110.53µs med=33.88ms max=234.56ms p(90)=60.89ms  p(95)=79.13ms
       { expected_response:true }...: avg=37.74ms  min=110.53µs med=33.88ms max=234.56ms p(90)=60.89ms  p(95)=79.13ms
     http_req_failed................: 0.00%   ✓ 0           ✗ 2981718
     http_req_receiving.............: avg=1.94ms   min=6.99µs   med=18.21µs max=153.28ms p(90)=230.35µs p(95)=13.43ms
     http_req_sending...............: avg=216.62µs min=4.34µs   med=8.59µs  max=140.01ms p(90)=14.18µs  p(95)=54.66µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s     
     http_req_waiting...............: avg=35.57ms  min=87.55µs  med=33.66ms max=159.98ms p(90)=55.49ms  p(95)=64.29ms
     http_reqs......................: 2981718 49679.42471/s
     iteration_duration.............: avg=40.08ms  min=289.64µs med=35ms    max=242.04ms p(90)=66.69ms  p(95)=86.39ms
     iterations.....................: 2981718 49679.42471/s
     vus............................: 2048    min=2048      max=2048 
     vus_max........................: 2048    min=2048      max=2048 

```

```sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 2048 max VUs, 1m30s max duration (incl. graceful stop):
              * default: 2048 looping VUs for 1m0s (gracefulStop: 30s)


     data_received..................: 2.3 GB  38 MB/s
     data_sent......................: 796 MB  13 MB/s
     http_req_blocked...............: avg=43.9µs   min=592ns    med=1.83µs  max=227.2ms  p(90)=2.62µs   p(95)=3.2µs   
     http_req_connecting............: avg=37.83µs  min=0s       med=0s      max=107ms    p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=40.99ms  min=248.47µs med=34.53ms max=382.27ms p(90)=78.3ms   p(95)=100.47ms
       { expected_response:true }...: avg=40.99ms  min=248.47µs med=34.53ms max=382.27ms p(90)=78.3ms   p(95)=100.47ms
     http_req_failed................: 0.00%   ✓ 2            ✗ 2396340
     http_req_receiving.............: avg=2.25ms   min=7.32µs   med=19.5µs  max=207.84ms p(90)=207.11µs p(95)=11.02ms 
     http_req_sending...............: avg=423.45µs min=4.47µs   med=10.01µs max=265.66ms p(90)=21.1µs   p(95)=109.73µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=38.31ms  min=218.38µs med=34.23ms max=271.86ms p(90)=72.25ms  p(95)=86.96ms 
     http_reqs......................: 2396342 39930.188462/s
     iteration_duration.............: avg=48.44ms  min=347.82µs med=39.12ms max=382.38ms p(90)=93.61ms  p(95)=114.06ms
     iterations.....................: 2396342 39930.188462/s
     vus............................: 2048    min=2048       max=2048 
     vus_max........................: 2048    min=2048       max=2048 
```

```sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 4096 max VUs, 1m30s max duration (incl. graceful stop):
              * default: 4096 looping VUs for 1m0s (gracefulStop: 30s)


     data_received..................: 2.1 GB  36 MB/s
     data_sent......................: 748 MB  13 MB/s
     http_req_blocked...............: avg=172.12µs min=606ns    med=1.95µs  max=447.73ms p(90)=2.83µs   p(95)=3.49µs  
     http_req_connecting............: avg=161.73µs min=0s       med=0s      max=197.5ms  p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=88.87ms  min=257.34µs med=75.14ms max=527.66ms p(90)=178.9ms  p(95)=203.9ms 
       { expected_response:true }...: avg=88.87ms  min=257.34µs med=75.14ms max=527.66ms p(90)=178.9ms  p(95)=203.9ms 
     http_req_failed................: 0.00%   ✓ 2           ✗ 2253565
     http_req_receiving.............: avg=5.06ms   min=9.07µs   med=19.65µs max=488.35ms p(90)=305.2µs  p(95)=24.27ms 
     http_req_sending...............: avg=258.55µs min=4.84µs   med=10.11µs max=345.63ms p(90)=24.27µs  p(95)=123.58µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=83.55ms  min=219.55µs med=74.73ms max=480.15ms p(90)=163.86ms p(95)=180.88ms
     http_reqs......................: 2253567 37541.51416/s
     iteration_duration.............: avg=103.15ms min=358.36µs med=82.83ms max=667.58ms p(90)=194.4ms  p(95)=226.49ms
     iterations.....................: 2253567 37541.51416/s
     vus............................: 4096    min=4096      max=4096 
     vus_max........................: 4096    min=4096      max=4096 
```

```sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 8000 max VUs, 1m30s max duration (incl. graceful stop):
              * default: 8000 looping VUs for 1m0s (gracefulStop: 30s)


     data_received..................: 2.1 GB  34 MB/s
     data_sent......................: 722 MB  12 MB/s
     http_req_blocked...............: avg=768.64µs min=590ns    med=2.04µs   max=465.58ms p(90)=2.96µs   p(95)=3.69µs  
     http_req_connecting............: avg=746.16µs min=0s       med=0s       max=380.72ms p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=191ms    min=265.87µs med=158.22ms max=833.56ms p(90)=366.27ms p(95)=396.63ms
       { expected_response:true }...: avg=191ms    min=265.87µs med=158.22ms max=833.56ms p(90)=366.27ms p(95)=396.63ms
     http_req_failed................: 0.00%   ✓ 0            ✗ 2173054
     http_req_receiving.............: avg=6.29ms   min=9.43µs   med=19.86µs  max=512.37ms p(90)=316.28µs p(95)=30.39ms 
     http_req_sending...............: avg=393.93µs min=4.97µs   med=10.2µs   max=465.79ms p(90)=27.71µs  p(95)=135.88µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s       max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=184.31ms min=222.28µs med=157.92ms max=589.73ms p(90)=343.73ms p(95)=370.69ms
     http_reqs......................: 2173054 36146.659244/s
     iteration_duration.............: avg=210.79ms min=562.16µs med=167.03ms max=902.75ms p(90)=384.85ms p(95)=424.46ms
     iterations.....................: 2173054 36146.659244/s
     vus............................: 8000    min=8000       max=8000 
     vus_max........................: 8000    min=8000       max=8000 
```

openresty

``` sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 2048 max VUs, 1m30s max duration (incl. graceful stop):
              * default: 2048 looping VUs for 1m0s (gracefulStop: 30s)


     data_received..................: 2.4 GB  40 MB/s
     data_sent......................: 849 MB  14 MB/s
     http_req_blocked...............: avg=39.77µs  min=562ns    med=1.36µs  max=132.17ms p(90)=2.15µs   p(95)=2.52µs 
     http_req_connecting............: avg=31.71µs  min=0s       med=0s      max=81.76ms  p(90)=0s       p(95)=0s     
     http_req_duration..............: avg=45.58ms  min=149.67µs med=43.51ms max=223.83ms p(90)=66.37ms  p(95)=83.7ms 
       { expected_response:true }...: avg=45.58ms  min=149.67µs med=43.51ms max=223.83ms p(90)=66.37ms  p(95)=83.7ms 
     http_req_failed................: 0.00%   ✓ 0           ✗ 2579744
     http_req_receiving.............: avg=2.05ms   min=7.26µs   med=19.19µs max=146.64ms p(90)=250.14µs p(95)=15.57ms
     http_req_sending...............: avg=179.64µs min=4.61µs   med=9.12µs  max=129.64ms p(90)=13.97µs  p(95)=62.71µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s     
     http_req_waiting...............: avg=43.35ms  min=122.81µs med=43.42ms max=130.36ms p(90)=59.31ms  p(95)=68.11ms
     http_reqs......................: 2579744 42970.51902/s
     iteration_duration.............: avg=46.98ms  min=332.03µs med=43.75ms max=223.9ms  p(90)=69.89ms  p(95)=90.29ms
     iterations.....................: 2579744 42970.51902/s
     vus............................: 2048    min=2048      max=2048 
     vus_max........................: 2048    min=2048      max=2048 
```

```sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 2048 max VUs, 1m30s max duration (incl. graceful stop):
              * default: 2048 looping VUs for 1m0s (gracefulStop: 30s)


     data_received..................: 2.2 GB  37 MB/s
     data_sent......................: 778 MB  13 MB/s
     http_req_blocked...............: avg=84.37µs  min=626ns    med=1.83µs  max=193.79ms p(90)=2.61µs   p(95)=3.19µs  
     http_req_connecting............: avg=76.42µs  min=0s       med=0s      max=193.75ms p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=43.6ms   min=267.67µs med=35.69ms max=393.68ms p(90)=87.23ms  p(95)=108.31ms
       { expected_response:true }...: avg=43.6ms   min=267.67µs med=35.69ms max=393.68ms p(90)=87.23ms  p(95)=108.31ms
     http_req_failed................: 0.00%   ✓ 0           ✗ 2358537
     http_req_receiving.............: avg=2.66ms   min=8.4µs    med=19.72µs max=285.56ms p(90)=271.33µs p(95)=17.15ms 
     http_req_sending...............: avg=316.72µs min=4.77µs   med=9.99µs  max=184.35ms p(90)=21.06µs  p(95)=108.18µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=40.62ms  min=248.02µs med=35.42ms max=300.65ms p(90)=80.7ms   p(95)=91.5ms  
     http_reqs......................: 2358537 39296.63817/s
     iteration_duration.............: avg=49.94ms  min=392.51µs med=39.66ms max=393.8ms  p(90)=98.09ms  p(95)=122.01ms
     iterations.....................: 2358537 39296.63817/s
     vus............................: 2048    min=2048      max=2048 
     vus_max........................: 2048    min=2048      max=2048 
```

```sh
     execution: local
        script: place_order.js
        output: -

     scenarios: (100.00%) 1 scenario, 4096 max VUs, 1m30s max duration (incl. graceful stop):
              * default: 4096 looping VUs for 1m0s (gracefulStop: 30s)


     data_received..................: 2.1 GB  35 MB/s
     data_sent......................: 740 MB  12 MB/s
     http_req_blocked...............: avg=338.23µs min=619ns    med=1.94µs  max=308.26ms p(90)=2.77µs   p(95)=3.42µs  
     http_req_connecting............: avg=326.78µs min=0s       med=0s      max=308.24ms p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=93.23ms  min=280.88µs med=77.49ms max=591.14ms p(90)=185.63ms p(95)=211.08ms
       { expected_response:true }...: avg=93.23ms  min=280.88µs med=77.49ms max=591.14ms p(90)=185.63ms p(95)=211.08ms
     http_req_failed................: 0.00%   ✓ 0           ✗ 2241373
     http_req_receiving.............: avg=4.67ms   min=8.82µs   med=19.75µs max=293.68ms p(90)=329.14µs p(95)=26.89ms 
     http_req_sending...............: avg=411.67µs min=5.01µs   med=10.07µs max=292.43ms p(90)=23.9µs   p(95)=126.5µs 
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s      max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=88.15ms  min=254.65µs med=77.19ms max=404.7ms  p(90)=169.33ms p(95)=186.27ms
     http_reqs......................: 2241373 37342.57571/s
     iteration_duration.............: avg=104.63ms min=482.98µs med=83.5ms  max=611.22ms p(90)=199.01ms p(95)=231.19ms
     iterations.....................: 2241373 37342.57571/s
     vus............................: 4096    min=4096      max=4096 
     vus_max........................: 4096    min=4096      max=4096 
```

```sh
     data_received..................: 2.0 GB  33 MB/s
     data_sent......................: 694 MB  12 MB/s
     http_req_blocked...............: avg=1.23ms   min=584ns    med=2.05µs   max=543.01ms p(90)=2.99µs   p(95)=3.83µs  
     http_req_connecting............: avg=1.19ms   min=0s       med=0s       max=542.98ms p(90)=0s       p(95)=0s      
     http_req_duration..............: avg=204.41ms min=0s       med=145.3ms  max=3.5s     p(90)=370.57ms p(95)=424.21ms
       { expected_response:true }...: avg=194.66ms min=264.94µs med=145.17ms max=1.8s     p(90)=368.43ms p(95)=417.68ms
     http_req_failed................: 0.39%   ✓ 8283         ✗ 2094467
     http_req_receiving.............: avg=7.43ms   min=0s       med=20.34µs  max=467.42ms p(90)=366µs    p(95)=31.49ms 
     http_req_sending...............: avg=400.37µs min=0s       med=10.37µs  max=493.08ms p(90)=31.65µs  p(95)=141.66µs
     http_req_tls_handshaking.......: avg=0s       min=0s       med=0s       max=0s       p(90)=0s       p(95)=0s      
     http_req_waiting...............: avg=196.58ms min=0s       med=144.88ms max=3.5s     p(90)=344ms    p(95)=375.19ms
     http_reqs......................: 2102750 34889.574111/s
     iteration_duration.............: avg=220.7ms  min=562.23µs med=155.44ms max=3.67s    p(90)=389.8ms  p(95)=456.86ms
     iterations.....................: 2102750 34889.574111/s
     vus............................: 8000    min=8000       max=8000 
     vus_max........................: 8000    min=8000       max=8000 
```
