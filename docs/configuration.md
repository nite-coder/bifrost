# Configuration File

This configuration file is divided into two primary types: `static configuration` and `dynamic configuration`.

* `static configuration`: This configuration does not update dynamically. To apply any changes, the process must be restarted; only then will the configuration take effect. Examples include modifying server ports.
* `dynamic configuration`: This configuration takes effect immediately upon modification. Currently, only `routes`, `services`, `upstreams`, and `middlewares` fall under this category.

## Table of Contents

* [watch](#watch)
* [timer_resolution](#timer_resolution)
* [num_loops](#num_loops)
* [resolver](#resolver)
* [pid_file](#pid_file)
* [upgrade_sock](#upgrade_sock)
* [providers](#providers)
  * [file](#file)
* [logging](#logging)
* [metrics](#metrics)
* [tracing](#tracing)
* [access_logs](#access_logs)
* [servers](#servers)
* [routes](#routes)
* [services](#services)
* [upstreams](#upstreams)

## watch

Determines if configuration monitoring is enabled. When enabled, `dynamic configuration` changes take effect immediately upon file modification.

```yaml
watch: true # Immediate effect
```

## timer_resolution

Sets the precision of the gateway's time settings. The default is 1 second, with a minimum setting of 1 millisecond (ms).

```yaml
timer_resolution: 1ms
```

## num_loops

This represents the number of epoll created by bifrost, which has been automatically adjusted according to the number of P (runtime.GOMAXPROCS(0)) by default, and users generally don’t need to care.

```yaml
num_loops: 4
```

## resolver

The DNS resolver configuration.  By default, Bifrost will resolve all domain name and cache all IPs at beginning.  The cache will not refresh until the gateway is restarted or reloaded.

Example:

```yaml
resolver:
  addr_port: "8.8.8.8:53"
  valid: 5s
  skip_test: false
```

| Field     | Default | Description                                                                        |
| --------- | ------- | ---------------------------------------------------------------------------------- |
| addr_port |         | DNS server address and port.  If empty, the local `/etc/resolv.conf` will be used. |
| valid     | 0       | Time to refresh the DNS cache.  `0`: means no refresh.  It must be greater than 0. |
| skip_test | false   | Skip the dns check during testing                                                  |

## pid_file

When the gateway process runs as a background task (daemon), the system records the current process's PID in this file.

## upgrade_sock

Facilitates communication between two gateway processes during upgrades via a UNIX socket.

## providers

Providers enable integration with various services, managing configuration files and service discovery. Currently, only the `file` provider is supported.

### file

Allows gateway configuration through files.

Example:

```yaml
providers:
  file:
    enabled: true
    extensions:
      - ".yaml"
      - ".yml"
      - ".json"
    paths:
      - "./conf"
```

| Field      | Default                | Description                                                                                                         |
| ---------- | ---------------------- | ------------------------------------------------------------------------------------------------------------------- |
| enabled    | false                  | Enables the file provider                                                                                           |
| extensions | `.yaml`,`.yml`, `json` | Allowed file extensions                                                                                             |
| paths      |                        | Directories or files to be loaded.  Recursively traverse all subdirectories and files under the specified directory |

## logging

Error logging configuration.

Example:

```yaml
logging:
  handler: text
  level: info
  output: stderr
```

| Field   | Default | Description                                                                      |
| ------- | ------- | -------------------------------------------------------------------------------- |
| handler | text    | Log format; supports `text` and `json`                                           |
| level   | info    | Log level; options are  `debug`, `info`, `warn`, `error`. Not enabled by default |
| output  |         | Log output location, currently supports `stderr` or file path                    |

## metrics

Monitoring indicators; currently supports `prometheus`.

Example:

```yaml
metrics:
  prometheus:
    enabled: true
    server_id: "apiv1"
    path: "/metrics"
    buckets: [0.005000, 0.010000, 0.025000, 0.050000, 0.10000, 0.250000, 0.500000, 1.00000, 2.50000, 5.000000, 10.000000]
```

| Field                | Default                                                                                                    | Description                    |
| -------------------- | ---------------------------------------------------------------------------------------------------------- | ------------------------------ |
| prometheus.enabled   | false                                                                                                      | Enables prometheus support     |
| prometheus.server_id |                                                                                                            | Server  used to expose metrics |
| prometheus.path      | `/metrics`                                                                                                 | set the metric                 |
| prometheus.buckets   | 0.005000, 0.010000, 0.025000, 0.050000, 0.10000, 0.250000, 0.500000, 1.00000, 2.50000, 5.000000, 10.000000 | Latency bucket levels          |

## tracing

Supports `opentelemetry` for tracing, passing logs to an otel collector server. To enable tracing features, configure it with the tracing middleware.
Bifrost follows [official OpenTelemetry semantic conventions v1.26.0](https://github.com/open-telemetry/semantic-conventions/blob/v1.26.0/docs/http/http-spans.md)

Example:

```yaml
tracing:
  enabled: false
  service_name: "bifrost"
  propagators: ["tracecontext", "baggage"]
  endpoint: otel-collector:4317
  insecure: true
  sampling_rate: 1.0
  batch_size: 500
  flush: 2s
  queue_size: 50000
```

| Field         | Default                   | Description                                                                                                              |
| ------------- | ------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| enabled       | false                     | Enables opentelemetry tracing support                                                                                    |
| service_name  | `bifrsot`                 | The service name of the gateway                                                                                          |
| propagators   | `tracecontext`, `baggage` | The supported propagators are: `tracecontext`, `baggage`, `b3`, `jaeger`                                                 |
| endpoint      | `localhost:4318`          | The address and port of the otel collector                                                                               |
| insecure      | false                     | Certificate verification                                                                                                 |
| sampling_rate | 1.0                       | otlp collector grpc port                                                                                                 |
| batch_size    | 100                       | Maximum number of spans to be sent in a single batch export                                                              |
| flush         | `5s`                      | Maximum time to wait before sending a batch of spans, regardless of batch size                                           |
| queue_size    | 10000                     | Maximum number of spans that can be queued before being dropped                                                          |
| timeout       | `10s`                     | Maximum duration allowed for the entire trace export operation, including connection establishment and data transmission |

## middlewares

Supports custom middleware development with Golang for external middleware. Details are available in the [middlewares](./middlewares.md)

Example:

```yaml
middlewares:
  timing:  # Middleware name, must be unique
    type: timing_logger
```

| Field  | Default | Description           |
| ------ | ------- | --------------------- |
| type   |         | Middleware type       |
| params |         | Middleware parameters |

## access_logs

Request logging; variables are detailed in the [access logs](./access_logs.md)

Example:

```yaml
access_logs:
  my_access_log: # Unique request log name for reuse
    enabled: true
    output: stderr
    buffering_size: 65536
    time_format: "2006-01-02T15:04:05"
    escape: json
    flush: 1m
    template: >
      {"time":"$time",
      "remote_addr":"$network.peer.address",
      "host": "$http.request.host",
      "request":"$http.request",
      "req_body":"$http.request.body",
      "x_forwarded_for":"$http.request.header.x-forwarded-for",
      "upstream_addr":"$upstream.request.host",
      "upstream_request":"$upstream.request",
      "upstream_duration":$upstream.duration,
      "upstream_status":$upstream.response.status_code,
      "status":$http.response.status_code,
      "grpc_status":"$grpc.status_code",
      "grpc_messaage":"$grpc.message",
      "duration":$duration}
```

| Field          | Default | Description                                              |
| -------------- | ------- | -------------------------------------------------------- |
| enabled        | false   | Enables request logging                                  |
| output         |         | Output location; supports `stderr` or file path          |
| buffering_size | 64 KB   | Output buffer size, in bytes                             |
| time_format    |         | Time format                                              |
| escape         | none    | Escape characters; options are `none`, `json`, `default` |
| template       |         | Request log format                                       |
| flush          | 1m      | Time interval for writing logs to disk                   |

## servers

Server configuration, supporting `middlewares` for port control and other settings. Server names must be unique.

```yaml
servers:
  my-wallet:  # Unique server name
    bind: ":8001"
    reuse_port: false
    tcp_fast_open: false
    tcp_quick_ack: false
    http2: false
    logging:
      level: info
      handler: text
      output: "./logs/extenal.log"
    timeout:
      keepalive: 60s
      read: 60s
      write: 60s
      graceful: 10s
    access_log_id: my_access_log
    middlewares:
      - type: tracing
```

| Field                 | Default | Description                                                                                  |
| --------------------- | ------- | -------------------------------------------------------------------------------------------- |
| bind                  |         | Port binding                                                                                 |
| reuse_port            | false   | Enables reuse port; Linux only                                                               |
| tcp_fast_open         | false   | Enables TCP fast open; Linux only                                                            |
| tcp_quick_ack         | false   | Enables TCP quick ack; Linux only                                                            |
| backlog               |         | Limits TCP backlog count; Linux only                                                         |
| http2                 | false   | Enables HTTP2                                                                                |
| logging.handler       | text    | Log format; supports `text`, `json`                                                          |
| logging.level         | info    | Log level; options are `debug`, `info`, `warn`, `error`. Not enabled by default              |
| logging.output        |         | Log output location; `stderr` or file path                                                   |
| timeout.keepalive     | 60s     | Keepalive timeout                                                                            |
| timeout.read          | 60s     | Read timeout                                                                                 |
| timeout.write         | 60s     | Write timeout                                                                                |
| timeout.graceful      | 10s     | Graceful shutdown timeout                                                                    |
| max_request_body_size | 4MB     | Max body size of a request.  Unit: byte                                                      |
| read_buffer_size      | 4MB     | Set the read buffer size while limiting the HTTP header size.  Unit: byte                    |
| pprof                 | false   | pprof lets you collect CPU profiles, traces, and heap profiles for your Go programs via HTTP |
| access_log_id         |         | Specifies the access log to use                                                              |
| middlewares           |         | middleware of the server                                                                     |

## routes

Routing configuration, controlling request path forwarding rules to a specified `service`. Supports middlewares. Route names must be unique. Details in the [Routing Guide](./routes.md)

```yaml
routes:
  spot-orders: # Unique route name
    methods: []
    paths:
      - /api/v1
    servers: ["extenal", "extenal_tls"]
    service_id: api-service
    middlewares:
      - type: tracing
```

| Field       | Default | Description                                      |
| ----------- | ------- | ------------------------------------------------ |
| methods     |         | HTTP methods; if empty, all methods supported    |
| paths       |         | http path                                        |
| servers     |         | Servers to apply the route; all servers if empty |
| service_id  |         | Service ID                                       |
| middlewares |         | middleware of the routes                         |

## services

Defines business services, managing configuration, protocol details, and other settings. Services can share the same `upstream`. Service names must be unique.

```yaml
services:
  api-service: # Unique service name
    timeout:
      read: 3s
      write: 3s
      idle: 600s
      dail: 3s
    max_conns_per_host: 1
    tls_verify: false
    protocol: http
    url: http://test-server:8000
```

| Field              | Default | Description                                                  |
| ------------------ | ------- | ------------------------------------------------------------ |
| timeout.read       | 60s     | Read timeout                                                 |
| timeout.write      | 60s     | Write timeout                                                |
| timeout.idle       | 60s     | Idle timeout                                                 |
| timeout.dail       | 60s     | Dial timeout                                                 |
| timeout.grpc       | 0       | `grpc` request timeout                                       |
| max_conns_per_host | 1024    | Maximum connections per host, no limit if `0`                |
| tls_verify         | false   | Validates server certificate                                 |
| protocol           | http    | Protocol for upstream, `http`, `http2`, `grpc` are supported |
| url                |         | Upstream URL                                                 |
| middlewares        |         | middleware of the service                                    |

## upstreams

The upstream configuration defines load balancing rules for backend servers. The upstream name must be unique.

```yaml
upstreams:
  test-server: # Unique upstream name
    strategy: "round_robin"
    hash_on: ""
    targets:
      - target: "127.0.0.1:8000"
        max_fails: 1
        fail_timeout: 10s
        weight: 1
```

| Field                | Default     | Description                                                                          |
| -------------------- | ----------- | ------------------------------------------------------------------------------------ |
| strategy             | round_robin | Load balancing algorithm; supports `round_robin`、`random`、`weighted`、`hashing`    |
| hash_on              |             | Variable used for hash-based load balancing, effective only if strategy is `hashing` |
| targets.target       |             | Target IP                                                                            |
| targets.max_fails    | 0           | Maximum failure count; `0` - indicates no limit                                      |
| targets.fail_timeout | 10s         | Time window for tracking failure counts                                              |
| targets.weight       | 1           | Weight for load balancing                                                            |
