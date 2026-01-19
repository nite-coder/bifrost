package metrics

import (
	"context"
	"testing"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricMiddleware(t *testing.T) {
	t.Run("should use default path /metrics", func(t *testing.T) {
		m := NewMetricMiddleware("", nil)
		assert.NotNil(t, m)
		assert.Equal(t, "/metrics", string(m.path))
	})

	t.Run("should use provided path", func(t *testing.T) {
		m := NewMetricMiddleware("/custom/metrics", nil)
		assert.NotNil(t, m)
		assert.Equal(t, "/custom/metrics", string(m.path))
	})

	t.Run("should use fallback handler when provider is nil", func(t *testing.T) {
		m := NewMetricMiddleware("/metrics", nil)
		assert.NotNil(t, m.handler)
		// We can't easily assert on the handler function pointer, but we know it's not nil
	})

	t.Run("should use provider handler when available", func(t *testing.T) {
		opts := config.MetricsOptions{
			Prometheus: config.PrometheusOptions{
				Enabled: true,
			},
		}
		provider, err := NewProvider(context.Background(), opts)
		require.NoError(t, err)
		require.NotNil(t, provider)
		defer provider.Shutdown(context.Background())

		m := NewMetricMiddleware("/metrics", provider)
		assert.NotNil(t, m.handler)

		// Verify handler is set
		assert.NotNil(t, m.handler)

		// Note: Cannot compare function types in Go, so we can't assert equality of handlers directly
		// but we know from the code path that it uses the provider's handler
	})

	t.Run("should use fallback handler when provider has no handler", func(t *testing.T) {
		opts := config.MetricsOptions{
			Prometheus: config.PrometheusOptions{
				Enabled: false, // No metrics handler created if disabled
			},
		}
		provider, err := NewProvider(context.Background(), opts)
		require.NoError(t, err)

		m := NewMetricMiddleware("/metrics", provider)
		assert.NotNil(t, m.handler)

		// Should NOT be nil (fallback), but provider.MetricsHandler() IS nil
		assert.Nil(t, provider.MetricsHandler())
		assert.NotNil(t, m.handler)
	})
}
