# CHANGELOG

## Next

### **Features**

- Introduce the `cors`, `parallel` middlewares.
- Introduce `set` options for `request_transformer` and `response_transformer` middlewares.
- [[#10](https://github.com/nite-coder/bifrost/pull/10)]:Introduce `nacos` config provider.
- Introduce `watch` field for file provider.
- Introduce `TestAndSkipResovler` for config package.

### **Bug Fixes**

- Fix systemd unit file.  No more `Can't open PID file` error
- Fix validation of service URL when using IP address
- Fix spelling error in `window_size` field for`rate_limit` middleware.
- Fix regex path order issue for routes.

### **Refactor**

- Redesigned the `setvars` middleware to support the `default` field.
- The `params` field of middlewares has been changed to the `any` type.
- Enhance validation of `upstream` and `service` in the config package.
- Remove `skip_test` from resolver.

### **Test**

- Improve `router`, `zero`, `gateway` `provider`, `proxy`(websocket) package test coverage.

### **Documents**

- Add "user_group" section in configuration.
- Add new `Providers` document.

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
