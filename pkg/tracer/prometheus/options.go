package prometheus

import (
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var defaultBuckets = []float64{0.005000, 0.010000, 0.025000, 0.050000, 0.10000, 0.250000, 0.500000, 1.00000, 2.50000, 5.000000, 10.000000}

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
	runtimeMetricRules []collectors.GoRuntimeMetricsRule
	disableServer      bool
}

func defaultConfig() *promConfig {
	return &promConfig{
		buckets:           defaultBuckets,
		enableGoCollector: false,
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
