# CHANGELOG

## next version

### **Features**

- Introduce `$http.request.body.json.<key>` directive, which get values from a json document in request body
- Introduce `$http.response.body.json.<key>` directive, which get values from a json document in response body
- [[#13](https://github.com/nite-coder/bifrost/pull/13)]Introduce `http_service_open_connections` metric for bifrost service

## 0.3.0 (2025-03-29)

### **Features**

- Introduce the `cors`, `parallel` middlewares.
- Introduce `set` options for `request_transformer` and `response_transformer` middlewares.
- [[#10](https://github.com/nite-coder/bifrost/pull/10)]:Introduce `nacos` config provider.
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

- [[#7](https://github.com/nite-coder/bifrost/pull/7)]: Upgraded and renamed access log directives.
- [[#8](https://github.com/nite-coder/bifrost/pull/8)]: Introduced a new `response-transformer` middleware.

## 0.1.0 (2024-12-22)

- all features are implemented and allow for previewing
