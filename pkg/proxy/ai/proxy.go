package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	hertzresp "github.com/cloudwego/hertz/pkg/protocol/http1/resp"

	"github.com/nite-coder/bifrost/pkg/ai"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/proxy"
	"github.com/nite-coder/bifrost/pkg/telemetry/metrics"
	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/bifrost/pkg/variable"
)

const (
	// TargetPartsCount is the expected number of parts when splitting target (provider/model).
	TargetPartsCount = 2
	// StreamReadBufferSize is the buffer size for reading SSE streams.
	StreamReadBufferSize = 4096
)

// Proxy implements proxy.Proxy for LLM upstream connections.
type Proxy struct {
	id             string
	target         string // e.g. "provider/model"
	weight         uint32
	options        *config.AIOptions
	pricing        *config.AIPricingOptions
	httpClient     *client.Client
	adapter        ai.LLMAdapter
	metricsEnabled bool
	upstreamHost   string
	initOnce       sync.Once
	initErr        error
}

var _ proxy.Proxy = (*Proxy)(nil)

// ProxyOptions contains configuration options for creating a new AI Proxy.
type ProxyOptions struct {
	ID             string
	Target         string
	Weight         uint32
	AIOptions      *config.AIOptions
	MetricsEnabled bool
	Pricing        *config.AIPricingOptions
}

// NewProxy creates a new AIProxy instance.
func NewProxy(opts ProxyOptions) *Proxy {
	p := &Proxy{
		id:             opts.ID,
		target:         opts.Target,
		weight:         opts.Weight,
		options:        opts.AIOptions,
		metricsEnabled: opts.MetricsEnabled,
		pricing:        opts.Pricing,
	}

	parts := strings.SplitN(opts.Target, "/", TargetPartsCount)
	if len(parts) == TargetPartsCount && opts.AIOptions != nil && opts.AIOptions.Providers != nil {
		if prov, ok := opts.AIOptions.Providers[parts[0]]; ok && prov.BaseURL != "" {
			if addr, err := url.Parse(prov.BaseURL); err == nil {
				p.upstreamHost = addr.Host
			}
		}
	}

	return p
}

// ID returns the unique identifier for the proxy.
func (p *Proxy) ID() string {
	return p.id
}

// Target returns the backend target (provider/model format).
func (p *Proxy) Target() string {
	return p.target
}

// Weight returns the load balancing weight.
func (p *Proxy) Weight() uint32 {
	return p.weight
}

// IsAvailable returns true if the proxy is active and healthy.
func (p *Proxy) IsAvailable() bool {
	return true
}

// AddFailedCount logs a request failure.
func (p *Proxy) AddFailedCount(_ uint) error {
	return nil
}

