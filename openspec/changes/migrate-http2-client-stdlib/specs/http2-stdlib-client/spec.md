## ADDED Requirements

### Requirement: Standard Library HTTP/2 Client Transport

The system SHALL use Go standard library's `net/http` transport (specifically Go 1.24+ `http.Transport` which natively supports HTTP/2) for establishing and managing HTTP/2 client connections, replacing the legacy `hertz-contrib/http2` implementation.

#### Scenario: Successful HTTP/2 Request

- **WHEN** a proxy request is made to an upstream target configured with `protocol: http2`
- **THEN** the request is sent using HTTP/2 protocol
- **THEN** response headers and body are correctly received and forwarded

#### Scenario: gRPC Trailer Propagation

- **WHEN** a gRPC request (which uses HTTP/2) is forwarded through the proxy
- **THEN** request trailers are correctly sent to the upstream
- **THEN** response trailers (including `grpc-status` and `grpc-message`) are correctly received from the upstream and propagated to the client

### Requirement: Client Factory Adapater

The implementation SHALL provide a `ClientFactory` that produces a client adhering to the Hertz `client.HostClient` interface, bridging Hertz's request/response models to Go's standard library models.

#### Scenario: Hertz Integration

- **WHEN** the proxy initializes a client using `NewClient` with `IsHTTP2: true`
- **THEN** the returned client authenticates as a `client.HostClient`
- **AND** `Do` method calls correctly convert `protocol.Request` to `http.Request` and `http.Response` back to `protocol.Response`
