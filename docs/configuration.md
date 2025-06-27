# Configuration File

This configuration file is divided into two primary types: `static configuration` and `dynamic configuration`.  The format should be `yaml`.

* `static configuration`: This configuration does not update dynamically. To apply any changes, the process must be restarted; only then will the configuration take effect. Examples include modifying server ports.
* `dynamic configuration`: This configuration takes effect immediately upon modification. Currently, only `routes`, `services`, `upstreams`, and `middlewares` fall under this category.

## Table of Contents

* [watch](#watch)
* [timer_resolution](#timer_resolution)
* [event_loops](#event_loops)
* [gopool](#gopool)
* [user_group](#user_group)
* [resolver](#resolver)
* [pid_file](#pid_file)
* [upgrade_sock](#upgrade_sock)
* [providers](#providers)
* [logging](#logging)
* [metrics](#metrics)
* [tracing](#tracing)
* [access_logs](#access_logs)
* [servers](#servers)
* [routes](#routes)
* [services](#services)
* [upstreams](#upstreams)
* [default](#default)

## watch

If `true`, the `watch` features of each provider will be enabled.  By default, `watch` is `true`.

```yaml
watch: true
```

## timer_resolution

Sets the precision of the gateway's time cache settings. The default is 1 second, with a minimum setting of 1 millisecond (ms).

```yaml
timer_resolution: 1ms
```

## event_loops

This represents the number of epoll created by bifrost, which has been automatically adjusted according to the number of P (runtime.GOMAXPROCS(0)) by default, and users generally don't need to care.

```yaml
event_loops: 4
```

## gopool

If `true`, the goroutine pool is used.  Default is `false`

```yaml
gopool: true
```

## user_group

The gateway process runs as a background task (daemon) under a specified user and group.

```yaml
user: nobody
group: nogroup
```

## resolver

The list of dns servers.  By default, Bifrost will resolve all domain name and cache all IPs when the gateway starts.  The cache will not refresh until the gateway is restarted or reloaded.

Example:

```yaml
resolver:
  servers: ["8.8.8.8:53"]
  timeout: 3s
  hosts_file: "/etc/hosts"
  order: ["last", "a", "cname"]
```

| Field      | Type       | Default                  | Description                                                            |
| ---------- | ---------- | ------------------------ | ---------------------------------------------------------------------- |
| servers    | `[]string` |                          | List of dns servers.  Default load dns servers from `/etc/resolv.conf` |
| timeout    | `string`   | `0s`                     | Query timeout for dns server                                           |
| hosts_file | `string`   | `/etc/hosts`             | Path of hosts file                                                     |
| order      | `[]string` | `["last", "a", "cname"]` | Order of dns resolution                                                |

## pid_file

When the gateway process runs as a background task (daemon), the system records the current process's PID in this file.

## upgrade_sock

Facilitates communication between two gateway processes during upgrades via a UNIX socket.

## providers

Please refer to [providers.md](providers.md)

## logging

Error logging configuration.

Example:

```yaml
logging:
  handler: text
  level: info
  output: stderr
```

| Field                   | Type     | Default | Description                                                                      |
| ----------------------- | -------- | ------- | -------------------------------------------------------------------------------- |
| handler                 | `string` | `text`  | Log format; supports `text` and `json`                                           |
| level                   | `string` | `info`  | Log level; options are  `debug`, `info`, `warn`, `error`. Not enabled by default |
| output                  | `string` |         | Log output location, currently supports `stderr` or file path                    |
| disable_redirect_stderr | `bool`   | `false` | Disable redirection of os.stderr to the log file                                 |

## metrics

Monitoring indicators; currently supports `prometheus`.

Example:

```yaml
metrics:
  prometheus:
    enabled: true
    server_id: "apiv1"
    path: "/metrics"
    buckets: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
```

| Field                | Type        | Default                                                                       | Description                    |
| -------------------- | ----------- | ----------------------------------------------------------------------------- | ------------------------------ |
| prometheus.enabled   | `bool`      | `false`                                                                       | Enables prometheus support     |
| prometheus.server_id | `string`    |                                                                               | Server  used to expose metrics |
| prometheus.path      | `string`    | `/metrics`                                                                    | set the metric                 |
| prometheus.buckets   | `[]float64` | `0.005`, `0.01`, `0.025`, `0.05`, `0.1`, `0.25`, `0.5`, `1`, `2.5`, `5`, `10` | Latency bucket levels          |

## tracing

Supports `opentelemetry` for tracing, passing logs to an otel collector server. To enable tracing features, configure it with the tracing middleware.
Bifrost follows [official OpenTelemetry semantic conventions v1.32.0](https://github.com/open-telemetry/semantic-conventions/blob/v1.32.0/docs/http/http-spans.md)

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

| Field         | Type            | Default                   | Description                                                                                                                                                         |
| :------------ | :-------------- | :------------------------ | :------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| enabled       | `bool`          | `false`                   | Enables opentelemetry tracing support                                                                                                                               |
| service_name  | `string`        | `bifrsot`                 | The service name of the gateway                                                                                                                                     |
| propagators   | `[]string`      | `tracecontext`, `baggage` | The supported propagators are: `tracecontext`, `baggage`, `b3`, `jaeger`                                                                                            |
| endpoint      | `string`        | `localhost:4318`          | The address and port of the otel collector. If the endpoint starts with `http` or `https`, it will use the HTTP protocol. Otherwise, it will use the gRPC protocol. |
| insecure      | `bool`          | `false`                   | Certificate verification                                                                                                                                            |
| sampling_rate | `float64`       | `1.0`                     | A given fraction of traces. Fractions >= 1 will always sample. Fractions < 0 are treated as zero.                                                                   |
| batch_size    | `int64`         | `100`                     | Maximum number of spans to be sent in a single batch export                                                                                                         |
| flush         | `time.Duration` | `5s`                      | Maximum time to wait before sending a batch of spans, regardless of batch size                                                                                      |
| queue_size    | `int64`         | `10000`                   | Maximum number of spans that can be queued before being dropped                                                                                                     |
| timeout       | `time.Duration` | `10s`                     | Maximum duration allowed for the entire trace export operation, including connection establishment and data transmission                                            |

## middlewares

Supports custom middleware development with Golang for external middleware. Details are available in the [middlewares](./middlewares.md)

Example:

```yaml
middlewares:
  timing:  # Middleware name, must be unique
    type: timing_logger
```

| Field  | Type     | Default | Description           |
| ------ | -------- | ------- | --------------------- |
| type   | `string` |         | Middleware type       |
| params | `any`    |         | Middleware parameters |

## access_logs

Record the access log; you can use the directives lists in the [directives](./directive.md)

Example:

```yaml
access_logs:
  my_access_log: # Unique request log name for reuse
    output: stderr
    buffering_size: 65536
    time_format: "2006-01-02T15:04:05"
    escape: json
    flush: 1m
    template: >
      {"time":"$time",
      "remote_addr":"$network.peer.address",
      "host":"$http.request.host",
      "request":"$http.request",
      "req_body":"$http.request.body",
      "x_forwarded_for":"$http.request.header.x-forwarded-for",
      "upstream_addr":"$upstream.request.host",
      "upstream_request":"$upstream.request",
      "upstream_duration":"$upstream.duration",
      "upstream_status":"$upstream.response.status_code",
      "status":"$http.response.status_code",
      "duration":"$http.request.duration"}
```

| Field          | Type            | Default               | Description                                                                                                |
| -------------- | --------------- | --------------------- | ---------------------------------------------------------------------------------------------------------- |
| output         | `string`        |                       | Output location; supports `stderr` or file path. If empty, no log will be printed                          |
| buffering_size | `int`           | `65536`               | Output buffer size, in bytes                                                                               |
| time_format    | `string`        | `2006-01-02 15:04:05` | Time format; use [golang time format](https://yourbasic.org/golang/format-parse-string-time-date-example/) |
| escape         | `string`        | `none`                | Escape characters; options are `none`, `json`, `default`                                                   |
| template       | `string`        |                       | Request log format                                                                                         |
| flush          | `time.Duration` | `0`                   | Time interval for writing logs to disk; `0`: allow the OS to flush logs to disk.                           |

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
    observability:
      tracing:
        enabled: true
        attributes:
          network.local.address: "$hostname"
    access_log_id: my_access_log
    middlewares:
      - type: xxx
```

| Field                            | Type                | Default                       | Description                                                                                  |
| -------------------------------- | ------------------- | ----------------------------- | -------------------------------------------------------------------------------------------- |
| bind                             | `string`            |                               | Port binding                                                                                 |
| reuse_port                       | `bool`              | `false`                       | Enables reuse port; Linux only                                                               |
| tcp_fast_open                    | `bool`              | `false`                       | Enables TCP fast open; Linux only                                                            |
| tcp_quick_ack                    | `bool`              | `false`                       | Enables TCP quick ack; Linux only                                                            |
| backlog                          | `int`               |                               | Limits TCP backlog count; Linux only                                                         |
| http2                            | `bool`              | `false`                       | Enables HTTP2                                                                                |
| proxy_protocol                   | `bool`              | `false`                       | Enables proxy protocol (support v1/v2)                                                       |
| logging.handler                  | `string`            | `text`                        | Log format; supports `text`, `json`                                                          |
| logging.level                    | `string`            | `info`                        | Log level; options are `debug`, `info`, `warn`, `error`. Not enabled by default              |
| logging.output                   | `string`            |                               | Output location; supports `stderr` or file path. If empty, no log will be printed            |
| timeout.keepalive                | `time.Duration`     | `60s`                         | Keepalive timeout                                                                            |
| timeout.read                     | `time.Duration`     | `60s`                         | Read timeout                                                                                 |
| timeout.write                    | `time.Duration`     | `60s`                         | Write timeout                                                                                |
| timeout.graceful                 | `time.Duration`     | `10s`                         | Graceful shutdown timeout                                                                    |
| max_request_body_size            | `int`               | `4MB`                         | Max body size of a request.  Unit: byte                                                      |
| read_buffer_size                 | `int`               | `4MB`                         | Set the read buffer size while limiting the HTTP header size.  Unit: byte                    |
| pprof                            | `bool`              | `false`                       | pprof lets you collect CPU profiles, traces, and heap profiles for your Go programs via HTTP |
| access_log_id                    | `string`            |                               | Specifies the access log to use                                                              |
| trusted_cidrs                    | `[]string`          | `0.0.0.0/0`,`::/0`            | Defines trusted CIDR blocks that are known to parse client ip address                        |
| remote_ip_headers                | `[]string`          | `X-Forwarded-For`,`X-Real-IP` | Defines the request header field whose value will be used to replace the client address      |
| observability.tracing.enabled    | `bool`              | `true`                        | Enable or disable the tracing feature                                                        |
| observability.tracing.attributes | `map[string]string` | `true`                        | The attributes of the span                                                                   |
| middlewares                      | `[]Middleware`      |                               | middleware of the server. Details are available in the [middlewares](./middlewares.md)       |

## routes

Routing configuration, controlling request path forwarding rules to a specified `service`. Supports middlewares. Route names must be unique. Details in the [Routing Guide](./routes.md)

```yaml
routes:
  spot-orders: # Unique route name
    methods: []
    paths:
      - /api/v1/orders
    route: /api/v1/orders/{order_id}
    servers: ["extenal", "extenal_tls"]
    tags: ["order"]
    service_id: api-service
    middlewares:
      - type: xxx
```

| Field       | Type           | Default | Description                                                                            |
| ----------- | -------------- | ------- | -------------------------------------------------------------------------------------- |
| methods     | `[]string`     |         | HTTP methods; if empty, all http methods are supported                                 |
| paths       | `[]string`     |         | http path                                                                              |
| route       | `string`       |         | The path template is in the format used for displaying metric paths                    |
| servers     | `[]string`     |         | Servers to apply the route; all servers if empty                                       |
| tags        | `[]string`     |         | List of tags                                                                           |
| service_id  | `string`       |         | Service ID                                                                             |
| middlewares | `[]Middleware` |         | middleware of the routes. Details are available in the [middlewares](./middlewares.md) |

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

| Field              | Type            | Default | Description                                                                             |
| ------------------ | --------------- | ------- | --------------------------------------------------------------------------------------- |
| timeout.read       | `time.Duration` | `60s`   | Read timeout                                                                            |
| timeout.write      | `time.Duration` | `60s`   | Write timeout                                                                           |
| timeout.idle       | `time.Duration` | `60s`   | Idle timeout                                                                            |
| timeout.dail       | `time.Duration` | `60s`   | Dial timeout                                                                            |
| timeout.grpc       | `time.Duration` | `0`     | `grpc` request timeout                                                                  |
| max_conns_per_host | `int64`         | `1024`  | Maximum connections per host                                                            |
| tls_verify         | `bool`          | `false` | Validates server certificate                                                            |
| pass_host_header   | `bool`          | `true`  | Allows to forward client Host header to upstream target                                 |
| protocol           | `string`        | `http`  | Protocol for upstream, `http`, `http2`, `grpc` are supported                            |
| url                | `string`        |         | Upstream URL                                                                            |
| middlewares        | `string`        |         | middleware of the service. Details are available in the [middlewares](./middlewares.md) |

## upstreams

The upstream configuration defines load balancing rules for backend servers. The upstream name must be unique.

```yaml
upstreams:
  test-server: # Unique upstream name
    strategy: "round_robin"
    hash_on: ""
    health_check:
      passive:
        max_fails: 1
        fail_timeout: 10s
    targets:
      - target: "127.0.0.1:8000"
        weight: 1
```

| Field                             | Type            | Default       | Description                                                                          |
| --------------------------------- | --------------- | ------------- | ------------------------------------------------------------------------------------ |
| strategy                          | `string`        | `round_robin` | Load balancing algorithm; supports `round_robin`、`random`、`weighted`、`hashing`    |
| hash_on                           | `string`        |               | Variable used for hash-based load balancing, effective only if strategy is `hashing` |
| health_check.passive.max_fails    | `int32`         | `0`           | Maximum failure count; `0` - indicates no limit                                      |
| health_check.passive.fail_timeout | `time.Duration` | `0`           | Time window for tracking failure counts                                              |
| targets.target                    | `string`        |               | Target address                                                                       |
| targets.weight                    | `int32`         | `1`           | Weight for load balancing                                                            |

## default

The default option allow you to set default value for `service`, `upstream` object.

```yaml
default:
  service:
    timeout:
      read: 3s
      write: 3s
      idle: 600s
      dail: 3s
  upstream:
    max_fails: 1
    fail_timeout: 10s
```

| Field                 | Type            | Default | Description                                                  |
| --------------------- | --------------- | ------- | ------------------------------------------------------------ |
| service.protocol      | `string`        |         | Protocol for upstream, `http`, `http2`, `grpc` are supported |
| service.timeout.read  | `time.Duration` |         | Read timeout                                                 |
| service.timeout.write | `time.Duration` |         | Write timeout                                                |
| service.timeout.idle  | `time.Duration` |         | Idle timeout                                                 |
| service.timeout.dail  | `time.Duration` |         | Dial timeout                                                 |
| service.timeout.grpc  | `time.Duration` |         | `grpc` request timeout                                       |
| upstream.max_fails    | `int32`         |         | Maximum failure count; `0` - indicates no limit              |
| upstream.fail_timeout | `int32`         |         | Time window for tracking failure counts                      |
