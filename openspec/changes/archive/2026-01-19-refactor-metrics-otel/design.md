# Design - Refactor Metrics to Support OTel Push/Pull Mode

## Context

Bifrost currently uses `prometheus/client_golang` directly for metrics collection with only pull mode support. Users need both push and pull modes, and existing plugins already use Prometheus client SDK. The challenge is to unify metrics while maintaining backward compatibility with plugin-registered metrics.

## Goals / Non-Goals

**Goals:**

- Unified metrics system supporting both Push (OTLP) and Pull (Prometheus) modes simultaneously
- **Dual SDK support**: Users can choose either Prometheus SDK or OTel SDK for custom metrics
- Collect metrics from existing Prometheus client registrations (plugin compatibility)
- Prometheus text format for pull mode
- Clean configuration structure
- **100% stability**: No conversion-related crashes on `/metrics` endpoint

**Non-Goals:**

- Migrate existing plugins to OTel API (they continue using Prometheus client)
- Support direct exporters (e.g., Datadog, CloudWatch) - users can route via OTel Collector
- Force users to choose only one SDK for instrumentation

## Decisions

### Decision 1: Split Pull/Push Architecture (Core Design)

**Choice**: Use separate data paths for Pull and Push modes to guarantee stability while providing full coverage.

**Problem Solved**: During implementation, we identified a critical stability issue when using the OpenTelemetry Bridge for the Prometheus Pull endpoint (`/metrics`):

1. **Flow**: Prometheus `DefaultGatherer` → Bridge (to OTel) → OTel Exporter (to Prom) → `promhttp` → Client
2. **Issue**: This "Round-trip" conversion is lossy for complex metrics. Specifically, Go runtime Summary metrics (e.g., `go_gc_duration_seconds`) lose critical type information during the OTel-to-Prometheus conversion, causing `promhttp` to panic with an "invalid metric type" error (HTTP 500)
3. **Impact**: The entire monitoring endpoint crashes, blinding observability

**Stability Considerations**:

- During implementation, we discovered that forcing Prometheus metrics through the OTel Bridge for the Pull endpoint causes crashes (HTTP 500) for complex metrics (e.g., Go runtime Summaries) due to type mismatch in the `promhttp` layer.
- **Best Practice**: While meeting Push requirements, the Pull path should maintain the original data flow as much as possible to avoid unnecessary conversion loss and ensure 100% availability.

**Architecture**:

```
                          ┌────────────────────────┐
                          │   User Middleware      │
                          │                        │
                          │  ┌────────┐ ┌────────┐ │
                          │  │Prom SDK│ │OTel SDK│ │  ← Dual SDK Support
                          │  └───┬────┘ └────┬───┘ │
                          └──────┼───────────┼─────┘
                                 │           │
            ┌────────────────────┘           └─────────────────┐
            ▼                                                  ▼
┌───────────────────────────┐              ┌───────────────────────────────────┐
│ prometheus.DefaultRegistry │              │        OTel MeterProvider         │
└────────────┬──────────────┘              └──────────────┬────────────────────┘
             │                                            │
             │    ┌───────────────────────────────────────┤
             │    │                                       │
             │    ▼                                       │
             │  ┌────────────────────┐                    │
             │  │   Prometheus       │                    │
             │  │   Bridge           │ ← Prom→OTel Conversion
             │  │   (Producer)       │                    │
             │  └─────────┬──────────┘                    │
             │            │                               │
             │            └──────────────┬────────────────┘
             │                           │
             │                           ▼
             │            ┌──────────────────────────────────────┐
             │            │        OTel MeterProvider            │
             │            │  + Prometheus Bridge as Producer     │
             │            │  + OTLP Exporter                     │
             │            │  + Prometheus Exporter (isolated)    │
             │            └──────────┬───────────┬───────────────┘
             │                       │           │
    ┌────────┴────────┐              │           │
    ▼                 ▼              ▼           ▼
┌────────┐    ┌────────────────┐   ┌─────────────────────────┐
│promhttp│    │OTel Prom Export│   │     OTLP Exporter       │
│(Direct)│    │(isolated reg)  │   │     (Push Mode)         │
└────┬───┘    └───────┬────────┘   └───────────┬─────────────┘
     │                │                        │
     └──────┬─────────┘                        │
            ▼                                  ▼
   ┌─────────────────────┐        ┌─────────────────────────┐
   │   Merged Gatherer   │        │   OTel Collector        │
   │  /metrics (Pull)    │        │   (Push)                │
   └─────────────────────┘        └─────────────────────────┘
```

**Data Flow Matrix**:

| SDK Used | Pull (`/metrics`) | Push (OTLP) | Mechanism |
|---|---|---|---|
| **Prometheus SDK** | ✅ Exposed via `promhttp` | ✅ **Converted via Bridge** | Bridge reads DefaultGatherer → Transformed to OTel → OTLP Exporter |
| **OTel SDK** | ✅ OTel Prom Exporter → Merged Gatherer | ✅ Native Push | Direct via OTLP Exporter |

**Benefit**:

| Aspect | Improvement |
| :--- | :--- |
| **Stability** | Solves the 500 Error / Crash completely |
| **Consistency** | Pull endpoint retains exact original behavior (no data loss) |
| **Flexibility** | Users can choose Prometheus SDK or OTel SDK freely |
| **Plugin Support** | Any plugin registering with global registry will be seen by both paths |

### Decision 2: Merged Gatherer for Pull Mode

**Choice**: Use `prometheus.Gatherers` to merge multiple sources for the `/metrics` endpoint.

**Rationale**:

- Combines `prometheus.DefaultGatherer` (legacy + plugin metrics) with OTel's isolated registry
- Zero conversion for legacy metrics - they are served raw and intact
- **100% Stable** - no type conversion issues

