## Context

Bifrost currently uses `prometheus/client_golang` directly for metrics collection with only pull mode support. Users need both push and pull modes, and existing plugins already use Prometheus client SDK. The challenge is to unify metrics while maintaining backward compatibility with plugin-registered metrics.

## Goals / Non-Goals

**Goals:**

- Unified metrics system supporting both Push (OTLP) and Pull (Prometheus) modes simultaneously
- **Dual SDK support**: Users can choose either Prometheus SDK or OTel SDK for custom metrics
- Collect metrics from existing Prometheus client registrations (plugin compatibility)
- Prometheus text format for pull mode
- Clean configuration structure

**Non-Goals:**

- Migrate existing plugins to OTel API (they continue using Prometheus client)
- Support direct exporters (e.g., Datadog, CloudWatch) - users can route via OTel Collector
- Provide new Bifrost-specific metrics API (users continue using `prometheus/client_golang` for custom metrics)

## Decisions

### Decision 1: Use OTel SDK with Prometheus Bridge

**Choice**: Use OpenTelemetry Metrics SDK as the core, with Prometheus Bridge to collect existing `prometheus/client_golang` metrics.

**Rationale**:

- OTel provides native support for multiple exporters
- Prometheus Bridge (`go.opentelemetry.io/otel/bridge/prometheus`) can wrap existing Prometheus registry
- Avoids rewriting plugin metrics

**Alternatives Considered**:

1. **Dual metrics systems** - Run Prometheus and OTel in parallel
   - Rejected: Increased complexity, duplicate metrics logic
2. **Pure OTel migration** - Require all code to use OTel API
   - Rejected: Breaking change for plugins, not backward compatible

### Decision 2: Use OTel Prometheus Exporter for Pull Mode

**Choice**: Use `go.opentelemetry.io/otel/exporters/prometheus` to serve `/metrics` endpoint.

**Rationale**:

- Outputs standard Prometheus text format
- Automatically collects from OTel MeterProvider
- Works with Prometheus Bridge to include plugin metrics

### Decision 3: Package Structure

**Choice**: Create new `pkg/telemetry/metrics/` package, deprecate `pkg/tracer/prometheus/`.

```
pkg/telemetry/
└── metrics/
    ├── provider.go      # MeterProvider initialization
    ├── exporter.go      # Exporter configuration (prometheus, otlp)
    ├── recorder.go      # HTTP request metrics recorder (replaces serverTracer)
    ├── middleware.go    # HTTP middleware to serve /metrics
    └── bridge.go        # Prometheus bridge configuration
```

**Rationale**:

- "telemetry/metrics" aligns better with OTel three pillars (metrics, tracing, logging)
- Clear separation from tracing (`pkg/tracing/`)
- Allows future expansion to include tracing and logging under `pkg/telemetry/`

### Decision 4: Configuration Backward Compatibility

**Choice**: Keep `metrics.prometheus` structure unchanged, add `metrics.otlp` as new section.

**Rationale**:

- Prometheus-only users have zero configuration changes
- OTLP users add new section with `service_name` inside
- No breaking change for existing deployments

### Decision 5: Preserve Hertz Tracer Interface

**Choice**: Keep implementing `github.com/cloudwego/hertz/pkg/common/tracer.Tracer` interface. Only replace internal implementation from `prometheus/client_golang` to OTel Metrics API.

**Current Architecture (unchanged):**

```
Hertz Server
    └── tracer.Tracer interface
            └── Start(ctx, c) / Finish(ctx, c)
```

**Implementation Change:**

```go
// Before: pkg/tracer/prometheus/tracer.go
type serverTracer struct {
    httpServerRequestDuration *prom.HistogramVec  // prometheus/client_golang
}

// After: pkg/telemetry/metrics/recorder.go
type serverTracer struct {
    httpServerRequestDuration metric.Float64Histogram  // OTel Metrics API
}
```

**Rationale**:

- Hertz tracer interface provides lifecycle hooks (Start/Finish) for each request
- Minimal code change - only metric recording implementation changes
- No changes needed in `pkg/gateway/bifrost.go` tracer registration

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| OTel Prometheus Bridge may miss some metric types | Test with existing plugins; document limitations |
| Performance overhead of bridge | Benchmark before/after; OTel is designed for low overhead |
| Hot reload may cause metric duplication | Use singleton pattern for MeterProvider |

## Dependencies

New dependencies to add:

```go
go.opentelemetry.io/otel/sdk/metric
go.opentelemetry.io/otel/exporters/prometheus
go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc
go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp
go.opentelemetry.io/otel/bridge/prometheus
```

## Resolved Questions

- ~~Should we provide a migration tool/script for configuration files?~~ → No, config is backward compatible
- ~~Should metrics have their own `service_name` or share with tracing?~~ → Separate, defined in `metrics.otlp.service_name`
