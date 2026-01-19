# metrics Specification

## Purpose
TBD - created by archiving change refactor-metrics-otel. Update Purpose after archive.
## Requirements
### Requirement: Split Pull/Push Architecture

The system SHALL use separate data paths for Pull and Push modes to guarantee stability while providing full coverage.

#### Scenario: Pull mode uses Merged Gatherer

- **WHEN** Prometheus pull mode is enabled
- **THEN** the system SHALL use a Merged Gatherer combining:
  - `prometheus.DefaultGatherer` (legacy + plugin metrics)
  - OTel Prometheus Exporter's isolated registry (core metrics)
- **AND** the `/metrics` endpoint SHALL serve metrics without any round-trip conversion
- **AND** the endpoint SHALL NOT crash due to metric type conversion issues

#### Scenario: Push mode uses Bridge conversion

- **WHEN** OTLP push mode is enabled
- **THEN** the system SHALL use Prometheus Bridge to convert metrics to OTel format
- **AND** if individual metrics fail conversion, they SHALL be skipped without crashing
- **AND** the push pipeline SHALL remain stable

---

### Requirement: Dual SDK Support

The system SHALL support both Prometheus SDK and OTel SDK for custom metrics instrumentation.

#### Scenario: User uses Prometheus SDK for custom metrics

- **WHEN** a plugin or middleware uses `prometheus.NewCounter()` or similar
- **THEN** those metrics SHALL appear in Pull mode (`/metrics` endpoint)
- **AND** those metrics SHALL appear in Push mode via Bridge conversion

#### Scenario: User uses OTel SDK for custom metrics

- **WHEN** a plugin or middleware uses `bifrost.GetMeterProvider().Meter()` to create metrics
- **THEN** those metrics SHALL appear in Pull mode via OTel Prometheus Exporter
- **AND** those metrics SHALL appear in Push mode via native OTLP export

#### Scenario: Mixed SDK usage

- **WHEN** different plugins use different SDKs (some use Prometheus, some use OTel)
- **THEN** all metrics SHALL be visible in both Pull and Push outputs
- **AND** there SHALL be no conflicts between the two SDK approaches

---

### Requirement: Unified Metrics System

The system SHALL provide a unified metrics collection system that supports multiple export modes simultaneously.

#### Scenario: Initialize metrics with both exporters enabled

- **WHEN** metrics configuration has both `prometheus.enabled: true` and `otlp.enabled: true`
- **THEN** the system SHALL create a MeterProvider with both exporters configured
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
- **AND** the response SHALL NOT return HTTP 500 errors due to metric conversion

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

#### Scenario: Graceful handling of conversion failures

- **WHEN** a Prometheus metric cannot be converted to OTel format
- **THEN** that specific metric SHALL be skipped
- **AND** other metrics SHALL continue to be pushed normally
- **AND** the system SHALL NOT crash

---

### Requirement: Prometheus Client SDK Compatibility

The system SHALL collect metrics registered via `prometheus/client_golang` package to maintain compatibility with existing plugins.

#### Scenario: Collect plugin metrics in Pull mode

- **WHEN** a plugin registers metrics using `prometheus.MustRegister()`
- **THEN** those metrics SHALL appear in the `/metrics` endpoint via `DefaultGatherer`

#### Scenario: Collect plugin metrics in Push mode

- **WHEN** a plugin registers metrics using `prometheus.MustRegister()`
- **THEN** those metrics SHALL be converted via Prometheus Bridge
- **AND** those metrics SHALL be pushed to the OTLP endpoint

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

