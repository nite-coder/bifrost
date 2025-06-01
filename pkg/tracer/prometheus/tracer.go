package prometheus

import (
	"context"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/tracer"
	"github.com/cloudwego/hertz/pkg/common/tracer/stats"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/nite-coder/blackbear/pkg/cast"
	prom "github.com/prometheus/client_golang/prometheus"
)

const (
	labelServerID     = "server_id"
	labelMethod       = "method"
	labelPath         = "path"
	labelRouteID      = "route_id"
	labelServiceID    = "service_id"
	labelStatusCode   = "status_code"
	unknownLabelValue = "unknown"
)

// genRequestDurationLabels make labels values.
func genRequestDurationLabels(c *app.RequestContext) prom.Labels {
	labels := make(prom.Labels)

	serverID := variable.GetString(variable.ServerID, c)
	routeID := variable.GetString(variable.RouteID, c)
	serviceID := variable.GetString(variable.ServiceID, c)
	method := variable.GetString(variable.HTTPRequestMethod, c)

	labels[labelServerID] = defaultValIfEmpty(serverID, unknownLabelValue)
	labels[labelRouteID] = defaultValIfEmpty(routeID, unknownLabelValue)
	labels[labelServiceID] = defaultValIfEmpty(serviceID, unknownLabelValue)
	labels[labelMethod] = defaultValIfEmpty(method, unknownLabelValue) 
	labels[labelStatusCode] = defaultValIfEmpty(strconv.Itoa(c.Response.Header.StatusCode()), unknownLabelValue)

	path := variable.GetString(variable.HTTPRoute, c)
	if path == "" {
		path = variable.GetString(variable.HTTPRequestPath, c)
		if path == "" {
			path = cast.B2S(c.Request.Path())
		}
	}
	labels[labelPath] = defaultValIfEmpty(path, unknownLabelValue)

	return labels
}

type serverTracer struct {
	httpServerRequestBodySize  *prom.CounterVec
	httpServerResponseBodySize *prom.CounterVec
	httpServerRequests         *prom.CounterVec
	httpServerActiveRequests   prom.Gauge
	httpServerRequestDuration  *prom.HistogramVec
	httpBifrostRequestDuration *prom.HistogramVec
}

// Start record the beginning of server handling request from client.
func (s *serverTracer) Start(ctx context.Context, c *app.RequestContext) context.Context {
	s.httpServerActiveRequests.Inc()
	return ctx
}

// Finish record the ending of server handling request from client.
func (s *serverTracer) Finish(ctx context.Context, c *app.RequestContext) {
	if c.GetTraceInfo().Stats().Level() == stats.LevelDisabled {
		return
	}

	info := c.GetTraceInfo().Stats()
	serverID := variable.GetString(variable.ServerID, c)

	httpStart := info.GetEvent(stats.HTTPStart)
	httpFinish := info.GetEvent(stats.HTTPFinish)
	if httpFinish == nil || httpStart == nil {
		return
	}

	s.httpServerActiveRequests.Dec()

	reqDuration := httpFinish.Time().Sub(httpStart.Time())
	_ = counterAdd(s.httpServerRequests, 1, genRequestDurationLabels(c))
	_ = histogramObserve(s.httpServerRequestDuration, reqDuration, genRequestDurationLabels(c))

	upstreamDuration := c.GetDuration(variable.UpstreamDuration)

	bifrostDuration := reqDuration - upstreamDuration
	_ = histogramObserve(s.httpBifrostRequestDuration, bifrostDuration, genRequestDurationLabels(c))

	serverLabel := make(prom.Labels)
	serverLabel[labelServerID] = serverID
	requestSize := info.RecvSize()
	responseSize := info.SendSize()

	_ = counterAdd(s.httpServerRequestBodySize, requestSize, serverLabel)
	_ = counterAdd(s.httpServerResponseBodySize, responseSize, serverLabel)

}

// NewTracer provides tracer for server access, addr and path is the scrape_configs for prometheus server.
func NewTracer(opts ...Option) tracer.Tracer {
	cfg := defaultConfig()

	for _, opts := range opts {
		opts.apply(cfg)
	}

	httpServerRequestBodySize := prom.NewCounterVec(
		prom.CounterOpts{
			Name: "http_server_request_body_size",
			Help: "Size of HTTP server request bodies.",
		},
		[]string{labelServerID},
	)
	prom.MustRegister(httpServerRequestBodySize)

	httpServerResponseBodySize := prom.NewCounterVec(
		prom.CounterOpts{
			Name: "http_server_response_body_size",
			Help: "Size of HTTP server response bodies.",
		},
		[]string{labelServerID},
	)
	prom.MustRegister(httpServerResponseBodySize)

	httpServerRequests := prom.NewCounterVec(
		prom.CounterOpts{
			Name: "http_server_requests",
			Help: "Total number of HTTPs completed by the server, regardless of success or failure",
		},
		[]string{labelMethod, labelPath, labelStatusCode, labelServerID, labelRouteID, labelServiceID},
	)
	prom.MustRegister(httpServerRequests)

	httpServerActiveRequests := prom.NewGauge(
		prom.GaugeOpts{
			Name: "http_server_active_requests",
			Help: "Number of active HTTP server requests",
		},
	)

	prom.MustRegister(httpServerActiveRequests)

	httpServerRequestDuration := prom.NewHistogramVec(
		prom.HistogramOpts{
			Name:    "http_server_request_duration",
			Help:    "Duration of HTTP server requests. (seconds)",
			Buckets: cfg.buckets,
		},
		[]string{labelMethod, labelPath, labelStatusCode, labelServerID, labelRouteID, labelServiceID},
	)
	prom.MustRegister(httpServerRequestDuration)

	httpBifrostRequestDuration := prom.NewHistogramVec(
		prom.HistogramOpts{
			Name:    "http_bifrost_request_duration",
			Help:    "Duration of HTTP requests handled by bifrost. (seconds)",
			Buckets: cfg.buckets,
		},
		[]string{labelMethod, labelPath, labelStatusCode, labelServerID, labelRouteID, labelServiceID},
	)
	prom.MustRegister(httpBifrostRequestDuration)

	// TODO: add total connections

	return &serverTracer{
		httpServerRequestBodySize:  httpServerRequestBodySize,
		httpServerResponseBodySize: httpServerResponseBodySize,
		httpServerRequests:         httpServerRequests,
		httpServerActiveRequests:   httpServerActiveRequests,
		httpServerRequestDuration:  httpServerRequestDuration,
		httpBifrostRequestDuration: httpBifrostRequestDuration,
	}
}

func defaultValIfEmpty(val, def string) string {
	if val == "" {
		return def
	}
	return val
}
