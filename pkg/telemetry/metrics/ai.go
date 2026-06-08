package metrics

import prom "github.com/prometheus/client_golang/prometheus"

var (
	// AIRequestTTFB represents the AI model thinking speed (TTFB) in seconds.
	AIRequestTTFB *prom.HistogramVec
	// AIRequestDuration represents the total AI request duration in seconds.
	AIRequestDuration *prom.HistogramVec
	// AIGenerationTPS represents the AI generation tokens per second rate.
	AIGenerationTPS *prom.HistogramVec
	// AIInputTokens represents the total prompt tokens consumed.
	AIInputTokens *prom.CounterVec
	// AIInputCachedTokens represents the total cached prompt tokens consumed.
	AIInputCachedTokens *prom.CounterVec
	// AIOutputTokens represents the total completion tokens consumed.
	AIOutputTokens *prom.CounterVec
	// AIOutputReasoningTokens represents the total reasoning tokens consumed.
	AIOutputReasoningTokens *prom.CounterVec
	// AITotalTokens represents the total tokens consumed (input + output).
	AITotalTokens *prom.CounterVec
)

// InitAI initializes AI-related Prometheus metrics with custom or default buckets.
func InitAI(latencyBuckets, tpsBuckets []float64) {
	if len(latencyBuckets) == 0 {
		latencyBuckets = prom.DefBuckets
	}
	if len(tpsBuckets) == 0 {
		tpsBuckets = defaultAITPSBuckets
	}

	AIRequestTTFB = prom.NewHistogramVec(
		prom.HistogramOpts{
			Name:    "bifrost_ai_request_ttfb_seconds",
			Help:    "AI model thinking speed (TTFB) in seconds",
			Buckets: latencyBuckets,
		},
		[]string{"model", "model_id"},
	)
	prom.MustRegister(AIRequestTTFB)

	AIRequestDuration = prom.NewHistogramVec(
		prom.HistogramOpts{
			Name:    "bifrost_ai_request_duration_seconds",
			Help:    "Total AI request duration in seconds",
			Buckets: latencyBuckets,
		},
		[]string{"model", "model_id"},
	)
	prom.MustRegister(AIRequestDuration)

	AIGenerationTPS = prom.NewHistogramVec(
		prom.HistogramOpts{
			Name:    "bifrost_ai_generation_tps",
			Help:    "AI generation tokens per second rate",
			Buckets: tpsBuckets,
		},
		[]string{"model", "model_id"},
	)
	prom.MustRegister(AIGenerationTPS)

	AIInputTokens = prom.NewCounterVec(
		prom.CounterOpts{
			Name: "bifrost_ai_input_tokens_total",
			Help: "Total prompt tokens consumed",
		},
		[]string{"model", "model_id"},
	)
	prom.MustRegister(AIInputTokens)

	AIInputCachedTokens = prom.NewCounterVec(
		prom.CounterOpts{
			Name: "bifrost_ai_input_cached_tokens_total",
			Help: "Total cached prompt tokens consumed",
		},
		[]string{"model", "model_id"},
	)
	prom.MustRegister(AIInputCachedTokens)

	AIOutputTokens = prom.NewCounterVec(
		prom.CounterOpts{
			Name: "bifrost_ai_output_tokens_total",
			Help: "Total completion tokens consumed",
		},
		[]string{"model", "model_id"},
	)
	prom.MustRegister(AIOutputTokens)

	AIOutputReasoningTokens = prom.NewCounterVec(
		prom.CounterOpts{
			Name: "bifrost_ai_output_reasoning_tokens_total",
			Help: "Total reasoning tokens consumed",
		},
		[]string{"model", "model_id"},
	)
	prom.MustRegister(AIOutputReasoningTokens)

	AITotalTokens = prom.NewCounterVec(
		prom.CounterOpts{
			Name: "bifrost_ai_total_tokens_total",
			Help: "Total tokens consumed (input + output)",
		},
		[]string{"model", "model_id"},
	)
	prom.MustRegister(AITotalTokens)
}
