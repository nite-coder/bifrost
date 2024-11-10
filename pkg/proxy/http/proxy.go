package http

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/nite-coder/bifrost/internal/pkg/runtime"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/proxy"
	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/bifrost/pkg/variable"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	hzerrors "github.com/cloudwego/hertz/pkg/common/errors"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/google/uuid"
	"github.com/nite-coder/blackbear/pkg/cast"
	"github.com/valyala/bytebufferpool"
)

// TrailerPrefix is a magic prefix for [ResponseWriter.Header] map keys
// that, if present, signals that the map entry is actually for
// the response trailers, and not the response headers. The prefix
// is stripped after the ServeHTTP call finishes and the values are
// sent in the trailers.
//
// This mechanism is intended only for trailers that are not known
// prior to the headers being written. If the set of trailers is fixed
// or known before the header is written, the normal Go trailers mechanism
// is preferred:
//
//	https://pkg.go.dev/net/http#ResponseWriter
//	https://pkg.go.dev/net/http#example-ResponseWriter-Trailers
const TrailerPrefix = "Trailer:"

type HTTPProxy struct {
	mu      sync.RWMutex
	id      string
	options *Options

	client *client.Client

	// target is set as a reverse proxy address
	target string

	// transferTrailer is whether to forward Trailer-related header
	transferTrailer bool

	// director must be a function which modifies the request
	// into a new request. Its response is then redirected
	// back to the original client unmodified.
	// director must not access the provided Request
	// after returning.
	director func(*protocol.Request)

	// errorHandler is an optional function that handles errors
	// reaching the backend or errors from modifyResponse.
	//
	// If nil, the default is to log the provided error and return
	// a 502 Status Bad Gateway response.
	errorHandler func(*app.RequestContext, error)

	targetHost string

	weight       uint32
	failedCount  uint
	failExpireAt time.Time
}

type Options struct {
	Target      string
	Protocol    config.Protocol
	Weight      uint32
	MaxFails    uint
	FailTimeout time.Duration
	HeaderHost  string
}

// Hop-by-hop headers. These are removed when sent to the backend.
// As of RFC 7230, hop-by-hop headers are required to appear in the
// Connection header field. These are the headers defined by the
// obsoleted RFC 2616 (section 13.5.1) and are used for backward
// compatibility.
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection", // non-standard but still sent by libcurl and rejected by e.g. google
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",      // canonicalized version of "TE"
	"Trailer", // not Trailers per URL above; https://www.rfc-editor.org/errata_search.php?eid=4522
	"Transfer-Encoding",
	"Upgrade",
}

// New creates a new reverse proxy instance.
//
// It takes a set of options and a client as parameters, and returns a new reverse proxy instance and an error.
// The options parameter specifies the target URL and other configuration options for the reverse proxy.
// The client parameter specifies the client to use for making requests to the target URL.
// The returned error is nil if the reverse proxy instance is created successfully, or an error if there is a problem.
func New(opts Options, client *client.Client) (proxy.Proxy, error) {
	addr, err := url.Parse(opts.Target)
	if err != nil {
		return nil, fmt.Errorf("proxy: http proxy fail to parse target url; %w", err)
	}

	if client == nil {
		clientOptions := ClientOptions{
			IsTracingEnabled: false,
			IsHTTP2:          false,
			HZOptions:        DefaultClientOptions(),
		}
		client, err = NewClient(clientOptions)
		if err != nil {
			return nil, err
		}
	}

	if opts.Weight == 0 {
		opts.Weight = 1
	}

	r := &HTTPProxy{
		id:              uuid.New().String(),
		transferTrailer: true,
		options:         &opts,
		target:          opts.Target,
		targetHost:      addr.Host,
		weight:          opts.Weight,
		director: func(req *protocol.Request) {

			switch opts.Protocol {
			case config.ProtocolHTTP2:
				req.Header.SetProtocol("HTTP/2.0")
			case config.ProtocolHTTP:
				req.Header.SetProtocol("HTTP/1.1")
			default:
			}

			switch addr.Scheme {
			case "http":
				req.SetIsTLS(false)
			case "https":
				req.SetIsTLS(true)
			default:
			}

			if opts.HeaderHost != "" {
				req.Header.Set("Host", opts.HeaderHost)
			}
			req.SetRequestURI(cast.B2S(JoinURLPath(req, opts.Target)))
			// req.Header.SetHostBytes(req.URI().Host())
		},
		client: client,
	}

	return r, nil
}

