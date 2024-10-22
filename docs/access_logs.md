# 請求日誌 ( Access logs)

The request log currently supports the following variables:

| 變量                 | 說明                                                                                                |
| :------------------- | :-------------------------------------------------------------------------------------------------- |
| `$time`              | Time the request log is generated                                                                   |
| `$remote_addr`       | Client IP address                                                                                   |
| `$host`              | Client's requested hostname                                                                         |
| `$request_method`    | Client's requested HTTP METHOD                                                                      |
| `$request_uri`       | Client's requested HTTP URI                                                                         |
| `$request_protocol`  | HTTP protocol of the request                                                                        |
| `$request_body`      | Body of the request                                                                                 |
| `$header_{xxx}`      | Client request header; replace {xxx} with the specific header name, e.g., `$header_X-Forwarded-For` |
| `$upstream_addr`     | Upstream host address                                                                               |
| `$upstream_uri`      | Upstream's requested HTTP URI                                                                       |
| `$upstream_duration` | Time taken to process the upstream request                                                          |
| `$upstream_status`   | HTTP STATUS CODE returned by the upstream target                                                    |
| `$grpc_status`       | GRPC STATUS CODE returned by the upstream target                                                    |
| `$grpc_messaage`     | GRPC Message returned by the upstream target                                                        |
| `$duration`          | Total time from when the client request was sent to when a response was returned                    |
| `$trace_id`          | Trace ID for tracking the request                                                                   |
| `$status`            | HTTP STATUS CODE returned to the client                                                             |
