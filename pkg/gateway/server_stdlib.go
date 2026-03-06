package gateway

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/cloudwego/hertz/pkg/common/tracer"
	"github.com/cloudwego/hertz/pkg/common/tracer/stats"
	"github.com/cloudwego/hertz/pkg/common/tracer/traceinfo"

	"github.com/nite-coder/bifrost/pkg/config"
)

// tracerController is a lightweight controller that drives Hertz tracers for
// requests handled outside the normal Hertz HTTP/1 pipeline (i.e. HTTP/2 via
// Go stdlib).  It mirrors the logic of Hertz's internal stats.Controller so
// that accesslog and metrics tracers work identically for gRPC/h2 traffic.
type tracerController struct {
	tracers []tracer.Tracer
}

func newTracerController(tracers []tracer.Tracer) *tracerController {
	return &tracerController{tracers: tracers}
}

func (tc *tracerController) hasTracer() bool {
	return len(tc.tracers) > 0
}

// doStart records the HTTPStart event on the RequestContext and calls
// tracer.Start on every registered tracer.
func (tc *tracerController) doStart(ctx context.Context, c *app.RequestContext) (startCtx context.Context) {
	startCtx = ctx
	defer tc.tryRecover()
	if ti := c.GetTraceInfo(); ti != nil {
		ti.Stats().Record(stats.HTTPStart, stats.StatusInfo, "")
	}
	for _, t := range tc.tracers {
		//nolint:fatcontext // intentional: mirrors Hertz DoStart; each tracer enriches the context
		startCtx = t.Start(startCtx, c)
	}
	return startCtx
}

// doFinish records the HTTPFinish event and calls tracer.Finish in reverse
// order, exactly matching Hertz's internal behaviour.
func (tc *tracerController) doFinish(ctx context.Context, c *app.RequestContext, err error) {
	defer tc.tryRecover()
	if ti := c.GetTraceInfo(); ti != nil {
		st := stats.StatusInfo
		if err != nil {
			st = stats.StatusError
			ti.Stats().SetError(err)
		}
		ti.Stats().Record(stats.HTTPFinish, st, "")
	}
	for i := len(tc.tracers) - 1; i >= 0; i-- {
		tc.tracers[i].Finish(ctx, c)
	}
}

func (tc *tracerController) tryRecover() {
	if r := recover(); r != nil {
		slog.Warn("panic in tracer call (HTTP2 bridge); metrics/logs may be incomplete",
			"panic", r,
			"stack", string(debug.Stack()),
		)
	}
}

// HertzBridge implements http.Handler by bridging to a Hertz instance.
// When tracers are registered it also drives the Hertz tracer pipeline so that
// accesslog and Prometheus metrics work correctly for HTTP/2 (gRPC) requests.
type HertzBridge struct {
	Hertz      *server.Hertz
	tracerCtl  *tracerController
	traceLevel stats.Level

	// ctxPool is per-bridge so each pool entry can carry a correctly
	// configured TraceInfo when tracers are active.
	ctxPool sync.Pool
}

func newHertzBridge(h *server.Hertz, tracers []tracer.Tracer) *HertzBridge {
	ctl := newTracerController(tracers)
	traceLevel := stats.LevelBase

	b := &HertzBridge{
		Hertz:      h,
		tracerCtl:  ctl,
		traceLevel: traceLevel,
	}

	b.ctxPool.New = func() any {
		c := app.NewContext(0)
		if ctl.hasTracer() {
			c.SetEnableTrace(true)
			ti := traceinfo.NewTraceInfo()
			ti.Stats().SetLevel(traceLevel)
			c.SetTraceInfo(ti)
		}
		return c
	}

	return b
}

func (b *HertzBridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := b.ctxPool.Get().(*app.RequestContext)
	defer func() {
		c.Reset()
		b.ctxPool.Put(c)
	}()

	// Convert http.Request to Hertz RequestContext
	err := adaptor.CopyToHertzRequest(r, &c.Request)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("failed to copy request"))
		return
	}

	// Start Hertz tracer pipeline (records HTTPStart, calls tracer.Start).
	ctx := r.Context()
	if b.tracerCtl.hasTracer() {
		ctx = b.tracerCtl.doStart(ctx, c)
		defer func() {
			// Set approx recv/send sizes for metrics before finishing.
			if ti := c.GetTraceInfo(); ti != nil {
				ti.Stats().SetRecvSize(c.Request.Header.Len() + len(c.Request.Body()))
				ti.Stats().SetSendSize(c.Response.Header.Len() + len(c.Response.Body()))
			}
			b.tracerCtl.doFinish(ctx, c, nil)
		}()
	}

	// Process request through Hertz engine
	b.Hertz.ServeHTTP(ctx, c)

	// Copy response headers
	h := w.Header()
	trailers := c.Response.Header.Trailer()

	// Pre-collect all potential trailers to announce them in the "Trailer" header
	trailerNames := make(map[string]bool)
	trailers.VisitAll(func(k, v []byte) {
		trailerNames[string(k)] = true
	})

	c.Response.Header.VisitAll(func(k, v []byte) {
		key := string(k)
		canonicalKey := http.CanonicalHeaderKey(key)

		// Forbidden headers in HTTP/2 and headers that should be treated as trailers
		switch canonicalKey {
		case "Connection",
			"Keep-Alive",
			"Proxy-Connection",
			"Transfer-Encoding",
			"Upgrade",
			"Trailer",
			"Content-Length":
			return
		}
		h.Add(key, string(v))
	})

	// Announce trailers
	for name := range trailerNames {
		h.Add("Trailer", name)
	}

	// For non-gRPC responses, set Content-Length if body is present in buffer
	isGRPC := strings.HasPrefix(h.Get("Content-Type"), "application/grpc")
	body := c.Response.Body()
	if !isGRPC && len(body) > 0 {
		h.Set("Content-Length", strconv.Itoa(len(body)))
	}

	// Write status code
	code := c.Response.StatusCode()
	if code == 0 {
		code = 200
	}
	w.WriteHeader(code)

	// Write response body (handle both buffer and stream)
	if len(body) > 0 {
		_, _ = w.Write(body)
	} else if c.Response.BodyStream() != nil {
		_, _ = io.Copy(w, c.Response.BodyStream())
	}

	// Copy Trailer values
	trailers.VisitAll(func(k, v []byte) {
		h.Add(string(k), string(v))
	})

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// NewStdlibServer creates a new http.Server that wraps a Hertz instance for HTTP/2 support.
// Pass the same tracers that are registered with the Hertz server so that accesslog and
// metrics are triggered correctly for HTTP/2 (gRPC) requests.
func NewStdlibServer(
	h *server.Hertz,
	options *config.ServerOptions,
	tlsConfig *tls.Config,
	tracers []tracer.Tracer,
) *http.Server {
	protocols := &http.Protocols{}
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	protocols.SetUnencryptedHTTP2(true) // Support h2c

	srv := &http.Server{
		Handler:      newHertzBridge(h, tracers),
		ReadTimeout:  options.Timeout.Read,
		WriteTimeout: options.Timeout.Write,
		IdleTimeout:  options.Timeout.Idle,
		Protocols:    protocols,
		TLSConfig:    tlsConfig,
	}

	return srv
}
