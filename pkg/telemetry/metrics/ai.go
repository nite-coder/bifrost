package metrics

import prom "github.com/prometheus/client_golang/prometheus"

var (
	// RequestTTFB represents the AI model thinking speed (TTFB) in seconds.
	RequestTTFB *prom.HistogramVec
	// RequestDuration represents the total AI request duration in seconds.
	RequestDuration *prom.HistogramVec
	// GenerationTPS represents the AI generation tokens per second rate.
	GenerationTPS *prom.HistogramVec
	// PromptTokens represents the total prompt tokens consumed.
	PromptTokens *prom.CounterVec
	// CompletionTokens represents the total completion tokens consumed.
	CompletionTokens *prom.CounterVec
)

func init() {
	RequestTTFB = prom.NewHistogramVec(
		prom.HistogramOpts{
			Name:    "bifrost_ai_request_ttfb_seconds",
			Help:    "AI model thinking speed (TTFB) in seconds",
			Buckets: prom.DefBuckets,
		},
		[]string{"model", "provider"},
	)
	prom.MustRegister(RequestTTFB)

	RequestDuration = prom.NewHistogramVec(
		prom.HistogramOpts{
			Name:    "bifrost_ai_request_duration_seconds",
			Help:    "Total AI request duration in seconds",
			Buckets: prom.DefBuckets,
		},
		[]string{"model", "provider"},
	)
	prom.MustRegister(RequestDuration)

	GenerationTPS = prom.NewHistogramVec(
		prom.HistogramOpts{
			Name:    "bifrost_ai_generation_tps",
			Help:    "AI generation tokens per second rate",
			Buckets: prom.DefBuckets,
		},
		[]string{"model", "provider"},
	)
	prom.MustRegister(GenerationTPS)

	PromptTokens = prom.NewCounterVec(
		prom.CounterOpts{
			Name: "bifrost_ai_prompt_tokens_total",
			Help: "Total prompt tokens consumed",
		},
		[]string{"model", "provider"},
	)
	prom.MustRegister(PromptTokens)

	CompletionTokens = prom.NewCounterVec(
		prom.CounterOpts{
			Name: "bifrost_ai_completion_tokens_total",
			Help: "Total completion tokens consumed",
		},
		[]string{"model", "provider"},
	)
	prom.MustRegister(CompletionTokens)
}
