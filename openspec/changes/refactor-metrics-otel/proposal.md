# Change: Refactor Metrics to Support OTel Push/Pull Mode

## Why

The current metrics implementation only supports Prometheus pull mode via `/metrics` endpoint. Users need the flexibility to use both push (OTLP) and pull (Prometheus) modes simultaneously, especially in cloud-native environments where push mode is preferred for serverless or ephemeral workloads.

## What Changes

- Add new `metrics.otlp` configuration section for push mode
- Create new `pkg/telemetry/metrics/` package with unified OTel Metrics API
- Use OTel Prometheus Bridge to collect existing Prometheus client metrics from plugins
- Support simultaneous push (OTLP) and pull (Prometheus) export modes
- Deprecate `pkg/tracer/prometheus/` package (functionality moves to `pkg/telemetry/metrics/`)
- **Backward Compatible**: Existing `metrics.prometheus` configuration unchanged

## Impact

- Affected specs: `metrics` (new capability)
- Affected code:
  - `pkg/config/options.go` - New `MetricsOptions` structure
  - `pkg/telemetry/metrics/` - New package for OTel metrics
  - `pkg/gateway/bifrost.go` - Initialize new metrics system
  - `pkg/gateway/upstream.go` - Update connection metrics registration
  - `pkg/tracer/prometheus/` - Deprecate, mark for removal
  - `docs/configuration.md` - Update documentation

## New Configuration Structure

```yaml
# Configuration maintains backward compatibility for Prometheus-only users
metrics:
  # Pull mode - Prometheus format endpoint (unchanged from before)
  prometheus:
    enabled: true
    server_id: "api"       # Which server exposes /metrics
    path: "/metrics"
    buckets: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
  
  # Push mode - OTLP exporter (new)
  otlp:
    enabled: false
    service_name: "bifrost"  # OTel resource service.name
    endpoint: "otel-collector:4317"
    protocol: "grpc"         # grpc | http
    interval: 15s            # Push interval
    timeout: 10s
    insecure: false
```

## Technical Approach

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Code                          │
│   (Bifrost core + Plugins using prometheus/client_golang)    │
└─────────────────────────┬───────────────────────────────────┘
                          │
        ┌─────────────────┴─────────────────┐
        │                                   │
        ▼                                   ▼
┌───────────────────┐           ┌───────────────────────┐
│   OTel Metrics    │           │  Prometheus Registry  │
│   (Bifrost core)  │           │  (Plugin metrics)     │
└────────┬──────────┘           └──────────┬────────────┘
         │                                 │
         │    ┌────────────────────────────┘
         │    │  (OTel Prometheus Bridge)
         ▼    ▼
┌─────────────────────────────────────────────────────────────┐
│              OTel MeterProvider (unified)                    │
└───────────────┬─────────────────────────┬───────────────────┘
                │                         │
                ▼                         ▼
┌───────────────────────┐     ┌───────────────────────┐
│  Prometheus Exporter  │     │    OTLP Exporter      │
│  (Pull: /metrics)     │     │  (Push: gRPC/HTTP)    │
└───────────────────────┘     └───────────────────────┘
```

### Key Components

1. **OTel Prometheus Bridge**: Uses `go.opentelemetry.io/otel/bridge/prometheus` to collect metrics from existing `prometheus/client_golang` registrations (for plugin compatibility)

2. **OTel Prometheus Exporter**: Uses `go.opentelemetry.io/otel/exporters/prometheus` to expose `/metrics` endpoint in Prometheus format

3. **OTel OTLP Exporter**: Uses existing `go.opentelemetry.io/otel/exporters/otlp/otlpmetric` for push mode
