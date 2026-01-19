package metrics_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/telemetry/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestNewProvider_WithPrometheusEnabled(t *testing.T) {
	opts := config.MetricsOptions{
		Prometheus: config.PrometheusOptions{
			Enabled:  true,
			ServerID: "api",
			Path:     "/metrics",
		},
	}

	provider, err := metrics.NewProvider(context.Background(), opts)
	require.NoError(t, err)
	require.NotNil(t, provider)
	require.NotNil(t, provider.MeterProvider())

	// Verify Prometheus options are stored correctly
	assert.Equal(t, true, provider.PrometheusOptions().Enabled)
	assert.Equal(t, "api", provider.PrometheusOptions().ServerID)
	assert.Equal(t, "/metrics", provider.PrometheusOptions().Path)

	// Verify MetricsHandler is created
	assert.NotNil(t, provider.MetricsHandler())

	// Shutdown should not error
	err = provider.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestNewProvider_WithNothingEnabled(t *testing.T) {
	opts := config.MetricsOptions{}

	provider, err := metrics.NewProvider(context.Background(), opts)
	assert.NoError(t, err)
	assert.Nil(t, provider)
}

func TestProviderNilSafety(t *testing.T) {
	var provider *metrics.Provider = nil

	// MeterProvider should return nil for nil provider
	assert.Nil(t, provider.MeterProvider())

	// Shutdown should not error for nil provider
	err := provider.Shutdown(context.Background())
	assert.NoError(t, err)

	// PrometheusOptions should return empty struct
	opts := provider.PrometheusOptions()
	assert.Equal(t, config.PrometheusOptions{}, opts)

	// MetricsHandler should return nil
	assert.Nil(t, provider.MetricsHandler())
}

func TestNewProvider_WithCustomBuckets(t *testing.T) {
	opts := config.MetricsOptions{
		Prometheus: config.PrometheusOptions{
			Enabled:  true,
			ServerID: "api",
			Buckets:  []float64{0.001, 0.01, 0.1, 1, 10},
		},
	}

	provider, err := metrics.NewProvider(context.Background(), opts)
	require.NoError(t, err)
	require.NotNil(t, provider)

	err = provider.Shutdown(context.Background())
	assert.NoError(t, err)
}

// Note: OTLP tests are typically integration tests or skipped in unit tests
// if they require actual connection or could hang.
// We verify that initialization works but avoid actual connection attempts if possible.
// The provider initialization creates client which *might* try to connect depending on options.
// But mostly OTLP exporter creation is non-blocking until first export unless configured otherwise.
func TestNewProvider_OTLPCreation_GRPC(t *testing.T) {
	grpcEndpoint, _, cleanup := startOTelCollector(t)
	defer cleanup()

	opts := config.MetricsOptions{
		OTLP: config.OTLPMetricsOptions{
			Enabled:     true,
			ServiceName: "test-service",
			Endpoint:    grpcEndpoint,
			Flush:       100 * time.Millisecond,
			Insecure:    true,
		},
	}

	provider, err := metrics.NewProvider(context.Background(), opts)
	require.NoError(t, err)
	require.NotNil(t, provider)

	err = provider.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestNewProvider_OTLPCreation_HTTP(t *testing.T) {
	_, httpEndpoint, cleanup := startOTelCollector(t)
	defer cleanup()

	opts := config.MetricsOptions{
		OTLP: config.OTLPMetricsOptions{
			Enabled:     true,
			ServiceName: "test-service-http",
			Endpoint:    httpEndpoint,
			Flush:       100 * time.Millisecond,
			Insecure:    true,
		},
	}

	provider, err := metrics.NewProvider(context.Background(), opts)
	require.NoError(t, err)
	require.NotNil(t, provider)

	err = provider.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestNewProvider_BothExportersEnabled(t *testing.T) {
	grpcEndpoint, _, cleanup := startOTelCollector(t)
	defer cleanup()

	opts := config.MetricsOptions{
		Prometheus: config.PrometheusOptions{
			Enabled:  true,
			ServerID: "api",
			Path:     "/metrics",
		},
		OTLP: config.OTLPMetricsOptions{
			Enabled:     true,
			ServiceName: "test-service",
			Endpoint:    grpcEndpoint,
			Flush:       100 * time.Millisecond,
			Insecure:    true,
		},
	}

	provider, err := metrics.NewProvider(context.Background(), opts)
	require.NoError(t, err)
	require.NotNil(t, provider)

	// Verify both handlers are configured
	assert.NotNil(t, provider.MetricsHandler())

	err = provider.Shutdown(context.Background())
	assert.NoError(t, err)
}

func startOTelCollector(t *testing.T) (grpcEndpoint string, httpEndpoint string, cleanup func()) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "otel/opentelemetry-collector-contrib:0.114.0",
		ExposedPorts: []string{"4317/tcp", "4318/tcp"},
		WaitingFor:   wait.ForLog("Everything is ready. Begin running and processing data."),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start otel collector: %s", err)
	}

	grpcPort, err := container.MappedPort(ctx, "4317")
	if err != nil {
		t.Fatal(err)
	}
	httpPort, err := container.MappedPort(ctx, "4318")
	if err != nil {
		t.Fatal(err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}

	grpcEndpoint = fmt.Sprintf("%s:%s", host, grpcPort.Port())
	httpEndpoint = fmt.Sprintf("http://%s:%s", host, httpPort.Port())

	return grpcEndpoint, httpEndpoint, func() {
		_ = container.Terminate(ctx)
	}
}
