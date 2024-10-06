# 請求日誌 ( Access logs)

請求日誌目前支持下面變量

| 變量                 |                                                                                 說明 |
| :------------------- | :----------------------------------------------------------------------------------- |
| `$time`              |                                                                     請求日誌生成時間 |
| `$remote_addr`       |                                                                            客戶端 IP |
| `$host`              |                                                                       客戶端請求域名 |
| `$request_method`    |                                                               客戶端請求 HTTP METHOD |
| `$request_uri`       |                                                                  客戶端請求 HTTP URI |
| `$request_protocol`  |                                                                       請求 HTTP 協議 |
| `$request_body`      |                                                                            請求 Body |
| `$header_{xxx}`      | 客戶端請求 header; {xxx} 替換你要抓取的 header 名稱，例如: `$header_X-Forwarded-For` |
| `$upstream_addr`     |                                                                         上游主機地址 |
| `$upstream_uri`      |                                                                上游請求請求 HTTP URI |
| `$upstream_duration` |                                                                         上游處裡耗時 |
| `$upstream_status`   |                                                        上游服務返回 HTTP STATUS CODE |
| `$grpc_status`       |                                                        上游服務返回 GRPC STATUS CODE |
| `$grpc_messaage`     |                                                            上游服務返回 GRPC Message |
| `$duration`          |                                                         客戶端發請求到返回的完整耗時 |
| `$trace_id`          |                                                                               追蹤ID |
| `$status`            |                                                           返回客戶端HTTP STATUS CODE |
