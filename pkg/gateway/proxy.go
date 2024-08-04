package gateway

import (
	"bytes"
	"context"
	"fmt"
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/log"
	"log/slog"
	"net"
	"net/textproto"
	"net/url"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"

	http2Config "github.com/hertz-contrib/http2/config"
	"github.com/hertz-contrib/http2/factory"
	hertztracing "github.com/hertz-contrib/obs-opentelemetry/tracing"
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

type Proxy struct {
	options *ProxyOptions

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

	weight int
}

type ProxyOptions struct {
	Target   string
	Protocol config.Protocol
	Weight   int
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

type clientOptions struct {
	isTracingEnabled bool
	http2            bool
	hzOptions        []hzconfig.ClientOption
}

func newClient(opts clientOptions) (*client.Client, error) {
	c, err := client.NewClient(opts.hzOptions...)
	if err != nil {
		return nil, err
	}

	if opts.http2 {
		c.SetClientFactory(factory.NewClientFactory(http2Config.WithAllowHTTP(true)))
	}

	if opts.isTracingEnabled {
		c.Use(hertztracing.ClientMiddleware())
	}

	return c, nil
}

func NewReverseProxy(opts ProxyOptions, client *client.Client) (*Proxy, error) {
	addr, err := url.Parse(opts.Target)
	if err != nil {
		return nil, err
	}

	if client == nil {
		clientOptions := clientOptions{
			isTracingEnabled: false,
			http2:            false,
			hzOptions:        newDefaultClientOptions(),
		}
		client, err = newClient(clientOptions)
		if err != nil {
			return nil, err
		}
	}

	r := &Proxy{
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
			}

			switch addr.Scheme {
			case "http":
				req.SetIsTLS(false)
			case "https":
				req.SetIsTLS(true)
			}

			req.SetRequestURI(cast.B2S(JoinURLPath(req, opts.Target)))
			//req.Header.SetHostBytes(req.URI().Host())
		},
		client: client,
	}

	return r, nil
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

func (r *Proxy) defaultErrorHandler(c *app.RequestContext, _ error) {
	c.Response.Header.SetStatusCode(consts.StatusBadGateway)
}

func (p *Proxy) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	outReq := &ctx.Request
	outResp := &ctx.Response

	var err error
	ctx.Set(config.UPSTREAM_ADDR, p.targetHost)

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
		p.getErrorHandler()(ctx, fmt.Errorf("backend tried to switch to invalid protocol %q", reqUpType))
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

			buf.Write(tmp)
			buf.WriteString(", ")
			buf.WriteString(ip)
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
			buf := bytebufferpool.Get()
			defer bytebufferpool.Put(buf)

			buf.Write(outReq.Method())
			buf.Write(spaceByte)
			buf.Write(outReq.URI().FullURI())
			uri := buf.String()

			logger := log.FromContext(c)
			logger.ErrorContext(c, "sent upstream error",
				slog.String("error", err.Error()),
				slog.String("upstream", uri),
			)

			if err.Error() == "timeout" {
				ctx.Set("target_timeout", true)
			}

			p.getErrorHandler()(ctx, err)
			return
		}
		return
	}

	err = p.client.Do(c, outReq, outResp)
	if err != nil {
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)

		buf.Write(outReq.Method())
		buf.Write(spaceByte)
		buf.Write(outReq.URI().FullURI())
		uri := buf.String()

		logger := log.FromContext(c)
		logger.ErrorContext(c, "sent upstream error",
			slog.String("error", err.Error()),
			slog.String("upstream", uri),
		)

		if err.Error() == "timeout" {
			ctx.Set("target_timeout", true)
		}

		p.getErrorHandler()(ctx, err)
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

// SetDirector use to customize protocol.Request
func (r *Proxy) SetDirector(director func(req *protocol.Request)) {
	r.director = director
}

// SetClient use to customize client
func (r *Proxy) SetClient(client *client.Client) {
	r.client = client
}

// SetErrorHandler use to customize error handler
func (r *Proxy) SetErrorHandler(eh func(c *app.RequestContext, err error)) {
	r.errorHandler = eh
}

func (r *Proxy) SetTransferTrailer(b bool) {
	r.transferTrailer = b
}

func (r *Proxy) getErrorHandler() func(c *app.RequestContext, err error) {
	if r.errorHandler != nil {
		return r.errorHandler
	}
	return r.defaultErrorHandler
}
