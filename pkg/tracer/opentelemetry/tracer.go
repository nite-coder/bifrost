package opentelemetry

import "http-benchmark/pkg/config"

type Tracer struct {
}

func NewTracer(opts config.TracingOptions) *Tracer {
	return &Tracer{}
}