func (p *HTTPProxy) IsAvailable() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.options.MaxFails == 0 {
		return true
	}

	now := timecache.Now()
	if now.After(p.failExpireAt) {
		return true
	}

	if p.failedCount < p.options.MaxFails {
		return true
	}

	return false
}

func (p *HTTPProxy) AddFailedCount(count uint) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := timecache.Now()
	if now.After(p.failExpireAt) {
		p.failExpireAt = now.Add(p.options.FailTimeout)
		p.failedCount = count
	} else {
		p.failedCount += count
	}

	if p.options.MaxFails > 0 && p.failedCount >= p.options.MaxFails {
		return proxy.ErrMaxFailedCount
	}

	return nil
}

func (p *HTTPProxy) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	logger := log.FromContext(c)

	defer func() {
		if r := recover(); r != nil {
			stackTrace := runtime.StackTrace()
			logger.ErrorContext(c, "proxy: http proxy panic recovered", slog.Any("error", r), slog.String("stack", stackTrace))
			ctx.Abort()
		}

		// check upstream health
		if ctx.Response.StatusCode() >= 500 {
			err := p.AddFailedCount(1)
			if err != nil {
				slog.Warn("upstream server temporarily disabled", "id", p.id)
			}
		}
	}()

	outReq := &ctx.Request
	outResp := &ctx.Response

	var err error
	ctx.Set(variable.UPSTREAM_ADDR, p.targetHost)

	if p.director != nil {
		p.director(&ctx.Request)
	}

	outReq.Header.ResetConnectionClose()

	hasTeTrailer := false
	if p.transferTrailer {
		hasTeTrailer = checkTeHeader(&outReq.Header)
	}

	reqUpType := upgradeReqType(&outReq.Header)
	if !IsASCIIPrint(reqUpType) { // We know reqUpType is ASCII, it's checked by the caller.
		p.handleError(c, ctx, fmt.Errorf("backend tried to switch to invalid protocol %q", reqUpType))
		return
	}

	removeRequestConnHeaders(ctx)
	// Remove hop-by-hop headers to the backend. Especially
	// important is "Connection" because we want a persistent
	// connection, regardless of what the client sent to us.
	for _, h := range hopHeaders {
		if p.transferTrailer && h == "Trailer" {
			continue
		}
		outReq.Header.DelBytes(cast.S2B(h))
	}

	// Check if 'trailers' exists in te header, If exists, add an additional Te header
	if p.transferTrailer && hasTeTrailer {
		outReq.Header.Set("Te", "trailers")
	}

	// prepare request(replace headers and some URL host)
	if ip, _, err := net.SplitHostPort(ctx.RemoteAddr().String()); err == nil {
		tmp := outReq.Header.Peek("X-Forwarded-For")

		if len(tmp) > 0 {
			buf := bytebufferpool.Get()
			defer bytebufferpool.Put(buf)

			_, _ = buf.Write(tmp)
			_, _ = buf.WriteString(", ")
			_, _ = buf.WriteString(ip)
			ip = buf.String()
		}
		if tmp == nil || string(tmp) != "" {
			outReq.Header.Set("X-Forwarded-For", ip)
		}
	}

	// After stripping all the hop-by-hop connection headers above, add back any
	// necessary for protocol upgrades, such as for websockets.
	if reqUpType != "" {
		outCtx := ctx.Copy()

		outReq = &outCtx.Request
		outResp = &outCtx.Response

		outReq.Header.Set("Connection", "Upgrade")
		outReq.Header.Set("Upgrade", reqUpType)

		err = p.roundTrip(c, ctx, outReq, outResp)
		if err != nil {
			p.handleError(c, ctx, err)
			return
		}
		return
	}

	// tracer := otel.Tracer("bifrost")
	// var span trace.Span
	// if tracer != nil {
	// 	c, span = tracer.Start(c, "upstream")
	// 	defer span.End()
	// }