// ServeHTTP handles incoming HTTP requests and proxies them to the target LLM provider.
func (p *Proxy) ServeHTTP(ctx context.Context, hzCtx *app.RequestContext) {
	clientAdapterVal, ok := hzCtx.Get(ai.ContextKeyClientAdapter)
	if !ok {
		hzCtx.SetStatusCode(http.StatusInternalServerError)
		return
	}
	clientAdapter, ok := clientAdapterVal.(ai.ClientAdapter)
	if !ok {
		hzCtx.SetStatusCode(http.StatusInternalServerError)
		return
	}

	familyVal, ok := hzCtx.Get(ai.ContextKeyAIFamily)
	if !ok {
		familyVal = ai.FamilyChat
	}
	aiFamily, ok := familyVal.(string)
	if !ok {
		aiFamily = ai.FamilyChat
	}

	virtualModelVal, ok := hzCtx.Get(ai.ContextKeyVirtualModelName)
	if !ok {
		hzCtx.SetStatusCode(http.StatusInternalServerError)
		return
	}
	virtualModel, ok := virtualModelVal.(string)
	if !ok {
		hzCtx.SetStatusCode(http.StatusInternalServerError)
		return
	}

	parts := strings.SplitN(p.target, "/", TargetPartsCount)
	if len(parts) != TargetPartsCount {
		_ = hzCtx.Error(&ai.AIError{
			Type:       "api_error",
			Message:    "invalid target format",
			StatusCode: http.StatusInternalServerError,
		})
		return
	}
	targetModel := parts[1]

	if p.upstreamHost != "" {
		hzCtx.Set(variable.UpstreamRequestHost, p.upstreamHost)
	}

	hzCtx.Set(variable.ModelID, p.target)

	adapter, err := p.LLMAdapter()
	if err != nil {
		_ = hzCtx.Error(&ai.AIError{
			Type:       "provider_error",
			Message:    err.Error(),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	switch aiFamily {
	case ai.FamilyChat:
		reqVal, ok := hzCtx.Get(ai.ContextKeyChatRequest)
		if !ok {
			hzCtx.SetStatusCode(http.StatusBadRequest)
			return
		}
		chatReq, ok := reqVal.(*ai.ChatRequest)
		if !ok {
			hzCtx.SetStatusCode(http.StatusBadRequest)
			return
		}
		chatReq.Model = targetModel

		if chatReq.Stream {
			if chatReq.StreamOptions == nil {
				chatReq.StreamOptions = &ai.StreamOptions{}
			}
			chatReq.StreamOptions.IncludeUsage = true
			p.handleChatStream(ctx, hzCtx, chatReq, adapter, clientAdapter, virtualModel, p.target)
		} else {
			p.handleChatUnary(ctx, hzCtx, chatReq, adapter, clientAdapter, virtualModel, p.target)
		}
	case ai.FamilyResponses:
		reqVal, ok := hzCtx.Get(ai.ContextKeyResponsesRequest)
		if !ok {
			hzCtx.SetStatusCode(http.StatusBadRequest)
			return
		}
		respReq, ok := reqVal.(*ai.ResponsesRequest)
		if !ok {
			hzCtx.SetStatusCode(http.StatusBadRequest)
			return
		}
		respReq.Model = targetModel

		p.handleResponses(ctx, hzCtx, respReq, adapter, clientAdapter, virtualModel, p.target)
	default:
		hzCtx.SetStatusCode(http.StatusInternalServerError)
		return
	}
}

// Tag retrieves a tag value.
func (p *Proxy) Tag(_ string) (value string, exist bool) {
	return "", false
}

// Tags returns all tags.
func (p *Proxy) Tags() map[string]string {
	return nil
}

// Close releases resources.
func (p *Proxy) Close() error {
	return nil
}

// LLMAdapter returns the initialized LLMAdapter for the proxy.
func (p *Proxy) LLMAdapter() (ai.LLMAdapter, error) {
	p.initOnce.Do(func() {
		parts := strings.Split(p.target, "/")
		if len(parts) != TargetPartsCount {
			p.initErr = fmt.Errorf("ai: invalid target format '%s'", p.target)
			return
		}
		providerID := parts[0]

		if p.options == nil || p.options.Providers == nil {
			p.initErr = errors.New("ai: providers configuration is missing")
			return
		}

		providerOptions, exists := p.options.Providers[providerID]
		if !exists {
			p.initErr = fmt.Errorf("ai: provider '%s' not found", providerID)
			return
		}

		if p.httpClient == nil {
			c, err := client.NewClient(client.WithResponseBodyStream(true))
			if err != nil {
				p.initErr = fmt.Errorf("ai: failed to create http client: %w", err)
				return
			}
			p.httpClient = c
		}

		opts := ai.LLMAdapterOptions{
			HTTPClient: p.httpClient,
			APIKey:     variable.GetString(providerOptions.APIKey, nil),
			BaseURL:    providerOptions.BaseURL,
		}
		adapter, err := ai.GetAdapter(providerOptions.Handler, opts)
		if err != nil {
			p.initErr = err
			return
		}
		p.adapter = adapter
	})
	return p.adapter, p.initErr
}

func (p *Proxy) handleChatUnary(
	ctx context.Context,
	hzCtx *app.RequestContext,
	chatReq *ai.ChatRequest,
	adapter ai.LLMAdapter,
	clientAdapter ai.ClientAdapter,
	virtualModel string,
	modelID string,
) {
	startTime := timecache.Now()

	resp, err := adapter.Chat(ctx, chatReq)
	if err != nil {
		var aiErr *ai.AIError
		if errors.As(err, &aiErr) {
			_ = hzCtx.Error(aiErr)
		} else {
			_ = hzCtx.Error(&ai.AIError{
				Type:       "api_error",
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
		}
		return
	}

	endTime := timecache.Now()
	durationSecs := endTime.Sub(startTime).Seconds()

	// Calculate cost
	resp.Usage.CalculateCost(p.pricing)

	// Record Prometheus Metrics
	if p.metricsEnabled {
		metrics.AIRequestDuration.WithLabelValues(virtualModel, modelID).Observe(durationSecs)
		metrics.AIInputTokens.WithLabelValues(virtualModel, modelID).Add(float64(resp.Usage.PromptTokens))
		if resp.Usage.PromptTokensDetails != nil && resp.Usage.PromptTokensDetails.CachedTokens > 0 {
			metrics.AIInputCachedTokens.WithLabelValues(virtualModel, modelID).
				Add(float64(resp.Usage.PromptTokensDetails.CachedTokens))
		}
		metrics.AIOutputTokens.WithLabelValues(virtualModel, modelID).Add(float64(resp.Usage.CompletionTokens))
		if resp.Usage.CompletionTokensDetails != nil && resp.Usage.CompletionTokensDetails.ReasoningTokens > 0 {
			metrics.AIOutputReasoningTokens.WithLabelValues(virtualModel, modelID).
				Add(float64(resp.Usage.CompletionTokensDetails.ReasoningTokens))
		}
		metrics.AITotalTokens.WithLabelValues(virtualModel, modelID).Add(float64(resp.Usage.TotalTokens))
		metrics.AIRequestCost.WithLabelValues(virtualModel, modelID).Add(resp.Usage.InputCost + resp.Usage.OutputCost)
	}

	// Mask model name in response
	resp.Model = virtualModel

	// Translate canonical response to client format
	clientResp, err := clientAdapter.ToClientChatResponse(resp)
	if err != nil {
		_ = hzCtx.Error(&ai.AIError{
			Type:       "api_error",
			Message:    "failed to translate response to client format: " + err.Error(),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	hzCtx.JSON(http.StatusOK, clientResp)
}

type streamUsageObserver struct {
	onUsage func(metadata ai.UsageMetadata, usage ai.Usage)
}

func (o *streamUsageObserver) OnUsage(_ context.Context, metadata ai.UsageMetadata, usage ai.Usage) {
	o.onUsage(metadata, usage)
}

func (p *Proxy) handleChatStream(
	ctx context.Context,
	hzCtx *app.RequestContext,
	chatReq *ai.ChatRequest,
	adapter ai.LLMAdapter,
	clientAdapter ai.ClientAdapter,
	virtualModel string,
	modelID string,
) {
	startTime := timecache.Now()

	stream, err := adapter.StreamChat(ctx, chatReq)
	if err != nil {
		var aiErr *ai.AIError
		if errors.As(err, &aiErr) {
			_ = hzCtx.Error(aiErr)
		} else {
			_ = hzCtx.Error(&ai.AIError{
				Type:       "api_error",
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
		}
		return
	}

	metadata := ai.UsageMetadata{
		Model:    virtualModel,
		Provider: modelID,
	}

	var totalCompletionTokens int
	var usageReceived bool

	observer := &streamUsageObserver{
		onUsage: func(_ ai.UsageMetadata, u ai.Usage) {
			totalCompletionTokens = u.TotalTokens - u.PromptTokens
			if u.CompletionTokens > 0 {
				totalCompletionTokens = u.CompletionTokens
			}
			usageReceived = true

			// Calculate cost
			u.CalculateCost(p.pricing)

			if p.metricsEnabled {
				metrics.AIInputTokens.WithLabelValues(metadata.Model, metadata.Provider).Add(float64(u.PromptTokens))
				if u.PromptTokensDetails != nil && u.PromptTokensDetails.CachedTokens > 0 {
					metrics.AIInputCachedTokens.WithLabelValues(metadata.Model, metadata.Provider).
						Add(float64(u.PromptTokensDetails.CachedTokens))
				}
				metrics.AIOutputTokens.WithLabelValues(metadata.Model, metadata.Provider).
					Add(float64(totalCompletionTokens))
				if u.CompletionTokensDetails != nil && u.CompletionTokensDetails.ReasoningTokens > 0 {
					metrics.AIOutputReasoningTokens.WithLabelValues(metadata.Model, metadata.Provider).
						Add(float64(u.CompletionTokensDetails.ReasoningTokens))
				}
				metrics.AITotalTokens.WithLabelValues(metadata.Model, metadata.Provider).Add(float64(u.TotalTokens))
				metrics.AIRequestCost.WithLabelValues(metadata.Model, metadata.Provider).Add(u.InputCost + u.OutputCost)
			}
		},
	}

	observedStream := ai.NewObservedStream(stream, observer, metadata)
	finalStream := clientAdapter.StreamConverter(observedStream)
	defer finalStream.Close()

	hzCtx.SetStatusCode(http.StatusOK)
	hzCtx.Response.Header.SetContentType("text/event-stream")
	hzCtx.Response.Header.Set("Cache-Control", "no-cache")
	hzCtx.Response.Header.Set("Connection", "keep-alive")
	hzCtx.Response.Header.Set("X-Accel-Buffering", "no")

	// Use chunked body writer for real-time streaming when available
	if w := hzCtx.GetWriter(); w != nil {
		hzCtx.Response.HijackWriter(hertzresp.NewChunkedBodyWriter(&hzCtx.Response, w))
	}
	_ = hzCtx.Flush()

	firstByteOnce := sync.Once{}
	var firstByteTime time.Time

	buf := make([]byte, StreamReadBufferSize)
	for {
		n, err := finalStream.Read(buf)
		if n > 0 {
			firstByteOnce.Do(func() {
				firstByteTime = timecache.Now()
				ttfb := firstByteTime.Sub(startTime).Seconds()
				if p.metricsEnabled {
					metrics.AIRequestTTFB.WithLabelValues(virtualModel, modelID).Observe(ttfb)
				}
			})

			_, writeErr := hzCtx.Write(buf[:n])
			if writeErr != nil {
				return
			}
			_ = hzCtx.Flush()
		}

		if err != nil {
			if err == io.EOF {
				break
			}

			// Mid-stream error
			var sseErrBytes []byte
			var aiErr *ai.AIError
			if errors.As(err, &aiErr) {
				formattedErr, _ := clientAdapter.ToClientError(aiErr)
				if errJSON, errMar := sonic.Marshal(formattedErr); errMar == nil {
					sseErrBytes = fmt.Appendf(nil, "data: %s\n\n", errJSON)
				}
			} else {
				// SECURITY: Log full error details internally, return generic error to client
				routeID := variable.GetString(variable.RouteID, hzCtx)
				slog.ErrorContext(ctx, "stream mid-error intercepted",
					"route_id", routeID,
					"virtual_model", virtualModel,
					"model_id", modelID,
					"error", err.Error(),
				)

				genericErr := &ai.AIError{
					Type:       "internal_error",
					Message:    "Internal server error",
					StatusCode: http.StatusBadGateway,
				}
				formattedErr, _ := clientAdapter.ToClientError(genericErr)
				if errJSON, errMar := sonic.Marshal(formattedErr); errMar == nil {
					sseErrBytes = fmt.Appendf(nil, "data: %s\n\n", errJSON)
				}
			}
			if len(sseErrBytes) == 0 {
				sseErrBytes = []byte("data: {\"error\":{\"message\":\"internal error\"}}\n\n")
			}
			_, _ = hzCtx.Write(sseErrBytes)
			_, _ = hzCtx.Write([]byte("data: [DONE]\n\n"))
			_ = hzCtx.Flush()
			return
		}
	}

	endTime := timecache.Now()
	durationSecs := endTime.Sub(startTime).Seconds()
	if p.metricsEnabled {
		metrics.AIRequestDuration.WithLabelValues(virtualModel, modelID).Observe(durationSecs)
	}

	if usageReceived && totalCompletionTokens > 0 {
		generationDuration := endTime.Sub(firstByteTime).Seconds()
		if generationDuration > 0 {
			tps := float64(totalCompletionTokens) / generationDuration
			if p.metricsEnabled {
				metrics.AIGenerationTPS.WithLabelValues(virtualModel, modelID).Observe(tps)
			}
		}
	}
}

func (p *Proxy) handleResponses(
	ctx context.Context,
	hzCtx *app.RequestContext,
	req *ai.ResponsesRequest,
	adapter ai.LLMAdapter,
	clientAdapter ai.ClientAdapter,
	virtualModel string,
	modelID string,
) {
	startTime := timecache.Now()

	resp, err := adapter.Responses(ctx, req)
	if err != nil {
		var aiErr *ai.AIError
		if errors.As(err, &aiErr) {
			_ = hzCtx.Error(aiErr)
		} else {
			_ = hzCtx.Error(&ai.AIError{
				Type:       "api_error",
				Message:    err.Error(),
				StatusCode: http.StatusInternalServerError,
			})
		}
		return
	}

	endTime := timecache.Now()
	durationSecs := endTime.Sub(startTime).Seconds()

	// Calculate cost
	resp.Usage.CalculateCost(p.pricing)

	// Record Prometheus Metrics
	if p.metricsEnabled {
		metrics.AIRequestDuration.WithLabelValues(virtualModel, modelID).Observe(durationSecs)
		metrics.AIInputTokens.WithLabelValues(virtualModel, modelID).Add(float64(resp.Usage.PromptTokens))
		if resp.Usage.PromptTokensDetails != nil && resp.Usage.PromptTokensDetails.CachedTokens > 0 {
			metrics.AIInputCachedTokens.WithLabelValues(virtualModel, modelID).
				Add(float64(resp.Usage.PromptTokensDetails.CachedTokens))
		}
		metrics.AIOutputTokens.WithLabelValues(virtualModel, modelID).Add(float64(resp.Usage.CompletionTokens))
		if resp.Usage.CompletionTokensDetails != nil && resp.Usage.CompletionTokensDetails.ReasoningTokens > 0 {
			metrics.AIOutputReasoningTokens.WithLabelValues(virtualModel, modelID).
				Add(float64(resp.Usage.CompletionTokensDetails.ReasoningTokens))
		}
		metrics.AITotalTokens.WithLabelValues(virtualModel, modelID).Add(float64(resp.Usage.TotalTokens))
		metrics.AIRequestCost.WithLabelValues(virtualModel, modelID).Add(resp.Usage.InputCost + resp.Usage.OutputCost)
	}

	// Mask model name in response
	resp.Model = virtualModel

	clientResp, err := clientAdapter.ToClientResponsesResponse(resp)
	if err != nil {
		_ = hzCtx.Error(&ai.AIError{
			Type:       "api_error",
			Message:    "failed to translate response to client format: " + err.Error(),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	hzCtx.JSON(http.StatusOK, clientResp)
}
