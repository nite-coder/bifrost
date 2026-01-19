package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	otelprom "go.opentelemetry.io/contrib/bridges/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// NewBridge creates a new OTel Metric Producer that bridges existing Prometheus metrics.
// It collects from prometheus.DefaultGatherer.
func NewBridge() sdkmetric.Producer {
	return otelprom.NewMetricProducer(
		otelprom.WithGatherer(prometheus.DefaultGatherer),
	)
}