ProxyPassLoop:

	err = p.client.Do(c, outReq, outResp)
	if err != nil {
		if errors.Is(err, hzerrors.ErrBadPoolConn) {
			goto ProxyPassLoop
		}

		p.handleError(c, ctx, err)
		return
	}

	announcedTrailers := outResp.Header.Peek("Trailer")

	removeResponseConnHeaders(ctx)

	for _, h := range hopHeaders {
		if p.transferTrailer && h == "Trailer" {
			continue
		}
		outResp.Header.DelBytes(cast.S2B(h))
	}

	if len(announcedTrailers) > 0 {
		outResp.Header.Add("Trailer", string(announcedTrailers))
	}

}

// ID return proxy's ID
func (p *HTTPProxy) ID() string {
	return p.id
}

// SetDirector use to customize protocol.Request
func (p *HTTPProxy) SetDirector(director func(req *protocol.Request)) {
	p.director = director
}

// SetClient use to customize client
func (p *HTTPProxy) SetClient(client *client.Client) {
	p.client = client
}

// SetErrorHandler use to customize error handler
func (p *HTTPProxy) SetErrorHandler(eh func(c *app.RequestContext, err error)) {
	p.errorHandler = eh
}

func (r *HTTPProxy) SetTransferTrailer(b bool) {
	r.transferTrailer = b
}

func (p *HTTPProxy) Weight() uint32 {
	return p.weight
}

func (p *HTTPProxy) Target() string {
	return p.target
}

func (p *HTTPProxy) Close() error {
	if p.client != nil {
		p.client.CloseIdleConnections()
		p.client = nil
	}

	return nil
}

func (r *HTTPProxy) handleError(ctx context.Context, c *app.RequestContext, err error) {
	if err == nil {
		return
	}

	logger := log.FromContext(ctx)

	fullURI := fullURI(&c.Request)

	val, _ := variable.Get(variable.REQUEST_PATH, c)
	originalPath, _ := cast.ToString(val)

	logger.ErrorContext(ctx, "fail to send request to upstream",
		slog.String("error", err.Error()),
		slog.String("original_path", originalPath),
		slog.String("upstream", fullURI),
		slog.Int("upstream_status", c.Response.StatusCode()),
	)

	if errors.Is(err, hzerrors.ErrTimeout) {
		c.Set(variable.TARGET_TIMEOUT, true)
	}

	c.Response.Header.SetStatusCode(http.StatusBadGateway)
}

// removeRequestConnHeaders removes hop-by-hop headers listed in the "Connection" header of h.
// See RFC 7230, section 6.1
func removeRequestConnHeaders(c *app.RequestContext) {
	c.Request.Header.VisitAll(func(k, v []byte) {
		if cast.B2S(k) == "Connection" {
			for _, sf := range strings.Split(cast.B2S(v), ",") {
				if sf = textproto.TrimString(sf); sf != "" {
					c.Request.Header.DelBytes(cast.S2B(sf))
				}
			}
		}
	})
}

// removeRespConnHeaders removes hop-by-hop headers listed in the "Connection" header of h.
// See RFC 7230, section 6.1
func removeResponseConnHeaders(c *app.RequestContext) {
	c.Response.Header.VisitAll(func(k, v []byte) {
		if cast.B2S(k) == "Connection" {
			for _, sf := range strings.Split(cast.B2S(v), ",") {
				if sf = textproto.TrimString(sf); sf != "" {
					c.Response.Header.DelBytes(cast.S2B(sf))
				}
			}
		}
	})
}

// checkTeHeader check RequestHeader if has 'Te: trailers'
// See https://github.com/golang/go/issues/21096
func checkTeHeader(header *protocol.RequestHeader) bool {
	teHeaders := header.PeekAll("Te")
	for _, te := range teHeaders {
		if bytes.Contains(te, []byte("trailers")) {
			return true
		}
	}
	return false
}
