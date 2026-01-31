## 1. Integration Tests (Baseline)

- [x] 1.1 Create `pkg/proxy/http/proxy_http2_test.go` with a test server supporting ONLY HTTP/2 (TLS).
- [x] 1.2 Implement basic request/response verification test case for HTTP/2 upstream.
- [x] 1.3 Implement gRPC Trailer verification test case (simulating `grpc-status` trailers).

## 2. Implementation

- [x] 2.1 Implement `stdlibFactory` struct in `pkg/proxy/http/stdlib_factory.go` adhering to `suite.ClientFactory`.
- [x] 2.2 Implement `stdlibHostClient` Adapter to bridge `net/http` Request/Response with Hertz.
- [x] 2.3 Modify `pkg/proxy/http/client.go` to use `stdlibFactory` when `IsHTTP2` is true.

## 3. Verification & Cleanup

- [x] 3.1 Verify all tests in `pkg/proxy/http` pass (especially new HTTP/2 tests).
- [x] 3.2 Remove `github.com/hertz-contrib/http2` and `github.com/nite-coder/http2` from `go.mod`.
- [x] 3.3 Verify `make release` passes.
