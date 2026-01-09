# AGENTS.md

## AI Interaction Guidelines & Rules

> [!IMPORTANT]
> **CRITICAL INSTRUCTIONS FOR AI AGENTS**
> 1. **Language Consistency**: Communicate in the same language as the user. If the user speaks Traditional Chinese, ALL outputs MUST be in Traditional Chinese. This includes but is not limited to:
>    - `Thinking` process and internal analysis.
>    - `TaskName`, `TaskStatus`, and `TaskSummary` in the `task_boundary` tool.
>    - All artifacts (`Task Lists`, `Implementation Plans`, `Walkthroughs`, `Report`).
>    - All tool descriptions and summaries.
> 2. **Deterministic Testing**: Prioritize using `assert.Eventually` over `time.Sleep` in unit tests. `time.Sleep` is strictly discouraged for synchronization.

### Coding Standards
- **Logging**: Use the `slog` package for all logging.
- **Type Safety**: Prefer `any` over `interface{}`.
- **Performance**: Avoid `fmt.Sprint`/`Sprintf` in hot paths; use `strconv` or direct concatenation.
- **Testing flags**: Always use `go test -race ./...` to detect race conditions.
- **Documentation**: Add comments for exported functions and structs.
- **Code Language**: Use English for all code and comments.

## Project description

Bifrost is a low-latency, high-throughput API gateway library written in Go. It is designed as an SDK/library rather than a standalone artifact, making it easy to integrate with existing Go services. Key features include:

- **High Performance**: Powered by the Hertz framework for low latency and high throughput
- **Native Go Middlewares**: Write middlewares in native Go (not TinyGo)
- **Hot Reload**: Millisecond-level graceful reload for route updates without interrupting requests
- **Protocol Support**: HTTP1.1, HTTP2, H2C, WebSocket, and gRPC
- **Load Balancing**: Multiple built-in algorithms (round_robin, random, weighted, chash) with custom balancer support
- **Observability**: Built-in Prometheus monitoring and OpenTelemetry tracing
- **Security**: Built-in web application firewall (WAF) with OWASP Core Rule Set support
- **Configuration Providers**: Support for local files, Kubernetes, etcd, and other configuration centers

## Project structure

```
bifrost/
├── client/           # Client SDK for interacting with Bifrost
├── config/           # Configuration files and examples
├── docs/             # Documentation (configuration, middlewares, providers, etc.)
├── examples/         # Example implementations and use cases
├── init/             # Initialization scripts (systemd service files, etc.)
├── internal/         # Internal packages (not for external use)
│   └── pkg/          # Internal shared packages
│       └── runtime/  # Process management (master/worker) and hot reload logic
├── pkg/              # Public packages - core library code
│   ├── balancer/     # Load balancing algorithms (round_robin, random, weighted, chash)
│   ├── config/       # Configuration parsing and management
│   ├── connector/    # Backend connection management
│   ├── gateway/      # Core gateway logic and request handling
│   ├── initialize/   # Initialization utilities
│   ├── log/          # Logging utilities (slog-based)
│   ├── middleware/   # Built-in middlewares (auth, cors, rate-limit, waf, etc.)
│   ├── provider/     # Configuration providers (file, k8s, etcd, etc.)
│   ├── proxy/        # HTTP/gRPC proxy implementation
│   ├── resolver/     # Service discovery and DNS resolution
│   ├── router/       # Request routing logic
│   ├── timecache/    # Time caching utilities
│   ├── tracer/       # Tracing implementations
│   ├── tracing/      # OpenTelemetry integration
│   └── variable/     # Variable handling for configuration
├── proto/            # Protocol buffer definitions
├── script/           # Build and utility scripts
├── server/           # Server implementations
│   ├── bifrost/      # Main Bifrost server binary
│   ├── hertz/        # Hertz-based server implementation
│   ├── openresty/    # OpenResty compatibility layer
│   ├── standard/     # Standard library-based server
│   └── testserver/   # Test server for development
└── test/             # Integration and E2E tests
```

