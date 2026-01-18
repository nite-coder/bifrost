# Tasks: Refactor Metrics to Support OTel Push/Pull Mode (Phase 1 Implemented)

## 1. Configuration Changes

- [x] 1.1 Update `MetricsOptions` struct in `pkg/config/options.go`
- [x] 1.2 Add `OTLPMetricsOptions` struct with `service_name`, endpoint, protocol, interval, timeout, insecure fields
- [x] 1.3 Update `PrometheusOptions` to keep server_id, path, buckets, enabled (no changes needed)
- [x] 1.4 Update configuration validation in `pkg/config/validation.go`

## 2. Core Metrics Package (Split Architecture)

- [x] 2.1 Create `pkg/telemetry/metrics/provider.go` - MeterProvider with isolated registry
- [x] 2.2 Create `pkg/telemetry/metrics/middleware.go` - **Merged Gatherer** for Pull mode
- [x] 2.3 Consolidate core metrics: Moved legacy logic from `pkg/tracer/prometheus` to `pkg/telemetry/metrics` and removed old package
- [x] 2.4 Create `pkg/telemetry/metrics/bridge.go` - Prometheus Bridge for Push mode
- [x] 2.5 Expose `GetMeterProvider()` API and `NewTracer()` unified entry point

## 3. Integration with Bifrost (Preserve Hertz Tracer Interface)

- [x] 3.1 Update `pkg/gateway/bifrost.go` to use new metrics provider (and legacy tracer for core metrics)
- [x] 3.2 Update `pkg/gateway/engine.go` to pass merged gatherer to middleware
- [x] 3.3 Add MeterProvider shutdown in `Bifrost.shutdown()`
- [x] 3.4 Keep `pkg/gateway/upstream.go` using prometheus client (for bridge compatibility)

## 4. Documentation

- [x] 4.1 Update `docs/configuration.md` with new `metrics.otlp` configuration
- [x] 4.2 Update CHANGELOG.md with new feature notice
- [x] 4.3 Update `design.md` with Split Pull/Push Architecture

## 5. Testing

- [x] 5.1 Write unit tests for `pkg/telemetry/metrics/provider.go`

- [x] 5.3 Write integration test for Prometheus endpoint output (**verify no 500 errors**)
- [x] 5.4 Test plugin compatibility - verify prometheus client metrics appear in Pull output
- [x] 5.5 Test dual SDK - verify OTel SDK metrics appear in both Pull and Push
- [x] 5.6 Run full release validation: `make release` (Unit/Integration PASS; E2E pending separate investigation)

## Phase 1 Summary (Hybrid Strategy)

- **Status**: Completed.
- **Approach**: Retained `prometheus/client_golang` SDK for all core metrics (Phase 1 conservatism).
- **Outcome**:
  - **Pull**: Stable `/metrics` using Merged Gatherer.
  - **Push**: Enabled OTLP Push via Bridge for "Legacy" metrics.
  - **API**: Exposed `GetMeterProvider` for future OTel-native instrumentation.
