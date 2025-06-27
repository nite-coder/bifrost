# CHANGELOG

## 0.5.0 (2025-06-28)

### **Features**

- Introduce PROXY protocol support for the server ([#33](https://github.com/nite-coder/bifrost/pull/33))
- Introduce `Coraza` web application firewall middleware ([#36](https://github.com/nite-coder/bifrost/pull/36))
- Introduce `k8s`service discovery provider ([#37](https://github.com/nite-coder/bifrost/pull/37))
- Introduce `$http.request.cookie.<key>` directive ([#46](https://github.com/nite-coder/bifrost/pull/46))
- Introduce `pass_host_header` service optoins (default: `true`). Allow to forward client `host` header to upstream ([#48](https://github.com/nite-coder/bifrost/pull/48))

### **Bug Fixes**

- Fixed a race condition caused by the reuse of app.RequestContext, which typically occurred when one or more upstream targets were unavailable. ([#32](https://github.com/nite-coder/bifrost/pull/32))
- Fixed a case sensitive issue in `variable` package ([#35](https://github.com/nite-coder/bifrost/pull/35))

### **Refactor**

- rename `redirect_stderr` to `disable_redirect_stderr` and allow to redirect stderr by default ([#39](https://github.com/nite-coder/bifrost/pull/39))
- improve `variable.HTTPRequestMethod` performance ([#41](https://github.com/nite-coder/bifrost/pull/41))
- struct alignment for better performance and lower memory usage ([#42](https://github.com/nite-coder/bifrost/pull/42))
- improve client cancel request when context is Canceled ([#45](https://github.com/nite-coder/bifrost/pull/45))
- allow to forward client `host` header to upstream ([#47](https://github.com/nite-coder/bifrost/pull/47))
- rename `num_loops` to `event_loops` ([#50](https://github.com/nite-coder/bifrost/pull/50))

### **Test**

- Improve `router` package test coverage. ([#34](https://github.com/nite-coder/bifrost/pull/34))

### **Documents**

- Add `coraza` middleware document ([#36](https://github.com/nite-coder/bifrost/pull/36))
- Add description for middleware document ([#40](https://github.com/nite-coder/bifrost/pull/40))
- Add `$server_id` directive

### **Chore**

- Bump OpenTelemetry package to `v1.36.0` ([#43](https://github.com/nite-coder/bifrost/pull/43))
- Bump Hertz package to `v0.10.0` ([#44](https://github.com/nite-coder/bifrost/pull/44))

## 0.4.0 (2025-05-14)

### **Features**

- Introduce `$http.request.body.json.<key>` directive, which get values from a json document in request body
- Introduce `$http.response.body.json.<key>` directive, which get values from a json document in response body
- Introduce `http_service_open_connections` metric for bifrost service in [[#13](https://github.com/nite-coder/bifrost/pull/13)]
- Introduce `http_server_open_connections` metric for bifrost server in [[#14](https://github.com/nite-coder/bifrost/pull/14)]
- Introduce `$upstream.conn_acquisition_time` directive, which indicates the time taken to get the upstream connection in [[#15](https://github.com/nite-coder/bifrost/pull/15)]
- Introduce service discovery provider in [[#18](https://github.com/nite-coder/bifrost/pull/18)]
- Introduce `dns` service discovery provider in [[#19](https://github.com/nite-coder/bifrost/pull/19)]
- Introduce `nacos` service discovery provider in [[#26](https://github.com/nite-coder/bifrost/pull/26)]
- Add `Set` function to redis connector
- Introduce `ip_restriction` middleware in [[#22](https://github.com/nite-coder/bifrost/pull/22)]
- Introduce `ua_restriction` middleware in [[#23](https://github.com/nite-coder/bifrost/pull/23)]
- Introduce `tags` for routes in [[#24](https://github.com/nite-coder/bifrost/pull/24)]
- Introduce `redirect_stderr` field for logging
- Experiment: Introduce `chunked_transfer` option

### **Bug Fixes**

- When the service encounters a `no free connections available to host` error, return HTTP status code `500` instead of `502`
- During the upgrade process, existing connections will be gracefully closed.
- `$http.start` and `$http.finish` need to be unix micro time when working with `ResponseTransformer` middleware

### **Refactor**

- move `fail_timeout` and `max_fails` from target to upstream level (passive healthcheck) in [[#17](https://github.com/nite-coder/bifrost/pull/17)]
- change `resolver` to support multiple dns servers, order, and hostfile, timeout in [[#18](https://github.com/nite-coder/bifrost/pull/18)] [[#20](https://github.com/nite-coder/bifrost/pull/20)]
- rename `http_status_code`, `http_content_type`, `http_response_body` to `rejected_http_status_code`, `rejected_http_content_type`, `rejected_http_response_body` for `rate_limit` middleware in [[#21](https://github.com/nite-coder/bifrost/pull/21)]
- Use `safety.Go` to run goroutine.
- Stop logging errors when the upstream is unavailable (returns HTTP 503)
- redesign redis rate limit middleware and improve performance

### **Test**

- Improve `middlware(mirror)` package test coverage.

### **Chore**

- Bump Hertz package to `v0.9.7`

## 0.3.0 (2025-03-29)

### **Features**

- Introduce the `cors`, `parallel` middlewares.
- Introduce `set` options for `request_transformer` and `response_transformer` middlewares.
- [[#10](https://github.com/nite-coder/bifrost/pull/10)] Introduce `nacos` config provider.
- Introduce `watch` field for file provider.
- Introduce `TestAndSkipResovler` function for config package.
- Introduce `Default` object for configuration file.
- Introduce `trusted_cidrs` and `remote_ip_headers` field for servers options.
- Introduce the `connector` package, which manages third-party connections and can be utilized by middlewares.
- Introduce the `$env` directive, which allow to get value from environment variable.

### **Bug Fixes**

- Fix systemd unit file.  No more `Can't open PID file` error
- Fix validation of service URL when using IP address
- Fix spelling error in `window_size` field for`rate_limit` middleware.
- Fix regex path order issue for routes.
- Fix accuracy of `http_bifrost_request_duration` metric.
- Fix dynamic service has been executed many times.
- Fix multiple server with the same route path.
- Fix goroutine leak in service request.
- Fix issue where panic recovery does not return HTTP status 500
- Fix FD leak issue

### **Refactor**

- Redesigned the `setvars` middleware to support the `default` field.
- The `params` field of middlewares has been changed to the `any` type.
- Enhance validation of `upstreams`, `middlewares`, `services` in the config package.
- Remove `skip_test` from resolver.
- Require `root` permissions to run as daemon.
- Redesign resolver system to support multiple resolvers.
- rename `statusCode` to `status_code` and add `service_id`, `route_id` in http_server_requests metric.

### **Test**

- Improve `router`, `zero`, `gateway`, `config`, `tracer(prometheus)` `provider`, `proxy(websocket)` package test coverage.

### **Documents**

- Add `user_group`, `gopool` objects in configuration.
- Add new `Providers` document.

### **Chore**

- Bump Hertz package to `v0.9.6`
- Bump Prometheus package to `v1.21.1`

## 0.2.0 (2024-02-14)

### **Features**

- Automatically load configuration from the default path when the config path is empty.
- Added support for systemd init scripts.
- Enabled handling of `SIGUSR1` signals for logging purposes.
- Introduce the `route` field in routes.
- Introduce the `observability` field in servers options.
- Introduce the `$error.type`, `$error.message`, `$trace_id`, `$http.request.query.<key>` directive.
- `Addprefix`,`Replacepath` and `Setvars` middlewares can use directive
- Add additional validations when running the -t flag via CLI.

### **Bug Fixes**

- Resolved issues with the order of routes.
- Fixed an unknown server name issue in Prometheus.
- Ensured validation of `server_id` in Prometheus configuration.
- Allowed tracing endpoints using IP addresses.
- Corrected inaccurate configuration validation.
- Prevented access log tracer from being added to the server multiple times.
- Added support for trailing slashes in paths.
- Corrected the order of regex routes when methods are used.
- Ensured `response_transformer` execution after requests
- the server id is missing when validating middleware in server options

### **Refactor**

- Enhanced DNS resolver to update upstreams.
- Excluded reading subdirectories in paths when using the file provider.
- the `tracing` middleware is removed.
- Updated metric endpoint to follow semantic conventions.
- Allow the `hash_on` field in upstream to support all directives.
- Move router from `gateway` package to `router` package

### **Chore**

- Updated Hertz package to `v0.9.5`
- Updated the minimum required Go version to `v1.23`

### **Pull Requests**

- [[#7](https://github.com/nite-coder/bifrost/pull/7)] Upgraded and renamed access log directives.
- [[#8](https://github.com/nite-coder/bifrost/pull/8)] Introduced a new `response-transformer` middleware.

## 0.1.0 (2024-12-22)

- all features are implemented and allow for previewing
