package gateway

import (
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestTracer(t *testing.T) {

	options := config.TracingOptions{
		OTLP: config.OTLPOptions{
			Enabled:      true,
			Propagators:  []string{"tracecontext", "baggage"},
			Endpoint:     "http://localhost:4317",
			Insecure:     false,
			SamplingRate: 1,
			BatchSize:    500,
			QueueSize:    50000,
			Flush:        10 * time.Second,
		},
	}

	tracer, err := initTracerProvider(options)
	assert.NoError(t, err)
	assert.NotNil(t, tracer)
}
