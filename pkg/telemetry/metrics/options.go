package metrics

import (
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var defaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// Option defines the configuration options for the metrics tracer.
type Option interface {
	apply(cfg *promConfig)
}

type option func(cfg *promConfig)

func (fn option) apply(cfg *promConfig) {
	fn(cfg)
}

type promConfig struct {
	buckets            []float64
	runtimeMetricRules []collectors.GoRuntimeMetricsRule
	enableGoCollector  bool
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
