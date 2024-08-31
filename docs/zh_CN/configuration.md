# 配置文件

在配置文件的部分，目前有分兩種類型，`靜態配置` 與`動態配置`。

* `靜態配置`: 這種配置的內容無法動態更新，如果想要更新則必須要重啟服務後配置才會生效，例如更換 servers 端口等
* `動態配置`: 屬於這一類的配置，當內容修改後，則配置會立即生效，目前只有 `routes`, `services`, `upstreams`, `middlewares` 配置屬於這配置屬於這類型

## 目錄
   * [providers](#providers)
      + [file](#file)
   * [logging](#logging)
   * [metrics](#metrics)
   * [tracing](#tracing)
   * [access_logs](#access_logs)
   * [servers](#servers)
   * [routes](#routes)
   * [services](#services)
   * [upstreams](#upstreams)

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
    watch: true
```

| 欄位       | 預設值                 | 說明                                                         |
| ---------- | ---------------------- | ------------------------------------------------------------ |
| enabled    | false                  | 是否開啟 file provider                                       |
| extensions | `.yaml`,`.yml`, `json` | 哪些檔案的附檔名才可以被載入                                 |
| paths      | `.`                    | 有哪些目錄或檔案文件需要被載入, 如果沒有配置則為當前目錄路徑 |
| watch      | false                  | 是否開啟監控，開啟後動態配置文件修改後將立即生效             |

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
    buckets: [0.01, 0.03, 0.05, 0.1]
```

| 欄位               | 預設值                                 | 說明                     |
| ------------------ | -------------------------------------- | ------------------------ |
| prometheus.enabled | false                                  | 是否開啟 prometheus 支持 |
| prometheus.buckets | 0.100000, 0.300000, 1.200000, 5.000000 | 延遲等級分類             |


## tracing

追蹤，目前支持 `opentelemetry`

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
| buffering_size | 64 KB  | 輸出緩衝                                     |
| time_format    |        | 時間格式                                     |
| escape         | none   | 字元跳脫; 目前支持 `none`, `json`, `default` |
| template       |        | 請求日誌格式                                 |


## servers

服務器組態，支持 middlwares 使用

```yaml
servers:
  api:  # server 名稱，必須是唯一值
    bind: ":8001"
    reuse_port: false
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
| reuse_port        | false  | 是否使用 reuse port                                                   |
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

路由組態，支持 middlwares 使用，更詳細的用法可以參考[路由用法](./routes.md)

```yaml
routes:
  spot-orders: # 路由名稱，必須是唯一值
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

業務服務組態，不同業務可以共用同一個 upstream，同時可以依照業務屬性配置不同的 service 參數等

```yaml
services:
  api-service: # 服務名稱，必須是唯一值
    timeout:
      read: 3s
      write: 3s
      idle: 600s
      dail: 3s
    #max_conns_per_host: 1
    tls_verify: false
    protocol: http
    url: http://test-server:8000
```

| 欄位          | 預設值 | 說明                                        |
| ------------- | ------ | ------------------------------------------- |
| timeout.read  | 60s    | 讀取的超時時間                              |
| timeout.write | 60s    | 寫入的超時時間                              |
| timeout.idle  | 60s    | 閒置超時時間                                |
| timeout.dail  | 60s    | 撥接超時時間                                |
| tls_verify    | false  | 是否驗證憑證                                |
| protocol      | http   | 轉發上游協議; 支持 http, http2              |
| url           |        | 轉發上游路徑，支持使用 upstream 名稱當 host |

## upstreams

上游組態

```yaml
upstreams:
  test-server: # 上游名稱，必須是唯一值
    strategy: "round_robin"
    hash_on: ""
    targets:
      - target: "127.0.0.1:8000"
        max_fails: 1
        fail_timeout: 10s
        weight: 1
```

| 欄位                 | 預設值      | 說明                                                             |
| -------------------- | ----------- | ---------------------------------------------------------------- |
| strategy             | round_robin | 分流算法; 目前支持`round_robin`、`random`、`weighted`、`hashing` |
| hash_on              |             | 依照哪個變量來計算哈希分流，僅當 strategy 為 `hashing` 生效      |
| targets.target       |             | 目標 IP                                                          |
| targets.max_fails    | 0           | 失敗次數; `0`: 無限制                                            |
| targets.fail_timeout | 10s         | 失敗次數有效時間範圍                                             |
| targets.weight       | 1           | 權重                                                             |