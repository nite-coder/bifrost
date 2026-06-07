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

	"github.com/nite-coder/bifrost/pkg/ai"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/proxy"
	"github.com/nite-coder/bifrost/pkg/telemetry/metrics"
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
	httpClient     *client.Client
	adapter        ai.LLMAdapter
	metricsEnabled bool
	upstreamHost   string
	initOnce       sync.Once
	initErr        error
}

var _ proxy.Proxy = (*Proxy)(nil)

// NewProxy creates a new AIProxy instance.
func NewProxy(id string, target string, weight uint32, options *config.AIOptions, metricsEnabled bool) *Proxy {
	p := &Proxy{
		id:             id,
		target:         target,
		weight:         weight,
		options:        options,
		metricsEnabled: metricsEnabled,
	}

	parts := strings.SplitN(target, "/", TargetPartsCount)
	if len(parts) == TargetPartsCount && options != nil && options.Providers != nil {
		if prov, ok := options.Providers[parts[0]]; ok && prov.BaseURL != "" {
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
	providerID := parts[0]
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
			p.handleChatStream(ctx, hzCtx, chatReq, adapter, clientAdapter, virtualModel, providerID)
		} else {
			p.handleChatUnary(ctx, hzCtx, chatReq, adapter, clientAdapter, virtualModel, providerID)
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

		p.handleResponses(ctx, hzCtx, respReq, adapter, clientAdapter, virtualModel, providerID)
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

		provOpts, exists := p.options.Providers[providerID]
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
			APIKey:     provOpts.APIKey,
			BaseURL:    provOpts.BaseURL,
		}
		adapter, err := ai.GetAdapter(provOpts.Handler, opts)
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
	providerID string,
) {
	startTime := time.Now()

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

	endTime := time.Now()
	durationSecs := endTime.Sub(startTime).Seconds()

	// Record Prometheus Metrics
	if p.metricsEnabled {
		metrics.RequestDuration.WithLabelValues(virtualModel, providerID).Observe(durationSecs)
		metrics.PromptTokens.WithLabelValues(virtualModel, providerID).Add(float64(resp.Usage.PromptTokens))
		metrics.CompletionTokens.WithLabelValues(virtualModel, providerID).Add(float64(resp.Usage.CompletionTokens))
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
	providerID string,
) {
	startTime := time.Now()

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
	defer stream.Close()

	metadata := ai.UsageMetadata{
		Model:    virtualModel,
		Provider: providerID,
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

			if p.metricsEnabled {
				metrics.PromptTokens.WithLabelValues(metadata.Model, metadata.Provider).Add(float64(u.PromptTokens))
				metrics.CompletionTokens.WithLabelValues(metadata.Model, metadata.Provider).
					Add(float64(totalCompletionTokens))
			}
		},
	}

	observedStream := ai.NewObservedStream(stream, observer, metadata)
	finalStream := clientAdapter.WrapEgressStream(observedStream)
	defer finalStream.Close()

	hzCtx.SetStatusCode(http.StatusOK)
	hzCtx.Response.Header.SetContentType("text/event-stream")
	hzCtx.Response.Header.Set("Cache-Control", "no-cache")
	hzCtx.Response.Header.Set("Connection", "keep-alive")
	hzCtx.Response.Header.Set("X-Accel-Buffering", "no")
	_ = hzCtx.Flush()

	firstByteOnce := sync.Once{}
	var firstByteTime time.Time

	buf := make([]byte, StreamReadBufferSize)
	for {
		n, err := finalStream.Read(buf)
		if n > 0 {
			firstByteOnce.Do(func() {
				firstByteTime = time.Now()
				ttfb := firstByteTime.Sub(startTime).Seconds()
				if p.metricsEnabled {
					metrics.RequestTTFB.WithLabelValues(virtualModel, providerID).Observe(ttfb)
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
					sseErrBytes = fmt.Appendf(nil, "data: %s\n\n", string(errJSON))
				}
			} else {
				// SECURITY: Log full error details internally, return generic error to client
				routeID := variable.GetString(variable.RouteID, hzCtx)
				slog.ErrorContext(ctx, "stream mid-error intercepted",
					"route_id", routeID,
					"virtual_model", virtualModel,
					"provider", providerID,
					"error", err.Error(),
				)

				genericErr := &ai.AIError{
					Type:       "internal_error",
					Message:    "Internal server error",
					StatusCode: http.StatusBadGateway,
				}
				formattedErr, _ := clientAdapter.ToClientError(genericErr)
				if errJSON, errMar := sonic.Marshal(formattedErr); errMar == nil {
					sseErrBytes = fmt.Appendf(nil, "data: %s\n\n", string(errJSON))
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

	endTime := time.Now()
	durationSecs := endTime.Sub(startTime).Seconds()
	if p.metricsEnabled {
		metrics.RequestDuration.WithLabelValues(virtualModel, providerID).Observe(durationSecs)
	}

	if usageReceived && totalCompletionTokens > 0 {
		generationDuration := endTime.Sub(firstByteTime).Seconds()
		if generationDuration > 0 {
			tps := float64(totalCompletionTokens) / generationDuration
			if p.metricsEnabled {
				metrics.GenerationTPS.WithLabelValues(virtualModel, providerID).Observe(tps)
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
	providerID string,
) {
	startTime := time.Now()

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

	endTime := time.Now()
	durationSecs := endTime.Sub(startTime).Seconds()

	// Record Prometheus Metrics
	if p.metricsEnabled {
		metrics.RequestDuration.WithLabelValues(virtualModel, providerID).Observe(durationSecs)
		metrics.PromptTokens.WithLabelValues(virtualModel, providerID).Add(float64(resp.Usage.PromptTokens))
		metrics.CompletionTokens.WithLabelValues(virtualModel, providerID).Add(float64(resp.Usage.CompletionTokens))
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
