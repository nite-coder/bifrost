# 配置文件

動態更新目前支持 `routes`, `services`, `upstreams`, `middlewares`

```yaml
providers:
  file:
    enabled: true
    paths:
      - "./conf"
    watch: true

logging:
  handler: text
  level: debug    # none, debug, info, warn, error
  output: stderr

metrics:
  prometheus:
    enabled: false
    buckets: [0.01, 0.03, 0.05, 0.1]

access_logs:
  my_access_log:  # access log 的名称, 必须是唯一的
    enabled: false
    output: stderr
    buffering_size: 65536
    time_format: "2006-01-02T15:04:05"
    escape: json
    template: >
      {"time":"$time",
      "remote_addr":"$remote_addr",
      "request_uri":"$request_method $request_uri $request_protocol",
      "req_body":"$request_body",
      "x_forwarded_for":"$header_X-Forwarded-For",
      "upstream_addr":"$upstream_addr",
      "upstream_uri":"$upstream_method $upstream_uri $upstream_protocol",
      "upstream_duration":$upstream_duration,
      "upstream_status":$upstream_status,
      "status":$status,
      "duration":$duration}

tracing:
  enabled: false
  otlp:
    http:
      endpoint: http://localhost:4318/v1/traces
    grpc:
      endpoint: localhost:4317
      insecure: true

servers:
  extenal:
    bind: ":80"
    reuse_port: true
    tls:
      cert_pem: ""
      key_pem: ""
    http2: false
    logging:
      level: debug
      handler: text
      output: stderr
    timeout:
      keepalive_timeout: 120s
      read_timeout: 60s
      write_timeout: 60s
      graceful_timeout: 1s
    access_log_id: my_access_log
    pprof: false  ## 是否開啟 go pprof
    middlewares:
      - use: timing

routes:
  spot-orders:
    methods: []
    paths:
      - /spot/orders
    servers: ["extenal"]
    service_id: spot-orders
    middlewares:
      - type: add_prefix
        params:
          prefix: /api/v1


services:
  spot-orders:
    read_timeout: 5s
    write_timeout: 5s
    idle_timeout: 5s
    dail_timeout: 5s
    max_conns_per_host: 1024  # default: 1024
    tls_verify: false
    protocol: http
    url: http://localhost:8000
    middlewares:


upstreams:
  default:
    strategy: "round_robin"
    hash_on: ""
    targets:
      - target: "127.0.0.1:8000"
        weight: 30
      - target: "127.0.0.1:800"
        weight: 70
```
