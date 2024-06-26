package prometheus

import (
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var defaultBuckets = []float64{0.100000, 0.300000, 1.200000, 5.000000}

// Option opts for monitor prometheus
type Option interface {
	apply(cfg *promConfig)
}

type option func(cfg *promConfig)

func (fn option) apply(cfg *promConfig) {
	fn(cfg)
}

type promConfig struct {
	buckets            []float64
	enableGoCollector  bool
	registry           *prom.Registry
	runtimeMetricRules []collectors.GoRuntimeMetricsRule
	disableServer      bool
}

func defaultConfig() *promConfig {
	return &promConfig{
		buckets:           defaultBuckets,
		enableGoCollector: false,
		registry:          prom.NewRegistry(),
		disableServer:     false,
	}
}

// WithEnableGoCollector enable go collector
func WithEnableGoCollector(enable bool) Option {
	return option(func(cfg *promConfig) {
		cfg.enableGoCollector = enable
	})
}

// WithGoCollectorRule define your custom go collector rule
func WithGoCollectorRule(rules ...collectors.GoRuntimeMetricsRule) Option {
	return option(func(cfg *promConfig) {
		cfg.runtimeMetricRules = rules
	})
}

// WithDisableServer disable prometheus server
func WithDisableServer(disable bool) Option {
	return option(func(cfg *promConfig) {
		cfg.disableServer = disable
	})
}

// WithHistogramBuckets define your custom histogram buckets base on your biz
func WithHistogramBuckets(buckets []float64) Option {
	return option(func(cfg *promConfig) {
		if len(buckets) > 0 {
			cfg.buckets = buckets
		}
	})
}

// WithRegistry define your custom registry
func WithRegistry(registry *prom.Registry) Option {
	return option(func(cfg *promConfig) {
		if registry != nil {
			cfg.registry = registry
		}
	})
}
