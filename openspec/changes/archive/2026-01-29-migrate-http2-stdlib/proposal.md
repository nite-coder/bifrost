# Proposal - Migrate HTTP/2 Server to Go Stdlib

## Why

The `hertz-contrib/http2` package is no longer maintained. Bifrost currently uses a fork (`nite-coder/http2`), but long-term dependence on unmaintained packages brings security and compatibility risks. Go 1.24+ standard library natively supports HTTP/2 (including h2c), providing a complete replacement for the existing solution.

## What Changes

- **Phase 1 (Current)**: Server-side HTTP/2 migration to Go stdlib
- **Phase 2 (Future)**: Client-side HTTP/2 migration - create custom HTTP/2 ClientFactory
- New hybrid Server architecture:
  - `http2: false` → Use Hertz Server (zero latency overhead)
  - `http2: true` → Use Go stdlib `net/http.Server` + Hertz Bridge
- Use `adaptor.CopyToHertzRequest` for request conversion
- Uniformly use Hertz `app.HandlerFunc` as Handlers (all middleware remain unchanged)
- Architecture extensible to support HTTP/3 (quic-go)

## Capabilities

### New Capabilities

- `http2-stdlib-server`: Go standard library HTTP/2 Server implementation, using `http.Server.Protocols` to configure HTTP/1+HTTP/2+h2c, connected to the Hertz Engine through a bridge layer.

### Modified Capabilities

(No modifications to existing specs required; this is a pure implementation-level change)

## Impact

**Code Changes**:

- `pkg/gateway/server_http.go`: Add conditional logic to select Server type based on `http2` configuration.
- `pkg/gateway/server_stdlib.go` (new file): Go stdlib HTTP Server + Hertz bridge implementation.
- `go.mod`: Remove `hertz-contrib/http2` and `replace` directives.

**Dependencies**:

- After Phase 1: `hertz-contrib/http2` is retained for client-side HTTP/2.
- After Phase 2: Completely remove `hertz-contrib/http2`.

**Performance**:

- HTTP/1 only: Zero impact (still using Hertz).
- HTTP/1+HTTP/2: Approximately +30-60μs bridging overhead.

**Breaking Changes**:

- HTTP/2 implementation changed from native Hertz to Go stdlib bridge.
- Users with code directly using `hertz-contrib/http2` need to remove it.
