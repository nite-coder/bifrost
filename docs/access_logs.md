# Access logs

The access log currently supports the following directives:

| Directive            | Description                                                                      | Example                                 |
| :------------------- | :------------------------------------------------------------------------------- | :-------------------------------------- |
| `$time`              | Time the request log is generated                                                | `2024-03-15T14:30:22.123Z`              |
| `$remote_addr`       | Client IP address                                                                | `192.168.1.100`                         |
| `$host`              | The hostname of the request                                                      | `api.example.com`                       |
| `$service_id`        | The ID of service which handles the request                                      | `user-service-prod`                     |
| `$route_id`          | The ID of route which handles the request                                        | `get-user-profile`                      |
| `$request`           | The client's request.                                                            | `GET /users/profile?id=12345 HTTP/1.1`  |
| `$request_method`    | The HTTP Method of the request.                                                  | `GET`                                   |
| `$request_path`      | The HTTP Path of the request,                                                    | `/users/profile`                        |
| `$request_uri`       | The HTTP URI of the request.                                                     | `/users/profile?id=12345`               |
| `$request_protocol`  | The HTTP protocol of the request.                                                | `HTTP/1.1`                              |
| `$request_body`      | Body of the request                                                              | `{"userId": 12345, "action": "update"}` |
| `$header_{xxx}`      | Client request header; replace {xxx} with the specific header name               | `$header_X-Forwarded-For`               |
| `$upstream`          | The request path send to upstream                                                | `GET /users/profile?id=12345 HTTP/1.1`  |
| `$upstream_id`       | The ID of upstream which handles the request                                     | `backend-cluster-01`                    |
| `$upstream_addr`     | Upstream host address                                                            | `10.0.0.50:8080`                        |
| `$upstream_uri`      | Upstream's requested HTTP URI                                                    | `/internal/user/12345`                  |
| `$upstream_duration` | Time taken to process the upstream request                                       | `0.125`                                 |
| `$upstream_status`   | HTTP STATUS CODE returned by the upstream target                                 | `200`                                   |
| `$grpc_status`       | GRPC STATUS CODE returned by the upstream target                                 | `0`                                     |
| `$grpc_messaage`     | GRPC Message returned by the upstream target                                     | `OK`                                    |
| `$duration`          | Total time from when the client request was sent to when a response was returned | `0.250`                                 |
| `$trace_id`          | Trace ID for tracking the request                                                | `ab123456-7890-1234-5678-90abcdef1234`  |
| `$status`            | HTTP STATUS CODE returned to the client                                          | `200`                                   |