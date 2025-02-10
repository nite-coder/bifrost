# Directive

The currently supports the following directives:

| Directive                        | Description                                                                                                          | Example                                 |
| :------------------------------- | :------------------------------------------------------------------------------------------------------------------- | :-------------------------------------- |
| `$time`                          | Time the request log is generated. (use timecache)                                                                   | `2024-03-15 14:30:21`                   |
| `$hostname`                      | The hostname of server                                                                                               | `sim01`                                 |
| `$network.peer.address`          | Peer address of the network connection                                                                               | `192.168.1.100`                         |
| `$service_id`                    | The service id of the request                                                                                        | `user-service-prod`                     |
| `$route_id`                      | The route id of the request                                                                                          | `get-user-profile`                      |
| `$upstream_id`                   | The upstream id of the request                                                                                       | `backend-cluster-01`                    |
| `$error.type`                    | The error type of the request. You need to write a middleware to extract the error type first.                       | `YOUR_ERROR_CODE`                       |
| `$error.message`                 | The error message of the request. You need to write a middleware to extract the error type first.                    | `YOUR_ERROR_MESSAGE`                    |
| `$trace_id`                      | The trace id of the requst                                                                                           | `33430285bcba2e8b593fa5f1bafc7507`      |
| `$http.start`                    | HTTP request start                                                                                                   | `1737793075600742`                      |
| `$http.end`                      | HTTP request finish                                                                                                  | `1737793075600742`                      |
| `$http.request`                  | HTTP request info                                                                                                    | `GET /users/profile?id=12345 HTTP/1.1`  |
| `$http.request.size`             | The total size of the request in bytes. This should be the total number of bytes sent over the wire (unit:bye)       | `832000`                                |
| `$http.request.scheme`           | HTTP request scheme                                                                                                  | `http`                                  |
| `$http.request.host`             | HTTP request host                                                                                                    | `api.example.com`                       |
| `$http.request.method`           | HTTP request method                                                                                                  | `GET`                                   |
| `$http.request.path`             | HTTP request path                                                                                                    | `/users/profile`                        |
| `$http.request.query`            | HTTP request querystring                                                                                             | `id=12345`                              |
| `$http.request.uri`              | HTTP request uri                                                                                                     | `/users/profile?id=12345`               |
| `$http.request.protocol`         | HTTP request protocol                                                                                                | `HTTP/1.1`                              |
| `$http.request.body`             | HTTP request body                                                                                                    | `{"userId": 12345, "action": "update"}` |
| `$http.request.header.<key>`     | HTTP request headers, `<key>` being the normalized HTTP Header name (lowercase), the value being the header values   | `$http.request.header.x-forwarded-for`  |
| `$http.request.query.<key>`      | HTTP request query, `<key>` being the normalized HTTP query name (lowercase), the value being the querystring values | `$http.request.header.order_id`         |
| `$http.response.size`            | The total size of the response in bytes. This should be the total number of bytes sent over the wire (unit:bye)      | `832000`                                |
| `$http.response.header.<key>`    | HTTP response headers, `<key>` being the normalized HTTP Header name (lowercase), the value being the header values  | `ab123456-7890-1234-5678-90abcdef1234`  |
| `$http.response.status_code`     | HTTP response status code                                                                                            | `200`                                   |
| `$http.request.duration`         | Duration of HTTP server requests                                                                                     | `0.250`                                 |
| `$upstream.request`              | Upstream request info                                                                                                | `GET /users/profile?id=12345 HTTP/1.1`  |
| `$upstream.request.scheme`       | Upstream request scheme                                                                                              | `http`                                  |
| `$upstream.request.host`         | Upstream request host                                                                                                | `10.0.0.50:8080`                        |
| `$upstream.request.method`       | Upstream request method                                                                                              | `GET`                                   |
| `$upstream.request.path`         | Upstream request path                                                                                                | `/users/profile`                        |
| `$upstream.request.query`        | Upstream request querystring                                                                                         | `id=12345`                              |
| `$upstream.request.uri`          | Upstream request uri                                                                                                 | `/internal/user/12345`                  |
| `$upstream.request.protocol`     | Upstream request protocol                                                                                            | `HTTP/1.1`                              |
| `$upstream.response.status_code` | Upstream response status code                                                                                        | `200`                                   |
| `$upstream.duration`             | Time taken to process the upstream request (use timecache)                                                           | `0.125`                                 |
| `$grpc.status_code`              | GRPC STATUS CODE returned by the upstream target                                                                     | `0`                                     |
| `$grpc.messaage`                 | GRPC Message returned by the upstream target                                                                         | `OK`                                    |
