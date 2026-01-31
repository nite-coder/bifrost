# Proposal - Migrate HTTP/2 Client to Go Stdlib

## Why

The `hertz-contrib/http2` package is no longer maintained. Bifrost's client side still depends on this package (and its fork). To ensure security, stability, and maintainability—similar to the server-side migration—we need to migrate the client-side HTTP/2 implementation to the Go standard library (`net/http`). Furthermore, the gRPC Proxy currently lacks integration tests for HTTP/2 upstreams, which is a# Design - Migrate HTTP/2 Client to Go Stdlib

## Context

ial risk point.

## What Changes

- **New HTTP/2 ClientFactory**: Implement an adapter following the Hertz `client.HostClient` interface, using Go standard library `http.Client` with `http2.Transport` (or Go 1.24+ `http.Transport`) under the hood.
- **Replace HTTP/2 Client**: Modify `pkg/proxy/http/client.go` to replace the existing `hertz-contrib/http2` Factory with the new implementation.
- **New Integration Tests**: Add integration tests for HTTP/2 upstreams in `pkg/proxy/http` or `pkg/gateway` to verify proxy behavior (including gRPC).
- **Remove Dependencies**: Completely remove `hertz-contrib/http2` and `nite-coder/http2` from `go.mod` upon completion.

## Capabilities

### New Capabilities

- `http2-stdlib-client`: New Hertz Client Factory implementation based on Go standard library, supporting HTTP/2 forwarding.

### Modified Capabilities

(None, this is a pure implementation-level change, replacing the existing HTTP/2 implementation)

## Impact

- **Code Changes**:
  - `pkg/proxy/http/client.go`: Use the new Factory.
  - `pkg/common/adaptor` (possibly needed): If more Request/Response conversion helpers are required.
  - `pkg/proxy/http/http2_test.go` (new): Tests targeting HTTP/2 upstreams.
- **Dependencies**: Remove `hertz-contrib/http2`.
- **Performance**: Similar to the server-side migration, a slight conversion overhead is expected, but stability and standard library support are gained.
