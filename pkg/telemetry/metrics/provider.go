// Package metrics provides OpenTelemetry metrics integration for Bifrost.
package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	promexp "go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	globalMeterProvider *sdkmetric.MeterProvider
	mu                  sync.RWMutex
)

// GetMeterProvider returns the global MeterProvider.
// This allows users to use OTel SDK for custom metrics if they choose to.
func GetMeterProvider() *sdkmetric.MeterProvider {
	mu.RLock()
	defer mu.RUnlock()
	return globalMeterProvider
}

// SetMeterProvider sets the global MeterProvider.
func SetMeterProvider(mp *sdkmetric.MeterProvider) {
	mu.Lock()
	defer mu.Unlock()
	globalMeterProvider = mp
}

// slogErrorLogger adapts slog.Logger to promhttp.Logger interface.
type slogErrorLogger struct{}

func (l *slogErrorLogger) Println(v ...interface{}) {
	slog.Error("promhttp error", "details", fmt.Sprint(v...))
}

// Provider holds the MeterProvider and associated resources.
type Provider struct {
	meterProvider  *sdkmetric.MeterProvider
	prometheusOpts config.PrometheusOptions
	metricsHandler http.Handler // The handler for /metrics endpoint
}

// NewProvider creates a new metrics Provider with the configured exporters.
// It supports both Prometheus (pull) and OTLP (push) export modes simultaneously.
func NewProvider(ctx context.Context, metricsOpts config.MetricsOptions) (*Provider, error) {
	// Bridge to collect existing prometheus/client_golang metrics.
	// We use the helper from bridge.go (or inline here if simple).
	// Using the centralized NewBridge function.
	promProducer := NewBridge()

	var readers []sdkmetric.Reader
	var metricsHandler http.Handler

	// Configure Prometheus exporter (pull mode)
	if metricsOpts.Prometheus.Enabled {
		// Create exporter purely for OTel Core metrics (no producer attached)
		// This uses an isolated registry for OTel metrics.
		promExporter, registry, err := createPrometheusExporter()
		if err != nil {
			return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
		}
		readers = append(readers, promExporter)

		// Use MergedGatherer to serve:
		// 1. Core OTel metrics (from registry)
		// 2. Legacy/Plugin metrics (directly from DefaultGatherer, NO BRIDGE involved)
		// This separation prevents "invalid metric type" errors in the Pull endpoint.
		gatherers := prom.Gatherers{
			registry,
			prom.DefaultGatherer,
		}

		// Create HTTP handler from the combined gatherers
		// Use slog-based error logger to integrate with Bifrost's logging system
		metricsHandler = promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{
			ErrorLog: &slogErrorLogger{},
		})
	}

	// Configure OTLP exporter (push mode)
	if metricsOpts.OTLP.Enabled {
		// OTLP Reader uses the Bridge to include legacy metrics in the push
		otlpReader, err := createOTLPReader(ctx, metricsOpts.OTLP, promProducer)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP reader: %w", err)
		}
		readers = append(readers, otlpReader)
	}

	if len(readers) == 0 {
		return nil, nil
	}

	// Create resource with service name
	serviceName := "bifrost"
	if metricsOpts.OTLP.Enabled && metricsOpts.OTLP.ServiceName != "" {
		serviceName = metricsOpts.OTLP.ServiceName
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Build MeterProvider options
	opts := []sdkmetric.Option{
		sdkmetric.WithResource(res),
	}
	for _, reader := range readers {
		opts = append(opts, sdkmetric.WithReader(reader))
	}

	meterProvider := sdkmetric.NewMeterProvider(opts...)
	SetMeterProvider(meterProvider)

	return &Provider{
		meterProvider:  meterProvider,
		prometheusOpts: metricsOpts.Prometheus,
		metricsHandler: metricsHandler,
	}, nil
}

// MeterProvider returns the underlying OTel MeterProvider.
func (p *Provider) MeterProvider() *sdkmetric.MeterProvider {
	if p == nil {
		return nil
	}
	return p.meterProvider
}

// Shutdown gracefully shuts down the MeterProvider.
func (p *Provider) Shutdown(ctx context.Context) error {
	if p == nil || p.meterProvider == nil {
		return nil
	}
	return p.meterProvider.Shutdown(ctx)
}

// PrometheusOptions returns the Prometheus configuration options.
func (p *Provider) PrometheusOptions() config.PrometheusOptions {
	if p == nil {
		return config.PrometheusOptions{}
	}
	return p.prometheusOpts
}

// MetricsHandler returns the HTTP handler for /metrics endpoint.
func (p *Provider) MetricsHandler() http.Handler {
	if p == nil {
		return nil
	}
	return p.metricsHandler
}

// createPrometheusExporter creates a Prometheus exporter for pull mode.
func createPrometheusExporter() (*promexp.Exporter, prom.Gatherer, error) {
	registry := prom.NewRegistry()
	exporter, err := promexp.New(
		promexp.WithRegisterer(registry),
	)
	if err != nil {
		return nil, nil, err
	}
	return exporter, registry, nil
}

// createOTLPReader creates an OTLP exporter reader for push mode.
func createOTLPReader(ctx context.Context, opts config.OTLPMetricsOptions, producer sdkmetric.Producer) (sdkmetric.Reader, error) {
	protocol := strings.ToLower(opts.Protocol)
	if protocol == "" {
		protocol = "grpc"
	}

	interval := opts.Interval
	if interval == 0 {
		interval = 15 * time.Second
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}

	var exporter sdkmetric.Exporter
	var err error

	switch protocol {
	case "grpc":
		grpcOpts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(opts.Endpoint),
			otlpmetricgrpc.WithTimeout(timeout),
		}
		if opts.Insecure {
			grpcOpts = append(grpcOpts, otlpmetricgrpc.WithTLSCredentials(insecure.NewCredentials()))
		}
		exporter, err = otlpmetricgrpc.New(ctx, grpcOpts...)
	case "http":
		httpOpts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(opts.Endpoint),
			otlpmetrichttp.WithTimeout(timeout),
		}
		if opts.Insecure {
			httpOpts = append(httpOpts, otlpmetrichttp.WithInsecure())
		}
		exporter, err = otlpmetrichttp.New(ctx, httpOpts...)
	default:
		return nil, fmt.Errorf("unsupported OTLP protocol: %s", protocol)
	}

	if err != nil {
		return nil, err
	}

	reader := sdkmetric.NewPeriodicReader(exporter,
		sdkmetric.WithInterval(interval),
		sdkmetric.WithProducer(producer),
	)
	return reader, nil
}
