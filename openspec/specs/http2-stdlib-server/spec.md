# http2-stdlib-server Specification

## Purpose

TBD - created by archiving change migrate-http2-stdlib. Update Purpose after archive.

## Requirements

### Requirement: Hybrid Server Selection

The system SHALL dynamically select the Server type based on configuration:

- When `http2: false`, use Hertz Server.
- When `http2: true`, use Go stdlib `net/http.Server`.

#### Scenario: HTTP/1 only mode

- **WHEN** configured with `http2: false`
- **THEN** system uses Hertz Server to handle requests without extra latency overhead.

#### Scenario: HTTP/1 + HTTP/2 mode

- **WHEN** configured with `http2: true`
- **THEN** system uses Go stdlib `net/http.Server` to handle requests.

---

### Requirement: Hertz Bridge Request Conversion

The system SHALL use `adaptor.CopyToHertzRequest` to convert `http.Request` to Hertz `RequestContext`.

#### Scenario: HTTP/2 request conversion

- **WHEN** Go stdlib server receives an HTTP/2 request
- **THEN** system use `adaptor.CopyToHertzRequest` to fill Hertz `RequestContext`
- **AND** request is passed to the Hertz Engine handler chain

#### Scenario: Headers preserved

- **WHEN** HTTP/2 request contains any headers
- **THEN** all headers are correctly passed to Hertz `RequestContext`

---

### Requirement: Hertz Bridge Response Writing

The system SHALL write Hertz `Response` back to `http.ResponseWriter`.

#### Scenario: Response headers and body

- **WHEN** Hertz handler chain completes processing
- **THEN** system copies response headers to `http.ResponseWriter`
- **AND** system writes response body to `http.ResponseWriter`

#### Scenario: Response status code

- **WHEN** Hertz handler sets status code
- **THEN** the status code is correctly written to the HTTP/2 response

---

### Requirement: HTTP/2 Protocol Configuration

The system SHALL use Go 1.24+ `http.Server.Protocols` to configure HTTP/2.

#### Scenario: HTTP/2 over TLS

- **WHEN** TLS is configured and `http2: true`
- **THEN** Server supports HTTP/2 over TLS (h2)

#### Scenario: HTTP/2 over cleartext (h2c)

- **WHEN** `http2: true` is configured and TLS is not used
- **THEN** Server use `SetUnencryptedHTTP2(true)` to support h2c

---

### Requirement: Unified Handler Chain

The system SHALL uniformly use Hertz `Engine.ServeHTTP` to handle all requests.

#### Scenario: Middleware execution

- **WHEN** HTTP/2 request enters the system through the bridge
- **THEN** all Hertz middleware execute in order

#### Scenario: Existing handlers unchanged

- **WHEN** users have defined Hertz handlers
- **THEN** these handlers can handle HTTP/2 requests without any modification

---

### Requirement: gRPC Proxy Compatibility

The system SHALL ensure gRPC Unary requests can be correctly handled through the bridge.

#### Scenario: gRPC content-type header

- **WHEN** gRPC request reaches the bridge
- **THEN** `content-type: application/grpc` header is correctly passed

#### Scenario: gRPC trailer headers

- **WHEN** gRPC response contains trailer headers
- **THEN** `grpc-status` and `grpc-message` are correctly written to HTTP/2 response trailers

#### Scenario: Existing gRPC tests pass

- **WHEN** executing `pkg/proxy/grpc/proxy_test.go`
- **THEN** all tests pass
