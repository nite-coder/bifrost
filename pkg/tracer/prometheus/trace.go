package prometheus

import (
	"context"
	"http-benchmark/pkg/config"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/tracer"
	"github.com/cloudwego/hertz/pkg/common/tracer/stats"
	prom "github.com/prometheus/client_golang/prometheus"
)

const (
	labelServer     = "server"
	labelMethod     = "method"
	labelPath       = "path"
	labelStatusCode = "statusCode"

	unknownLabelValue = "unknown"
)

// genLabels make labels values.
func genLabels(ctx *app.RequestContext) prom.Labels {
	labels := make(prom.Labels)

	serverID := ctx.GetString(config.SERVER_ID)
	labels[labelServer] = defaultValIfEmpty(serverID, unknownLabelValue)
	labels[labelMethod] = defaultValIfEmpty(string(ctx.Request.Method()), unknownLabelValue)
	labels[labelStatusCode] = defaultValIfEmpty(strconv.Itoa(ctx.Response.Header.StatusCode()), unknownLabelValue)
	labels[labelPath] = defaultValIfEmpty(string(ctx.Request.Path()), unknownLabelValue)

	return labels
}

type serverTracer struct {
	requestSizeTotalCounter   *prom.CounterVec
	respoonseSizeTotalCounter *prom.CounterVec
	requestTotalCounter       *prom.CounterVec
	requestDurationHistogram  *prom.HistogramVec
}

// Start record the beginning of server handling request from client.
func (s *serverTracer) Start(ctx context.Context, c *app.RequestContext) context.Context {
	return ctx
}

// Finish record the ending of server handling request from client.
func (s *serverTracer) Finish(ctx context.Context, c *app.RequestContext) {
	if c.GetTraceInfo().Stats().Level() == stats.LevelDisabled {
		return
	}

	info := c.GetTraceInfo().Stats()
	serverID := c.GetString(config.SERVER_ID)

	httpStart := info.GetEvent(stats.HTTPStart)
	httpFinish := info.GetEvent(stats.HTTPFinish)
	if httpFinish == nil || httpStart == nil {
		return
	}

	cost := httpFinish.Time().Sub(httpStart.Time())
	_ = counterAdd(s.requestTotalCounter, 1, genLabels(c))
	_ = histogramObserve(s.requestDurationHistogram, cost, genLabels(c))

	serverLabel := make(prom.Labels)
	serverLabel[labelServer] = serverID
	requestSize := info.RecvSize()
	responseSize := info.SendSize()

	_ = counterAdd(s.requestSizeTotalCounter, requestSize, serverLabel)
	_ = counterAdd(s.respoonseSizeTotalCounter, responseSize, serverLabel)

}

// NewTracer provides tracer for server access, addr and path is the scrape_configs for prometheus server.
func NewTracer(opts ...Option) tracer.Tracer {
	cfg := defaultConfig()

	for _, opts := range opts {
		opts.apply(cfg)
	}

	requestSizeTotalCounter := prom.NewCounterVec(
		prom.CounterOpts{
			Name: "bifrost_request_size_total",
			Help: "the server received request body size, unit byte.",
		},
		[]string{labelServer},
	)
	prom.MustRegister(requestSizeTotalCounter)

	responseSizeTotalCounter := prom.NewCounterVec(
		prom.CounterOpts{
			Name: "bifrost_response_size_total",
			Help: "the server send response body size, unit byte.",
		},
		[]string{labelServer},
	)
	prom.MustRegister(responseSizeTotalCounter)

	requestTotalCounter := prom.NewCounterVec(
		prom.CounterOpts{
			Name: "bifrost_request_total",
			Help: "Total number of HTTPs completed by the server, regardless of success or failure.",
		},
		[]string{labelServer, labelMethod, labelStatusCode, labelPath},
	)
	prom.MustRegister(requestTotalCounter)

	requestDurationHistogram := prom.NewHistogramVec(
		prom.HistogramOpts{
			Name:    "bifrost_request_duration",
			Help:    "Latency (seconds) of HTTP that had been application-level handled by the server.",
			Buckets: cfg.buckets,
		},
		[]string{labelServer, labelMethod, labelStatusCode, labelPath},
	)
	prom.MustRegister(requestDurationHistogram)

	return &serverTracer{
		requestSizeTotalCounter:   requestSizeTotalCounter,
		respoonseSizeTotalCounter: responseSizeTotalCounter,
		requestTotalCounter:       requestTotalCounter,
		requestDurationHistogram:  requestDurationHistogram,
	}
}

func defaultValIfEmpty(val, def string) string {
	if val == "" {
		return def
	}
	return val
}
