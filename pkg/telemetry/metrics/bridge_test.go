package metrics

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestBridge_Smoke(t *testing.T) {
	// 1. Register a Prometheus indicator for exclusive testing (using DefaultRegistry)
	counter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "bifrost_push_smoke_test_total",
		Help: "Smoke test for Bridge conversion",
	})

	// If already registered (e.g., in hot reload tests), unregister first to ensure cleanliness
	prometheus.Unregister(counter)
	err := prometheus.Register(counter)
	require.NoError(t, err)
	defer prometheus.Unregister(counter)

	// Record values
	counter.Add(10)

	// 2. Initialize our Bridge (Bifrost code)
	bridge := NewBridge()
	require.NotNil(t, bridge)

	// 3. Use OTel SDK's Manual Reader to intercept data
	reader := sdkmetric.NewManualReader(sdkmetric.WithProducer(bridge))

	// Fix: Reader must be registered with MeterProvider before Collect can be called
	_ = sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	// 4. Force collection
	var rm metricdata.ResourceMetrics
	err = reader.Collect(context.Background(), &rm)
	assert.NoError(t, err)

	// 5. Assertion verification: Check if OTel's data structure contains the recorded metrics
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "bifrost_push_smoke_test_total" {
				found = true
				// Verify if numerical values are correctly converted
				data, ok := m.Data.(metricdata.Sum[float64])
				assert.True(t, ok)
				assert.Equal(t, float64(10), data.DataPoints[0].Value)
			}
		}
	}

	assert.True(t, found, "Metric recorded in Prometheus should be visible in OTel via Bridge")
}
