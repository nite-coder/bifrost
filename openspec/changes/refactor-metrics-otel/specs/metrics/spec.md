## ADDED Requirements

### Requirement: Unified OTel Metrics System

The system SHALL provide a unified metrics collection system based on OpenTelemetry Metrics API that supports multiple export modes simultaneously.

#### Scenario: Initialize metrics with both exporters enabled

- **WHEN** metrics configuration has both `prometheus.enabled: true` and `otlp.enabled: true`
- **THEN** the system SHALL create a MeterProvider with both Prometheus and OTLP exporters
- **AND** metrics SHALL be available via `/metrics` endpoint (pull)
- **AND** metrics SHALL be pushed to the configured OTLP endpoint (push)

#### Scenario: Initialize metrics with only Prometheus enabled

- **WHEN** metrics configuration has `prometheus.enabled: true` and `otlp.enabled: false`
- **THEN** the system SHALL only expose metrics via `/metrics` endpoint
- **AND** no push exporter SHALL be initialized

#### Scenario: Initialize metrics with only OTLP enabled

- **WHEN** metrics configuration has `prometheus.enabled: false` and `otlp.enabled: true`
- **THEN** the system SHALL only push metrics to the OTLP endpoint
- **AND** no `/metrics` endpoint SHALL be exposed

---

### Requirement: Prometheus Format Pull Mode

The system SHALL expose metrics in Prometheus text format via a configurable HTTP endpoint for pull-based monitoring.

#### Scenario: Serve Prometheus metrics endpoint

- **WHEN** a scraper sends GET request to the configured metrics path (default `/metrics`)
- **THEN** the system SHALL respond with metrics in Prometheus text exposition format
- **AND** the response Content-Type SHALL be `text/plain; version=0.0.4; charset=utf-8`

#### Scenario: Configure custom metrics path

- **WHEN** configuration specifies `metrics.prometheus.path: "/custom/metrics"`
- **THEN** the metrics SHALL be available at `/custom/metrics` instead of `/metrics`

---

### Requirement: OTLP Push Mode

The system SHALL support pushing metrics to an OpenTelemetry Collector using the OTLP protocol.

#### Scenario: Push metrics via gRPC

- **WHEN** configuration specifies `metrics.otlp.protocol: "grpc"`
- **THEN** the system SHALL use OTLP/gRPC exporter to push metrics

#### Scenario: Push metrics via HTTP

- **WHEN** configuration specifies `metrics.otlp.protocol: "http"`
- **THEN** the system SHALL use OTLP/HTTP exporter to push metrics

#### Scenario: Configure push interval

- **WHEN** configuration specifies `metrics.otlp.interval: 30s`
- **THEN** the system SHALL push metrics every 30 seconds

---

### Requirement: Prometheus Client SDK Compatibility

The system SHALL collect metrics registered via `prometheus/client_golang` package to maintain compatibility with existing plugins.

#### Scenario: Collect plugin metrics

- **WHEN** a plugin registers metrics using `prometheus.MustRegister()`
- **THEN** those metrics SHALL be collected by the unified metrics system
- **AND** those metrics SHALL appear in both pull and push exports

#### Scenario: Avoid duplicate registration

- **WHEN** the same metric is registered multiple times during hot reload
- **THEN** the system SHALL handle the duplicate gracefully without panicking

---

### Requirement: HTTP Server Metrics

The system SHALL record standard HTTP server metrics for all incoming requests.

#### Scenario: Record request duration histogram

- **WHEN** an HTTP request completes
- **THEN** the system SHALL record the request duration in a histogram metric
- **AND** the metric SHALL include labels: `server_id`, `method`, `path`, `status_code`, `route_id`, `service_id`

#### Scenario: Record active requests gauge

- **WHEN** requests are being processed
- **THEN** the system SHALL maintain a gauge of currently active requests

#### Scenario: Record request/response body sizes

- **WHEN** an HTTP request completes
- **THEN** the system SHALL record the request and response body sizes
