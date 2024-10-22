# 配置文件

在配置文件的部分，目前有分兩種類型，`靜態配置` 與`動態配置`。

* `靜態配置`: 這種配置的內容無法動態更新，如果想要更新則必須要重啟服務後配置才會生效，例如更換 servers 端口等
* `動態配置`: 屬於這一類的配置，當內容修改後，則配置會立即生效，目前只有 `routes`, `services`, `upstreams`, `middlewares` 配置屬於這配置屬於這類型

## 目錄

* [watch](#watch)
* [timer_resolution](#timer_resolution)
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

是否開啟配置監控，如果當配置文件被修改後, `動態配置` 部分的配置可以立即生效

```yaml
watch: true #立即生效
```

## timer_resolution

配置網關時間精度，目前預設值是 1 秒，這個配置最低可以調整到 1毫秒 (ms)

```yaml
timer_resolution: 1ms
```

## pid_file

當 bifrost 程序是透過背景任務 (daemon) 來運作的，系統會把當前的 `process` 的 pid 記錄到這個檔案中

## upgrade_sock

這個檔案是兩個 bifrost process 再升級的過程中透過 unix socket 來溝通的方式

## providers

Providers 提供與不同的服務對接，可以用來管理配置文件與服務發現等，目前只支持 `file` Provider

### file

可以透過文件來配置網關

範例:

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

| 欄位       | 預設值                 | 說明                                                         |
| ---------- | ---------------------- | ------------------------------------------------------------ |
| enabled    | false                  | 是否開啟 file provider                                       |
| extensions | `.yaml`,`.yml`, `json` | 哪些檔案的附檔名才可以被載入                                 |
| paths      | `.`                    | 有哪些目錄或檔案文件需要被載入, 如果沒有配置則為當前目錄路徑 |

## logging

錯誤日誌

範例:

```yaml
logging:
  handler: text
  level: info
  output: stderr
```

| 欄位    | 預設值 | 說明                                                                  |
| ------- | ------ | --------------------------------------------------------------------- |
| handler | text   | 日誌格式，目前支持的格式有 `text`, `json`                             |
| level   |        | 日誌等級，目前支持的有 `debug`, `info`, `warn`, `error`，預設是不開啟 |
| output  | `.`    | 日誌輸出地方，目前有 `stderr`，文件路徑                               |

## metrics

監控指標，目前支持 `prometheus`

範例:

```yaml
metrics:
  prometheus:
    enabled: true
    buckets: [0.005000, 0.010000, 0.025000, 0.050000, 0.10000, 0.250000, 0.500000, 1.00000, 2.50000, 5.000000, 10.000000]
```

| 欄位               | 預設值                                                                                                     | 說明                     |
| ------------------ | ---------------------------------------------------------------------------------------------------------- | ------------------------ |
| prometheus.enabled | false                                                                                                      | 是否開啟 prometheus 支持 |
| prometheus.buckets | 0.005000, 0.010000, 0.025000, 0.050000, 0.10000, 0.250000, 0.500000, 1.00000, 2.50000, 5.000000, 10.000000 | 延遲等級分類             |

## tracing

鏈路追蹤，目前支持 `opentelemetry` 標準，這邊的配置只包含把日誌往 otel server 傳遞，如需要開啟鏈路追蹤功能需要搭配 `tracing` 的中間件

範例:

```yaml
tracing:
  otlp:
    enabled: true
    http:
      endpoint: otel-collector:4318
      insecure: true
    grpc:
      endpoint: otel-collector:4317
      insecure: true
```

| 欄位               | 預設值 | 說明                                |
| ------------------ | ------ | ----------------------------------- |
| otlp.enabled       | false  | 是否開啟 opentelemetry tracing 支持 |
| otlp.http.endpoint |        | otlp collector http 端口            |
| otlp.http.insecure | false  | 是否驗證憑證                        |
| otlp.grpc.endpoint |        | otlp collector grpc 端口            |
| otlp.grpc.insecure | false  | 是否驗證憑證                        |

## middlewares

中間件(插件)，支持使用原生 Golang 開發自定義外部中間件，細節可以參考[中間件](./middlewares.md)

範例:

```yaml
middlewares:
  timing:  #中間件名稱，必須是唯一值
    type: timing_logger
```

| 欄位   | 預設值 | 說明           |
| ------ | ------ | -------------- |
| type   |        | 中間件類型     |
| params |        | 中間件配置參數 |

## access_logs

請求日誌; 詳細變量支持可參考[請求日誌](./access_logs.md)

範例:

```yaml
access_logs:
  my_access_log: #請求日誌名稱, 可透過名稱複用，必須是唯一值
    enabled: true
    output: stderr
    buffering_size: 65536
    time_format: "2006-01-02T15:04:05"
    escape: json
    flush: 1m
    template: >
      {"time":"$time",
      "remote_addr":"$remote_addr",
      "host": "$host",
      "request_uri":"$request_method $request_uri $request_protocol",
      "req_body":"$request_body",
      "x_forwarded_for":"$header_X-Forwarded-For",
      "upstream_addr":"$upstream_addr",
      "upstream_uri":"$upstream_method $upstream_uri $upstream_protocol",
      "upstream_duration":$upstream_duration,
      "upstream_status":$upstream_status,
      "status":$status,
      "duration":$duration,
      "trace_id":"$trace_id"}
```

| 欄位           | 預設值 | 說明                                         |
| -------------- | ------ | -------------------------------------------- |
| enabled        | false  | 是否開啟請求日誌                             |
| output         |        | 輸出; 目前支持 `stderr` 或文件路徑           |
| buffering_size | 64 KB  | 輸出緩衝；單位 byte                          |
| time_format    |        | 時間格式                                     |
| escape         | none   | 字元跳脫; 目前支持 `none`, `json`, `default` |
| template       |        | 請求日誌格式                                 |
| flush          | 1m     | 定時多久把日誌寫入到 disk                    |

## servers

服務器組態，支持 `middlwares` 使用，用來控制哪個端口需要對外等，`server` 名稱必須是唯一值

```yaml
servers:
  my-wallet:  # server 名稱必須是唯一值
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

| 欄位              | 預設值 | 說明                                                                  |
| ----------------- | ------ | --------------------------------------------------------------------- |
| bind              |        | 端口綁定                                                              |
| reuse_port        | false  | 是否使用 reuse port; (僅支持 linux)                                   |
| tcp_fast_open     | false  | 是否開啟 tcp fast open; (僅支持 linux)                                |
| tcp_quick_ack     | false  | 是否開啟 tcp quickack (僅支持 linux)                                  |
| backlog           |        | 限制 tcp backlog 數量 (僅支持 linux)                                  |
| http2             | false  | 是否開啟 http2                                                        |
| logging.handler   | text   | 日誌格式，目前支持的格式有 `text`, `json`                             |
| logging.level     | ""     | 日誌等級，目前支持的有 `debug`, `info`, `warn`, `error`，預設是不開啟 |
| logging.output    | `.`    | 日誌輸出地方，目前有 `stderr`，文件路徑                               |
| timeout.keepalive | 60s    | keepalive 超時時間                                                    |
| timeout.read      | 60s    | 讀取的超時時間                                                        |
| timeout.write     | 60s    | 寫入的超時時間                                                        |
| timeout.graceful  | 10s    | 優雅關閉的超時時間                                                    |
| access_log_id     |        | 使用哪個請求日誌                                                      |

## routes

路由組態，用來控制請求路徑的轉發規則到哪一個 `service` 上，支持 middlwares 使用，路由名稱必須是唯一值，更詳細的用法可以參考[路由用法](./routes.md)

```yaml
routes:
  spot-orders: # 路由名稱必須是唯一值
    methods: []
    paths:
      - /api/v1
    servers: ["extenal", "extenal_tls"]
    service_id: api-service
    middlewares:
      - type: tracing
```

| 欄位       | 預設值 | 說明                                                              |
| ---------- | ------ | ----------------------------------------------------------------- |
| methods    |        | http method; 為空則支持全部 http method                           |
| paths      |        | http path                                                         |
| servers    |        | 選擇路由要套用到哪個 server 端口上，如果為空則全部 servers 都支持 |
| service_id |        | 服務 ID                                                           |

## services

業務服務組態，用來控制服務配置，例如協議資訊等，不同業務可以共用同一個 `upstream`，同時可以依照業務屬性配置不同的 service 參數等，服務名稱必須要是唯一值

```yaml
services:
  api-service: # 服務名稱必須是唯一值
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

| 欄位               | 預設值 | 說明                                          |
| ------------------ | ------ | --------------------------------------------- |
| timeout.read       | 60s    | 讀取的超時時間                                |
| timeout.write      | 60s    | 寫入的超時時間                                |
| timeout.idle       | 60s    | 閒置超時時間                                  |
| timeout.dail       | 60s    | 撥接超時時間                                  |
| timeout.grpc       | 0      | `grpc` 請求超時時間                           |
| max_conns_per_host | 1024   | 連線上游每台主機的最多連線數                  |
| tls_verify         | false  | 是否驗證憑證                                  |
| protocol           | http   | 轉發上游協議; 支持 `http`, `http2`, `grpc`    |
| url                |        | 轉發上游路徑，支持使用 `upstream` 名稱當 host |

## upstreams

上游組態，用來控制後端主機的負載均衡規則等，上游名稱必須是唯一值

```yaml
upstreams:
  test-server: # 上游名稱必須是唯一值
    strategy: "round_robin"
    hash_on: ""
    targets:
      - target: "127.0.0.1:8000"
        max_fails: 1
        fail_timeout: 10s
        weight: 1
```

| 欄位                 | 預設值      | 說明                                                                 |
| -------------------- | ----------- | -------------------------------------------------------------------- |
| strategy             | round_robin | 負載均衡算法; 目前支持`round_robin`、`random`、`weighted`、`hashing` |
| hash_on              |             | 依照哪個變量來計算哈希負載均衡，僅當 strategy 為 `hashing` 生效      |
| targets.target       |             | 目標 IP                                                              |
| targets.max_fails    | 0           | 失敗次數; `0`- 無限制                                                |
| targets.fail_timeout | 10s         | 失敗次數的有效時間範圍                                               |
| targets.weight       | 1           | 權重                                                                 |
