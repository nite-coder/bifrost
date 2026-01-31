## Context

Currently, the Bifrost client-side HTTP/2 implementation depends on `hertz-contrib/http2`, which is no longer maintained. While the server side has migrated to Stdlib, the client side needs to follow suit to unify the tech stack. The goal is to replace the old implementation using the native HTTP/2 support in Go 1.24+ `net/http`.

## Goals / Non-Goals

**Goals:**

- Implement an HTTP/2 Client Factory based on `net/http`.
- Ensure gRPC Trailers (e.g., `grpc-status`) can be correctly passed through the Proxy.
- Add integration tests targeting HTTP/2 upstreams.
- Remove the `hertz-contrib/http2` dependency.

**Non-Goals:**

- Do not change the HTTP/1.1 client implementation (still use Hertz's default netpoll implementation).

## Decisions

### 1. Use Go 1.24+ `http.Transport`

Since `http.Transport` in Go 1.24+ natively supports HTTP/2 by default (without needing manual `http2.Transport` configuration unless special adjustments are required), we will directly use `http.Transport` and configure TLS-related parameters.

- **Configuration**: `ForceAttemptHTTP2` will be set to true (default value), and ensure `TLSClientConfig` is correctly set.

### 2. ClientFactory Adapter Implementation

We will implement a new `stdlibFactory`, which needs to satisfy the Hertz `client.ClientFactory` interface and return an object that satisfies the `client.HostClient` interface.

- **Core Logic**: In the `Do` method, use `adaptor.GetCompatRequest` to convert the Hertz `protocol.Request` to an `http.Request`, execute `http.Client.Do`, and write the `http.Response` back to the Hertz `protocol.Response`.
- **Trailers**: Response Trailers need special handling, copying them from `http.Response.Trailer` back to the Hertz Response Headers marked as Trailers.

### 3. Testing Strategy

Due to the current lack of HTTP/2 upstream testing, we will add `proxy_http2_test.go` under `pkg/proxy/http`.

- **Upstream**: Use `net/http` to start an HTTP/2-only Test Server (using `httptest` + TLS).
- **Verification Points**: Generic Request/Response, Streaming Body, Trailers (gRPC simulation).

## Risks / Trade-offs

- **Performance**: The `adaptor` conversion may have slight overhead, but this is acceptable compared to the improvements in maintainability and correctness. Moreover, the server side has already verified this pattern is feasible.
- **Compatibility**: Need to ensure the `adaptor` compatibility for less common headers or special behaviors.

## Migration Plan

Directly replace the Factory references in the codebase and verify through tests. No data migration required.
