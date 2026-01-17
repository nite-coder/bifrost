## 1. Configuration Changes

- [ ] 1.1 Update `MetricsOptions` struct in `pkg/config/options.go`
- [ ] 1.2 Add `OTLPMetricsOptions` struct with `service_name`, endpoint, protocol, interval, timeout, insecure fields
- [ ] 1.3 Update `PrometheusOptions` to keep server_id, path, buckets, enabled (no changes needed)
- [ ] 1.4 Update configuration validation in `pkg/config/validation.go`

## 2. Core Metrics Package

- [ ] 2.1 Create `pkg/telemetry/metrics/provider.go` - MeterProvider initialization
- [ ] 2.2 Create `pkg/telemetry/metrics/exporter.go` - Prometheus and OTLP exporter setup
- [ ] 2.3 Create `pkg/telemetry/metrics/bridge.go` - Prometheus client bridge configuration
- [ ] 2.4 Create `pkg/telemetry/metrics/recorder.go` - HTTP metrics recorder (histogram, counters, gauges)
- [ ] 2.5 Create `pkg/telemetry/metrics/middleware.go` - HTTP handler for /metrics endpoint

## 3. Integration with Bifrost (Preserve Hertz Tracer Interface)

- [ ] 3.1 Create new `pkg/telemetry/metrics/recorder.go` implementing `tracer.Tracer` interface
- [ ] 3.2 Replace internal implementation from `prometheus/client_golang` to OTel Metrics API
- [ ] 3.3 Update `pkg/gateway/bifrost.go` to use new recorder from `pkg/telemetry/metrics/`
- [ ] 3.4 Update `pkg/gateway/engine.go` to use new metrics middleware for `/metrics` endpoint
- [ ] 3.5 Add MeterProvider shutdown in `Bifrost.shutdown()`
- [ ] 3.6 Keep `pkg/gateway/upstream.go` using prometheus client (for bridge compatibility)

## 4. Documentation

- [ ] 4.1 Update `docs/configuration.md` with new `metrics.otlp` configuration
- [ ] 4.2 Update CHANGELOG.md with new feature notice

## 5. Testing

- [ ] 5.1 Write unit tests for `pkg/telemetry/metrics/provider.go` using `metric.NewManualReader()`
- [ ] 5.2 Write unit tests for `pkg/telemetry/metrics/recorder.go`
- [ ] 5.3 Write integration test for Prometheus endpoint output
- [ ] 5.4 Write integration test for OTLP push (using ManualReader or mock collector)
- [ ] 5.5 Test plugin compatibility - verify prometheus client metrics appear in output
- [ ] 5.6 Ensure `pkg/telemetry/metrics/` package coverage >= 75%
- [ ] 5.7 Run full release validation: `make release` (includes build, lint, test, e2e-test)
