# Middlewares

Bifrost supports both built-in and custom middlewares that can be applied within `servers`, `routes`, and `services` configurations.

* `servers`: Middleware in this scope is executed for every HTTP request that passes through this server.
* `routes`: Middleware applied here will execute for requests matching this route.
* `services`: Middleware applied here will execute for requests matching this business service.

You can also develop custom middlewares directly in native Golang.

## Built-In Middlewares

Currently supported middlewares are below.

* [AddPrefix](#addprefix): Add a prefix to the request path.
* [Buffering](#buffering): Buffer the request body and enforce maximum size.
* [Compression](#compression): Compress the response body using gzip.
* [Coraza](#coraza): A Web application firewall.
* [Cors](#cors): A Middleware for Cross-Origin Resource Sharing.
* [IPRestriction](#iprestriction): Control client IP address that can access the service.
* [Mirror](#mirror): Mirror the request to another service.
* [Parallel](#parallel): Execute a group of middlewares concurrently.
* [RateLimit](#ratelimit): To control the Number of Requests going to a service
* [ReplacePath](#replacepath): Replace the request path.
* [ReplacePathRegex](#replacepathregex): Replace the request path with a regular expression.
* [RequestTermination](#requesttermination): Response the content to client and terminate the request.
* [RequestTransformer](#requesttransformer): Apply a request transformation to the request.
* [ResponseTransformer](#responsetransformer): Apply a response transformation to the response.
* [SetVars](#setvars): Set variables in the request context.
* [StripPrefix](#stripprefix): Remove a prefix from the request path.
* [TrafficSplitter](#trafficsplitter): Route requests to different services based on weights.
* [UARestriction](#uarestriction): Control user agent that can access the service.

### AddPrefix

Adds a prefix to the original request path before forwarding upstream.

Original request: `/foo` \
Forwarded path for upstream: `/api/v1/foo`

```yaml
routes:
  foo:
    paths:
      - /foo
    service_id: service1
    middlewares:
      - type: add_prefix
        params:
          prefix: /api/v1
```

params:

| Field  | Type     | Default | Description                    |
| ------ | -------- | ------- | ------------------------------ |
| prefix | `string` |         | Add prefix to the request path |

### Buffering

The `buffering` middleware is used to read the entire request body into memory before forwarding it to the upstream service. This is useful for:

1. Ensuring the upstream receives the request only after the full body is received (preventing slow body attacks).
2. Enforcing a maximum request body size early in the request lifecycle.
3. Enabling `Content-Length` forwarding instead of `Transfer-Encoding: chunked` for upstreams that do not support chunking.

```yaml
routes:
  foo:
    paths:
      - /upload
    service_id: upload_service
    middlewares:
      - type: buffering
        params:
          max_request_body_size: 10485760 # 10MB
```

params:

| Field                 | Type    | Default | Description                                                                 |
| --------------------- | ------- | ------- | --------------------------------------------------------------------------- |
| max_request_body_size | `int64` | 4194304 | Maximum number of bytes for the request body. Returns 413 if exceeded. (4MB) |

### Compression

Compresses the response body using gzip. This middleware is useful for reducing the size of the response payload, improving load times, and saving bandwidth.

```yaml
routes:
  foo:
    paths:
      - /foo
    service_id: service1
    middlewares:
      - type: compression
        params:
          level: 6
          excluded_paths: ["/excluded"]
```

params:

| Field          | Type       | Default | Description                                                |
| -------------- | ---------- | ------- | ---------------------------------------------------------- |
| level          | `int`      | 6       | Compression level from 1 (fastest) to 9 (best compression) |
| excluded_paths | `[]string` |         | Paths that should not be compressed                        |

### Coraza

Coraza is an open source, enterprise-grade, high performance Web Application Firewall (WAF) ready to protect your beloved applications. It is written in Go, supports ModSecurity SecLang rulesets and is 100% compatible with the OWASP Core Rule Set v4.

```yaml
routes:
  foo:
    paths:
      - /foo
    service_id: service1
    middlewares:
      - type: coraza
        params:
          directives: |
            Include @coraza.conf-recommended
            Include @crs-setup.conf.example
            Include @owasp_crs/*.conf
            SecRuleEngine On
          rejected_http_status_code: 403
          rejected_http_content_type: application/json
          rejected_http_response_body: "forbidden by waf"
```

params:

| Field                       | Type       | Default | Description                      |
| --------------------------- | ---------- | ------- | -------------------------------- |
| directives                  | `string`   |         | Coraza directives                |
| ip_allow_list               | `[]string` |         | IP allow list                    |
| rejected_http_status_code   | `int`      |         | The status code of the response  |
| rejected_http_content_type  | `string`   |         | The content type of the response |
| rejected_http_response_body | `string`   |         | The body of the response         |

### Cors

A Middleware for Cross-Origin Resource Sharing (CORS). This middleware allows you to control how resources on your server are shared with external domains. It is particularly useful for enabling secure cross-origin requests in web applications.

```yaml
routes:
  foo:
    paths:
      - /foo
    service_id: service1
    middlewares:
      - type: cors
        params:
          allow_all_origins: true
          allow_methods:
            ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"]
          allow_headers: ["Origin", "Content-Length", "Content-Type"]
          max_age: 12m
```

params:

| Field             | Type       | Default | Description                                                           |
| ----------------- | ---------- | ------- | --------------------------------------------------------------------- |
| allow_all_origins | `bool`     | `false` | If `true`, allows requests from all origins.                          |
| allow_origins     | `[]string` |         | A list of allowed origins. Cannot be used with `allow_all_origins`.   |
| allow_methods     | `[]string` |         | A list of HTTP methods allowed for cross-origin requests.             |
| allow_headers     | `[]string` |         | A list of HTTP headers allowed in cross-origin requests.              |
| expose_headers    | `[]string` |         | A list of headers exposed to the client in the response.              |
| allow_credentials | `bool`     | `false` | If `true`, allows credentials (cookies, authorization headers, etc.). |
| max_age           | `Duration` |         | The maximum time a preflight request can be cached by the client.     |

### IPRestriction

Control client IP address that can access the service.  Either one of `allow` or `deny` attribute must be specified. They cannot be used together.

```yaml
routes:
  foo:
    paths:
      - /foo
    service_id: service1
    middlewares:
      - type: ip_restriction
        params:
          allow: ["192.168.1.1"] # allow and deny can't be used at the same time
          deny: ["192.168.1.0/24"]
          rejected_http_status_code: 403
          rejected_http_content_type: application/json
          rejected_http_response_body: "forbidden"
```

params:

| Field                       | Type       | Default | Description                               |
| --------------------------- | ---------- | ------- | ----------------------------------------- |
| allow                       | `[]string` |         | IP allow list                             |
| deny                        | `[]string` |         | IP deny list                              |
| rejected_http_status_code   | `int`      |         | The status code of the rejected response  |
| rejected_http_content_type  | `string`   |         | The content type of the rejected response |
| rejected_http_response_body | `string`   |         | The body of the rejected response         |

### Mirror

Mirrors the request to another service. This middleware duplicates the incoming request and sends it to a secondary service (`service2`) while continuing to process the original request with the primary service (`service1`). The mirrored request does not affect the response returned to the client.

This is useful for scenarios like testing new services, logging, or analytics without impacting the primary service.

```yaml
routes:
  foo:
    paths:
      - /foo
    service_id: service1
    middlewares:
      - type: mirror
        params:
          service_id: service2
```

params:

| Field      | Type     | Required | Description                                                             |
| ---------- | -------- | -------- | ----------------------------------------------------------------------- |
| service_id | `string` | ✅       | The ID of the service to which the request will be mirrored.            |
| queue_size | `int`    | ❌       | The maximun size of the queue. If not set, the default value is `10000` |

### Parallel

Executes a group of middlewares concurrently. This middleware is useful for optimizing performance by running multiple middlewares in parallel. If any middleware in the group encounters an error, the request will be terminated immediately.

```yaml
routes:
  order:
    paths:
      - /order
    service_id: order_service
    middlewares:
      - type: parallel
        params:
          - type: setvars
            params:
              - key: user_id
                value: $http.request.header.user_id
          - type: setvars
            params:
              - key: is_vip
                value: true
```

### RateLimit

The `RateLimit` middleware ensures that services receive a fair share of requests and allows you to define what "fair" means.
The following example allows a user to send a maximum of 10 requests within 2 seconds.  If redis server is crashed, the requests will be passed. (downgrade)

```yaml
routes:
  order:
    paths:
      - /orders
    middlewares:
      - type: rate_limit
        params:
          strategy: local # local, redis
          limit_by: user_id:$var.user_id  # allow to use directive
          limit: 10
          window_size: 2s
          header_limit: x-ratelimit-limit
          header_remaining: x-ratelimit-remaining
          header_reset: x-ratelimit-reset
          rejected_http_status_code: 429 # when hit the rate limit
          rejected_http_content_type: application/json
          rejected_http_response_body: {"error": "too many requests"}
    service_id: order_service
```

params:

| Field                       | Type       | Default | Description                                                                               |
| --------------------------- | ---------- | ------- | ----------------------------------------------------------------------------------------- |
| strategy                    | `string`   |         | The strategy of the rate limit.  The value can be `local` or `redis`                      |
| limit_by                    | `string`   |         | The key of the rate limit                                                                 |
| limit                       | `int`      |         | The limit of the rate limit                                                               |
| window_size                 | `Duration` |         | The window size of the rate limit                                                         |
| header_limit                | `string`   |         | The name of the custom header used to indicate the limit number of rate limit             |
| header_remaining            | `string`   |         | The name of the custom header used to indicate the remaining number of allowed requests   |
| header_reset                | `string`   |         | The name of the custom header used to indicate the timestamp of the end of the rate limit |
| rejected_http_status_code   | `int`      |         | The status code of the rejected response                                                  |
| rejected_http_content_type  | `string`   |         | The content type of the rejected response                                                 |
| rejected_http_response_body | `string`   |         | The body of the rejected response                                                         |

### ReplacePath

Replaces the entire original request path with a different path before forwarding upstream. If the original request includes a query string, it will also be forwarded.

Original request: `/api/v1/user?name=john` \
Forwarded path for upstream: `/hoo/user?name=john`

```yaml
routes:
  user:
    paths:
      - /api/v1/user
    service_id: service1
    middlewares:
      - type: replace_path
        params:
          path: /hoo/user
```

params:

| Field | Type     | Default | Description                                          |
| ----- | -------- | ------- | ---------------------------------------------------- |
| path  | `string` |         | The path that will replace the original request path |

### ReplacePathRegex

Replaces the entire original request path using a regular expression before forwarding the request to the upstream service. If the original request includes a query string, it will be preserved.

Original request: `/api/v1/user?name=john` \
Forwarded request: `/hoo/user?name=john`

```yaml
routes:
  user:
    paths:
      - /api/v1/user
    service_id: service1
    middlewares:
      - type: replace_path_regex
        params:
          regex: ^/api/v1/(.*)$
          replacement: /hoo/$1
```

params:

| Field       | Type     | Default | Description                                                                        |
| ----------- | -------- | ------- | ---------------------------------------------------------------------------------- |
| regex       | `string` |         | Regular expression used to match and capture parts of the original request path.   |
| replacement | `string` |         | The replacement path pattern. Supports regex capture groups from the `regex` field |

### RequestTermination

Terminates the request early and immediately returns a custom response to the client. This is useful for mocking endpoints or short-circuiting requests before they reach the upstream service.

```yaml
routes:
  mock:
    paths:
      - /mock/order
    service_id: service1
    middlewares:
      - type: request_termination
        params:
          status_code: 200
          content_type: application/json
          body: "Hello World"

```

params:

| Field        | Type     | Default | Description                                                              |
| ------------ | -------- | ------- | ------------------------------------------------------------------------ |
| status_code  | `int`    |         | The HTTP status code to return to the client (e.g., 200, 403, 500).      |
| content_type | `string` |         | The MIME type of the response (e.g., `application/json`, `text/plain`).  |
| body         | `string` |         | The body content to return in the response. Supports plain text or JSON. |

### RequestTransformer

Applies transformations to the incoming request before forwarding it to the upstream service. You can remove, add, or set headers and query strings.

* `remove` - option deletes specified headers or query string parameters.
* `add` - option adds the specified values only if they don’t already exist.
* `set` - option adds or overwrites existing values, regardless of whether they previously existed.

```yaml
routes:
  mock:
    paths:
      - /mock/order
    service_id: service1
    middlewares:
      - type: request_transformer
        params:
          remove:
            headers:
              - x-user-id
            querystring:
              - mode
          add:
            headers:
              x-id: 123
          set:
            headers:
              x-source: "web"
```

params:

| Field                | Type                | Default | Description                                                     |
| -------------------- | ------------------- | ------- | --------------------------------------------------------------- |
| `remove.headers`     | `[]string`          |         | A list of request header names to remove.                       |
| `remove.querystring` | `[]string`          |         | A list of query string parameter names to remove.               |
| `add.headers`        | `map[string]string` |         | Adds headers only if they do not already exist.                 |
| `add.querystring`    | `map[string]string` |         | Adds query string parameters only if they do not already exist. |
| `set.headers`        | `map[string]string` |         | Sets (adds or overwrites) request headers.                      |
| `set.querystring`    | `map[string]string` |         | Sets (adds or overwrites) query string parameters.              |

### ResponseTransformer

Applies transformations to the outgoing response before it is returned to the client. This middleware allows you to manipulate response headers using three options:

* `remove` – Removes specified headers from the response.
* `add` – Adds headers only if they do not already exist.
* `set` – Adds headers, and overwrites existing ones if they already exist.

You can also use dynamic variables (e.g., $http.start, $http.finish) in values for enhanced traceability or debugging.

```yaml
routes:
  mock:
    paths:
      - /mock/order
    service_id: service1
    middlewares:
      - type: response_transformer
        params:
          remove:
            headers:
              - x-server
          add:
            headers:
              x-source: web
              x-http-start: $http.start
              x-http-finish: $http.finish
          set:
            headers:
              x-source: "web"
```

| Field            | Type                | Default | Description                                              |
| ---------------- | ------------------- | ------- | -------------------------------------------------------- |
| `remove.headers` | `string[]`          |         | A list of response header names to remove.               |
| `add.headers`    | `map[string]string` |         | Adds response headers only if they do not already exist. |
| `set.headers`    | `map[string]string` |         | Sets (adds or overwrites) response headers.              |

### SetVars

Sets custom variables into the request context. These variables can be referenced later in the routing pipeline, such as in transformations, logging, or upstream requests.

This middleware supports:

* Static values
* Dynamic directives (e.g., $header.x-user-id, $query.order_id)
* Fallback defaults – if the value is missing or not resolvable, it will fall back to the default.

```yaml
routes:
  orders:
    paths:
      - /api/v1/orders
    service_id: service1
    middlewares:
      - type: setvars
        params:
          - key: user_id
            value: "$header.x-user-id"   # Uses value from request header
            default: "0"                 # Default if header is missing
          - key: order_mode
            value: "standard"            # Static value
```

In this example:

* `user_id` will be set to the value of `x-user-id` from the request header, or `"0"` if it’s missing.
* `order_mode` is set statically to `"standard"`.

| Field     | Type     | Required | Description                                                                |
| --------- | -------- | -------- | -------------------------------------------------------------------------- |
| `key`     | `string` | ✅       | The name of the variable to set in the request context.                    |
| `value`   | `string` | ✅       | The value to assign. Can be a static string or a directive (e.g., `$...`). |
| `default` | `string` | ❌       | Optional fallback value if the `value` cannot be resolved.                 |

### StripPrefix

Strips one or more matching prefixes from the original request path before forwarding the request to the upstream service.

This is useful for decoupling the public API structure from the internal upstream structure. For example, if your public endpoint includes a versioned prefix like /api/v1, but your upstream only expects /payment, this middleware helps transform the path automatically.

Original request: `/api/v1/payment` \
Forwarded path for upstream: `/payment`

```yaml
routes:
  payment:
    paths:
      - /api/v1/payment
    service_id: service1
    middlewares:
      - type: strip_prefix
        params:
          prefixes:
            - /api/v1
```

params:

| Field      | Type       | Required | Description                                                                                                         |
| ---------- | ---------- | -------- | ------------------------------------------------------------------------------------------------------------------- |
| `prefixes` | `[]string` | ✅       | A list of path prefixes to strip from the beginning of the request path. Only the first matching prefix is removed. |

### TrafficSplitter

Route requests to different services based on weights. This middleware allows you to split traffic between multiple services based on predefined weights. It is particularly useful for scenarios like gradual rollouts, A/B testing, or canary deployments.

```yaml
servers:
  apiv1:
    bind: ":8001"
    reuse_port: true
    middlewares:
      - type: traffic_splitter
        params:
          key: $my_order
          destinations:
            - weight: 80
              to: old_service
            - weight: 20
              to: new_service
```

params:

| Field                 | Type       | Required | Description                                                                   |
| --------------------- | ---------- | -------- | ----------------------------------------------------------------------------- |
| `key`                 | `string`   | ✅       | The key used to determine how traffic is split. Can use dynamic variables.    |
| `destinations`        | `[]object` | ✅       | A list of destinations with their respective weights.                         |
| `destinations.weight` | `int`      | ✅       | The weight assigned to this destination. Higher weights receive more traffic. |
| `destinations.to`     | `string`   | ✅       | The service ID to which the traffic will be routed.                           |

### UARestriction

Controls which user agents can access the service. This middleware allows you to specify either an `allow` list or a `deny` list of user agents. You can use regular expressions to match user agents. Note that `allow` and `deny` cannot be used together.

```yaml
routes:
  foo:
    paths:
      - /foo
    service_id: service1
    middlewares:
      - type: ua_restriction
        params:
          allow: ["good-agent.*"] # allow and deny can't be used at the same time
          rejected_http_status_code: 403
          rejected_http_content_type: application/json
          rejected_http_response_body: "forbidden"
```

params:

| Field                         | Type       | Required | Description                                                                 |
| ----------------------------- | ---------- | -------- | --------------------------------------------------------------------------- |
| `allow`                       | `[]string` | ❌       | A list of allowed user agents (supports regex). Cannot be used with `deny`. |
| `deny`                        | `[]string` | ❌       | A list of denied user agents (supports regex). Cannot be used with `allow`. |
| `rejected_http_status_code`   | `int`      | ✅       | The HTTP status code to return for rejected requests.                       |
| `rejected_http_content_type`  | `string`   | ✅       | The content type of the rejected response.                                  |
| `rejected_http_response_body` | `string`   | ✅       | The body of the rejected response.                                          |
