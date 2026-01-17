# Project Context

## Purpose

Bifrost is a low-latency, high-throughput API gateway library written in Go. It is designed as an SDK/library rather than a standalone artifact, making it easy to integrate with existing Go services.

Key features:

- **High Performance**: Powered by the Hertz framework for low latency and high throughput
- **Native Go Middlewares**: Write middlewares in native Go (not TinyGo)
- **Hot Reload**: Millisecond-level graceful reload for route updates without interrupting requests
- **Protocol Support**: HTTP1.1, HTTP2, H2C, WebSocket, and gRPC
- **Load Balancing**: Multiple built-in algorithms (round_robin, random, weighted, chash) with custom balancer support
- **Observability**: Built-in Prometheus monitoring and OpenTelemetry tracing
- **Security**: Built-in Web Application Firewall (WAF) with OWASP Core Rule Set support
- **Configuration Providers**: Support for local files, Kubernetes, etcd, and other configuration centers

## Tech Stack

- **Language**: Go 1.24+
- **HTTP Framework**: Hertz (CloudWeGo)
- **gRPC**: google.golang.org/grpc
- **Configuration**: YAML, Viper
- **Monitoring**: Prometheus client_golang
- **Tracing**: OpenTelemetry
- **Testing**: testify (assert/require), testcontainers
- **Linting**: golangci-lint v2
- **WAF**: Coraza + OWASP Core Rule Set

## Project Conventions

### Code Style

- **Logging**: Use the `slog` package for all logging
- **Type Safety**: Prefer `any` over `interface{}`
- **Performance**: Avoid `fmt.Sprint`/`Sprintf` in hot paths; use `strconv` or direct concatenation
- **Code Language**: Use English for all code and comments
- **Documentation**: Add comments for exported functions and structs

### Architecture Patterns

- **SDK Design**: Built as a library, not a standalone application
- **Middleware Pattern**: Use `RegisterTyped` for type-safe middleware registration
- **Configuration-Driven**: Manage routes, services, upstreams via YAML configuration files
- **Multi-Process Architecture**: Master/Worker process model with graceful hot reload support

### Testing Strategy

- **Testing Framework**: Use `testify` package (`require` for critical assertions, `assert` for non-critical ones)
- **Deterministic Testing**: Prioritize `assert.Eventually` over `time.Sleep` for synchronization
- **Race Detection**: Always use `go test -race ./...` to detect race conditions
- **Coverage**: Generate coverage reports via `make coverage`
- **Integration Tests**: Use testcontainers for external dependency testing

### Git Workflow

- **Release Process**: `make release` (includes build, lint, test, e2e-test)
- **Changelog**: Maintain CHANGELOG.md for all significant changes
- **CI/CD**: GitHub Actions for automated builds and tests

## Domain Context

- **Servers**: Server configuration, supports middlewares, controls which port to expose
- **Routes**: Route configuration, controls request path forwarding rules to specific services
- **Services**: Service configuration, controls service details such as protocol information
- **Upstreams**: Upstream configuration, manages load balancing rules for backend hosts

## Important Constraints

- Prioritize performance and low latency
- Maintain similarity with Nginx configuration concepts for easy migration
- Middlewares must be pluggable and extensible
- Support graceful reload without interruption

## External Dependencies

- **Hertz**: CloudWeGo's high-performance HTTP framework (using nite-coder fork)
- **Prometheus**: Monitoring metrics collection
- **OpenTelemetry**: Distributed tracing
- **Kubernetes API**: Service discovery and configuration management
- **etcd/Nacos**: Configuration center integration
- **Redis**: Caching and rate limiting (optional)
