# Design - Migrate HTTP/2 Server to Go Stdlib

## Context

Bifrost is a high-performance API Gateway using Hertz as the HTTP/1 server. It currently supports HTTP/2 through `hertz-contrib/http2` (fork version `nite-coder/http2`), but this package is no longer maintained.

**Current Status:**

- Hertz + netpoll provide extremely low latency for HTTP/1 processing.
- HTTP/2 depends on an unmaintained fork.
- Go 1.25.4 (Go 1.24+) has built-in native HTTP/2 support.

## Goals / Non-Goals

**Goals:**

- Remove `hertz-contrib/http2` dependency and use the Go standard library.
- Maintain zero extra overhead for HTTP/1 only mode.
- Use the uniform Hertz handler chain (all middleware remains the same).
- Scalable architecture to support HTTP/3.

**Non-Goals:**

- Do not implement HTTP/3 in this change (only ensure the architecture is extensible).
- Do not change the user configuration format.
- Do not modify core Hertz code.
- **Do not modify Hertz Client HTTP/2** - The client side continues to use `hertz-contrib/http2`.

## Decisions

### 1. Hybrid Server Architecture

**Decision**: Dynamically select the server type based on configuration.

| Configuration | Server | Reason |
|---------------|--------|--------|
| `http2: false` | Hertz | Maintain optimal HTTP/1 performance |
| `http2: true` | Go stdlib | Use native HTTP/2 support |

**Alternatives Considered**:

- ❌ Use Go stdlib for everything: HTTP/1 would lose the Hertz/netpoll performance advantage.
- ❌ Maintain the http2 fork: High long-term maintenance cost.

### 2. Bridge Implementation

**Decision**: Use `adaptor.CopyToHertzRequest` + custom response write-back.

```go
func (b *hertzBridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    reqCtx := pool.Get().(*app.RequestContext)
    defer pool.Put(reqCtx)
    
    adaptor.CopyToHertzRequest(r, &reqCtx.Request)  // Official adaptor
    engine.ServeHTTP(ctx, reqCtx)                   // Hertz handler chain
    writeResponse(w, reqCtx)                        // Custom write-back
}
```

**Reason**:

- Use the official adaptor for the request direction to reduce maintenance.
- Custom implementation needed for the response direction (Hertz adaptor lacks a reverse function).

### 3. HTTP/2 Protocol Configuration

**Decision**: Use Go 1.24+ `http.Server.Protocols`.

```go
var protocols http.Protocols
protocols.SetHTTP1(true)
protocols.SetHTTP2(true)
protocols.SetUnencryptedHTTP2(true)  // h2c

server := &http.Server{
    Protocols: &protocols,
    HTTP2:     &http.HTTP2Config{...},
}
```

**Reason**: Pure standard library, no need for `golang.org/x/net/http2`.

## Risks / Trade-offs

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| HTTP/2 bridging latency (+30-60μs) | Medium | Only affects HTTP/2 mode; no impact on HTTP/1 users. |
| Response write-back logic needs manual maintenance | Low | Simple logic, just header copying + body writing. |
| Breaking change for http2 users | Medium | Document migration steps. |
| gRPC Proxy compatibility | High | Must verify through tests that gRPC requests correctly pass through the bridge. |

## Verification Requirements

### gRPC Proxy Testing

Must verify the following scenarios:

1. **gRPC Unary Request**: Single request/response (Streaming not supported).
2. **gRPC Headers**: Ensure `content-type: application/grpc` is correctly passed.
3. **gRPC Trailers**: Ensure trailer headers (`grpc-status`, `grpc-message`) are correctly handled.
4. **Existing Tests**: `pkg/proxy/grpc/proxy_test.go` must all pass.

## Architecture

```
          ┌─────────────┐
          │ Config      │
          │ http2: ?    │
          └──────┬──────┘
                 │
      ┌──────────┴──────────┐
      │ http2: false        │ http2: true
      ▼                     ▼
 ┌─────────┐         ┌────────────────┐
 │ Hertz   │         │ net/http.Server│
 │ Server  │         │ (HTTP/1+HTTP/2)│
 │         │         └───────┬────────┘
 │         │                 │
 │ Engine ◄├─────────────────┤ hertzBridge
 │         │                 │
 └─────────┘         ┌───────┴────────┐
                     │ adaptor.Copy   │
                     │ ToHertzRequest │
                     └────────────────┘
```

## Impact Analysis

### gRPC Proxy ✅ Unaffected

The gRPC Proxy (`pkg/proxy/grpc/proxy.go`) uses the `google.golang.org/grpc` client to connect upstream:

```go
client, err := grpc.NewClient(addr.Host, grpcOptions...)
```

**Workflow**:

1. HTTP/2 Server receives a gRPC request.
2. The bridge converts the `http.Request` to a Hertz `RequestContext`.
3. The gRPC Proxy handler extracts gRPC data from the `RequestContext`.
4. Forwarded to upstream using `grpc.Client`.

Conclusion: As long as the bridge correctly passes HTTP/2 headers (including the gRPC-specific `content-type: application/grpc`), the gRPC Proxy requires no modification.

### Hertz Client HTTP/2 ⚠️ Remains Unchanged

The HTTP Client (`pkg/proxy/http/client.go`) sending upstream still uses `hertz-contrib/http2`:

```go
c.SetClientFactory(factory.NewClientFactory(http2Config.WithAllowHTTP(true)))
```

**Decision**: In this change (Phase 1), only the server side is migrated; the client side continues to use `hertz-contrib/http2`.

**Phase 2 Plan**: Create a custom HTTP/2 ClientFactory.

```go
// Future implementation direction
type StdlibHTTP2ClientFactory struct {}

func (f *StdlibHTTP2ClientFactory) NewHostClient() (client.HostClient, error) {
    // Use Go stdlib http.Transport + HTTP/2
    // Wrap into Hertz HostClient interface
}
```

**Reason**:

- Separation of concerns, reducing risk.
- Client requirements differ from server requirements.
- Maintain the Hertz Client performance advantage, only replacing the HTTP/2 protocol layer.

## Open Questions

1. ~~Is h2c support required?~~ → Yes, via `SetUnencryptedHTTP2(true)`.
2. Should HTTP/2 parameters be exposed to users? → Recommended to expose basic parameters like `max_concurrent_streams`.
3. ~~How to handle Client HTTP/2?~~ → Phase 2 will create a custom HTTP/2 ClientFactory.
