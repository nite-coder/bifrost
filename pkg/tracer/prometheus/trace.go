package prometheus

import (
	"context"
	"http-benchmark/pkg/config"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/tracer"
	"github.com/cloudwego/hertz/pkg/common/tracer/stats"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

const (
	labelEntry      = "entry"
	labelMethod     = "method"
	labelPath       = "path"
	labelStatusCode = "statusCode"

	unknownLabelValue = "unknown"
)

// genLabels make labels values.
func genLabels(ctx *app.RequestContext) prom.Labels {
	labels := make(prom.Labels)

	entryID := ctx.GetString(config.ENTRY_ID)
	labels[labelEntry] = defaultValIfEmpty(entryID, unknownLabelValue)
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
	entryID := c.GetString(config.ENTRY_ID)

	httpStart := info.GetEvent(stats.HTTPStart)
	httpFinish := info.GetEvent(stats.HTTPFinish)
	if httpFinish == nil || httpStart == nil {
		return
	}

	cost := httpFinish.Time().Sub(httpStart.Time())
	_ = counterAdd(s.requestTotalCounter, 1, genLabels(c))
	_ = histogramObserve(s.requestDurationHistogram, cost, genLabels(c))

	entryLabel := make(prom.Labels)
	entryLabel[labelEntry] = entryID
	requestSize := info.RecvSize()
	responseSize := info.SendSize()

	_ = counterAdd(s.requestSizeTotalCounter, requestSize, entryLabel)
	_ = counterAdd(s.respoonseSizeTotalCounter, responseSize, entryLabel)

}

// NewTracer provides tracer for server access, addr and path is the scrape_configs for prometheus server.
func NewTracer(addr, path string, opts ...Option) tracer.Tracer {
	cfg := defaultConfig()

	for _, opts := range opts {
		opts.apply(cfg)
	}

	if !cfg.disableServer {
		http.Handle(path, promhttp.HandlerFor(cfg.registry, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError}))
		go func() {
			slog.Info("starting prometheus server", "addr", addr)
			if err := http.ListenAndServe(addr, nil); err != nil {
				hlog.Fatal("bifrost: Unable to start a promhttp server, err: " + err.Error())
			}
		}()
	}

	requestSizeTotalCounter := prom.NewCounterVec(
		prom.CounterOpts{
			Name: "bifrost_request_size_total",
			Help: "the server received request body size, unit byte.",
		},
		[]string{labelEntry},
	)
	cfg.registry.MustRegister(requestSizeTotalCounter)

	responseSizeTotalCounter := prom.NewCounterVec(
		prom.CounterOpts{
			Name: "bifrost_response_size_total",
			Help: "the server send response body size, unit byte.",
		},
		[]string{labelEntry},
	)
	cfg.registry.MustRegister(responseSizeTotalCounter)

	requestTotalCounter := prom.NewCounterVec(
		prom.CounterOpts{
			Name: "bifrost_request_total",
			Help: "Total number of HTTPs completed by the server, regardless of success or failure.",
		},
		[]string{labelEntry, labelMethod, labelStatusCode, labelPath},
	)
	cfg.registry.MustRegister(requestTotalCounter)

	requestDurationHistogram := prom.NewHistogramVec(
		prom.HistogramOpts{
			Name:    "bifrost_request_duration",
			Help:    "Latency (seconds) of HTTP that had been application-level handled by the server.",
			Buckets: cfg.buckets,
		},
		[]string{labelEntry, labelMethod, labelStatusCode, labelPath},
	)
	cfg.registry.MustRegister(requestDurationHistogram)

	if cfg.enableGoCollector {
		cfg.registry.MustRegister(collectors.NewGoCollector(collectors.WithGoCollectorRuntimeMetrics(cfg.runtimeMetricRules...)))
	}

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
