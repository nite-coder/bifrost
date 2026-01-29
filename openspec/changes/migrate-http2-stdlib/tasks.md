
## 1. Infrastructure Setup

- [ ] 1.1 Create `pkg/gateway/server_stdlib.go` with `StdlibHTTPServer` struct
- [ ] 1.2 Implement `hertzBridge` struct implementing `http.Handler`
- [ ] 1.3 Add `RequestContext` pool for bridge reuse

## 2. Bridge Implementation

- [ ] 2.1 Implement `hertzBridge.ServeHTTP` using `adaptor.CopyToHertzRequest`
- [ ] 2.2 Implement response writing from Hertz `Response` to `http.ResponseWriter`
- [ ] 2.3 Implement trailer headers support for gRPC responses

## 3. Server Integration

- [ ] 3.1 Modify `pkg/gateway/server_http.go` to conditionally create server based on `http2` config
- [ ] 3.2 Implement `http.Server.Protocols` configuration (HTTP/1 + HTTP/2 + h2c)
- [ ] 3.3 Wire `StdlibHTTPServer` to use `Engine.ServeHTTP` via bridge

## 4. Configuration

- [ ] 4.1 Verify existing `http2` config option works with new server selection
- [ ] 4.2 Expose HTTP/2 settings (e.g., `max_concurrent_streams`) if needed

## 5. Testing & Verification

- [ ] 5.1 Run existing `pkg/proxy/grpc/proxy_test.go` - all tests must pass
- [ ] 5.2 Add integration test for HTTP/2 request via bridge
- [ ] 5.3 Verify gRPC `content-type` and trailer headers pass through bridge
- [ ] 5.4 Benchmark bridge latency overhead

## 6. Cleanup

- [ ] 6.1 Remove `hertz-contrib/http2` from server-side code in `server_http.go`
- [ ] 6.2 Update `go.mod` to remove unused server-side http2 imports
- [ ] 6.3 Update documentation for HTTP/2 configuration
