# CHANGELOG

## Next

- refactor: enchance dns resolver to update upstreams.
- refactor: don't read subdir in paths when file provider is used
- refactor: optimize the information reported by OpenTelemetry tracing and redesign config.
- refactor: metric endoint follows semantic-conventions
- chore: upgrade hertz package to v0.9.5
- feat: load config from default config path when config path is empty.
- feat: support systemd init script.
- fix: order of routes issue
- fix: promethus server name is unkonwn issue
- fix: validate promethus serrver_id in config
- fix: allow tracing endpoint with ip
- fix: config validation is not correct
- fix: access log tracer is added to server more than once
- [[#7](https://github.com/nite-coder/bifrost/pull/7)]: upgrade and rename access log directives
- [[#8](https://github.com/nite-coder/bifrost/pull/8)]: new response-transformer middleware

## 0.1.0 (2024-12-22)

- all features are implemented and allow for previewing
