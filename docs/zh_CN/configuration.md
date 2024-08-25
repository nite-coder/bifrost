# 配置文件

在配置文件的部分，目前有分兩種類型，`靜態配置` 與`動態配置`。

* `靜態配置`: 這種配置的內容無法動態更新，如果想要更新則必須要重啟服務後配置才會生效，例如更換 server 端口等
* `動態配置`: 屬於這一類的配置，當內容修改後，則配置會立即生效，例如: `routes`, `services`, `upstreams`, `middlewares`

## 目錄
   * [providers](#providers)
      + [file](#file)
   * [logging](#logging)
   * [metrics](#metrics)
   * [tracing](#tracing)
   * [access_logs](#access_logs)
   * [servers](#servers)

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
| level   | ""     | 日誌等級，目前支持的有 `debug`, `info`, `warn`, `error`，預設是不開啟 |
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

| 欄位               | 預設值 | 說明                     |
| ------------------ | ------ | ------------------------ |
| prometheus.enabled | false  | 是否開啟 prometheus 支持 |
| prometheus.buckets |        | 延遲等級分類             |


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


## access_logs

請求日誌

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
| escape         |        | 字元跳脫; 目前支持 `none`, `json`, `default` |
| template       |        | 請求日誌格式                                 |


## servers

管理服務器組態

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
      read: 3s
      write: 3s
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
| timeout.keepalive |        | keepalive 超時時間                                                    |
| timeout.read      |        | 讀取的超時時間                                                        |
| timeout.write     |        | 寫入的超時時間                                                        |
| timeout.graceful  |        | 優雅關閉的超時時間                                                    |
| access_log_id     |        | 使用哪個請求日誌                                                      |