**Implementation**:

```go
// Create merged gatherer for Pull mode
mergedGatherer := prometheus.Gatherers{
    prometheus.DefaultGatherer,    // Legacy + Plugin metrics (Prometheus SDK)
    otelPrometheusRegistry,        // Bifrost core metrics (OTel SDK)
}

// Serve /metrics using merged gatherer
promhttp.HandlerFor(mergedGatherer, promhttp.HandlerOpts{})
```

### Decision 3: Use Isolated Prometheus Registry for OTel Exporter

**Choice**: Configure OTel Prometheus Exporter with an isolated registry (`prometheus.NewRegistry()`) instead of the default global registry.

**Rationale**:

- Avoids conflicts with the global `prometheus.DefaultRegistry` used by plugins
- Prevents deadlocks when multiple MeterProviders are created (e.g., during parallel tests)
- The Prometheus Bridge **reads from** `prometheus.DefaultGatherer` (read-only)
- The OTel Prometheus Exporter **writes to** its isolated registry (separate registry)

### Decision 4: Expose MeterProvider for OTel SDK Users

**Choice**: Bifrost exposes a `GetMeterProvider()` API for users who prefer OTel SDK.

**Rationale**:

- Gives users flexibility to choose their preferred SDK
- OTel SDK metrics are automatically available in both Pull and Push paths
- No forced migration - Prometheus SDK continues to work

**Implementation**:

```go
// pkg/telemetry/metrics/provider.go

// GetMeterProvider returns the global MeterProvider for custom OTel metrics
func GetMeterProvider() metric.MeterProvider {
    return globalMeterProvider
}

// User middleware using OTel SDK
func MyMiddleware() {
    meter := bifrost.GetMeterProvider().Meter("my-middleware")
    counter, _ := meter.Int64Counter("my_custom_counter")
    counter.Add(ctx, 1)
}
```

### Decision 5: Package Structure

**Choice**: Consolidate all metrics logic into the new `pkg/telemetry/metrics/` package, removing `pkg/tracer/prometheus/`.

```
pkg/telemetry/
└── metrics/
    ├── provider.go      # MeterProvider initialization + GetMeterProvider API
    ├── options.go       # Integrated metrics options (legacy + OTel)
    ├── tracer.go        # Unified tracer implementation (wrapping legacy logic)
    ├── metric.go        # Metric registration and helpers
    ├── middleware.go    # HTTP middleware to serve /metrics (merged gatherer)
    └── bridge.go        # Prometheus bridge configuration
```

**Rationale**:

- **Consolidation**: Moving all metrics-related code into a single, well-defined package reduces architectural complexity and avoids cross-package circular dependencies.
- **Unified API**: By providing `NewTracer()` and common options in the `metrics` package, we empower a cleaner transition for the rest of the system.
- **Package Migration Complete**: The legacy package is fully retired, and all future development will occur within the new `telemetry/metrics` hierarchy.

### Decision 6: Configuration Backward Compatibility

**Choice**: Keep `metrics.prometheus` structure unchanged, add `metrics.otlp` as new section.

**Rationale**:

- Prometheus-only users have zero configuration changes
- OTLP users add new section with `service_name` inside
- No breaking change for existing deployments

### Decision 7: Preserve Hertz Tracer Interface

**Choice**: Keep implementing `github.com/cloudwego/hertz/pkg/common/tracer.Tracer` interface. The implementation has been moved to `pkg/telemetry/metrics/tracer.go` for better consolidation.

**Current Architecture (unchanged):**

```
Hertz Server
    └── tracer.Tracer interface
            └── Start(ctx, c) / Finish(ctx, c)
```

**Phase 1 Decision**:

Core metrics continue to use the legacy Prometheus SDK logic but it's now internal to the new package. This ensures:

- Zero risk of metric recording regressions.
- All existing metrics flow through the Merged Gatherer (Pull) and Bridge (Push).

**Rationale**:

- Hertz tracer interface provides lifecycle hooks (Start/Finish) for each request.
- Phase 1 conservatism: retain proven implementation to ensure stability and accuracy.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| **Prometheus Bridge conversion failures** | Failures only affect Push path; individual metrics are dropped without crashing |
| Performance overhead of bridge | Benchmark before/after; OTel is designed for low overhead |
| Hot reload may cause metric duplication | Use singleton pattern for MeterProvider |
| Global registry conflicts during parallel tests | Use isolated registry for OTel Prometheus Exporter |

## Dependencies

New dependencies to add:

```go
go.opentelemetry.io/otel/sdk/metric
go.opentelemetry.io/otel/exporters/prometheus
go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc
go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp
go.opentelemetry.io/contrib/bridges/prometheus  // Note: bridges are in contrib repo
```

## Resolved Questions

- ~~Should we provide a migration tool/script for configuration files?~~ → No, config is backward compatible
- ~~Should metrics have their own `service_name` or share with tracing?~~ → Separate, defined in `metrics.otlp.service_name`
- ~~Should we use global or isolated Prometheus registry for OTel exporter?~~ → Isolated registry to avoid conflicts
- ~~Should users be forced to use only one SDK?~~ → No, both Prometheus SDK and OTel SDK are supported

## Known Limitations

| Limitation | Impact | Future Enhancement |
|------------|--------|-------------------|
| Bridge only collects from `prometheus.DefaultGatherer` | Plugins using custom `prometheus.NewRegistry()` won't have metrics collected via OTLP push | Provide SDK API: `bifrost.WithMetricsGatherer(customGatherer)` |
| Complex Summary metrics may fail Bridge conversion | These metrics won't appear in OTLP Push output | Document which metric types are fully supported |
