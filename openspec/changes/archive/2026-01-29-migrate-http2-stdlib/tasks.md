
## 1. Infrastructure Setup

- [x] 1.1 Create `pkg/gateway/server_stdlib.go` with `StdlibHTTPServer` struct
- [x] 1.2 Implement `hertzBridge` struct implementing `http.Handler`
- [x] 1.3 Add `RequestContext` pool for bridge reuse

## 2. Bridge Implementation

- [x] 2.1 Implement `hertzBridge.ServeHTTP` using `adaptor.CopyToHertzRequest`
- [x] 2.2 Implement response writing from Hertz `Response` to `http.ResponseWriter`
- [x] 2.3 Implement trailer headers support for gRPC responses

## 3. Server Integration

- [x] 3.1 Modify `pkg/gateway/server_http.go` to conditionally create server based on `http2` config
- [x] 3.2 Implement `http.Server.Protocols` configuration (HTTP/1 + HTTP/2 + h2c)
- [x] 3.3 Wire `StdlibHTTPServer` to use `Engine.ServeHTTP` via bridge

## 4. Configuration

- [x] 4.1 Verify existing `http2` config option works with new server selection
- [x] 4.2 Expose HTTP/2 settings (e.g., `max_concurrent_streams`) if needed

## 5. Testing & Verification

- [x] 5.1 Run existing `pkg/proxy/grpc/proxy_test.go` - all tests must pass
- [x] 5.2 Add integration test for HTTP/2 request via bridge
- [x] 5.3 Verify gRPC `content-type` and trailer headers pass through bridge
- [x] 5.4 Benchmark bridge latency overhead

## 6. Cleanup

- [x] 6.1 Remove `hertz-contrib/http2` from server-side code in `server_http.go`
- [x] 6.2 Update `go.mod` to remove unused server-side http2 imports
- [x] 6.3 Update documentation for HTTP/2 configuration
