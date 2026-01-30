package gateway

import (
	"crypto/tls"
	"io"
	"net/http"
	"strconv"
	"strings"

	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/nite-coder/bifrost/pkg/config"
)

var (
	requestContextPool = sync.Pool{
		New: func() any {
			return app.NewContext(0)
		},
	}
)

// HertzBridge implements http.Handler by bridging to a Hertz instance
type HertzBridge struct {
	Hertz *server.Hertz
}

func (b *HertzBridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := requestContextPool.Get().(*app.RequestContext)
	defer func() {
		c.Reset()
		requestContextPool.Put(c)
	}()

	// Convert http.Request to Hertz RequestContext
	if err := adaptor.CopyToHertzRequest(r, &c.Request); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("failed to copy request"))
		return
	}

	// Process request through Hertz engine
	b.Hertz.ServeHTTP(r.Context(), c)

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
		case "Connection", "Keep-Alive", "Proxy-Connection", "Transfer-Encoding", "Upgrade", "Trailer", "Content-Length":
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

// NewStdlibServer creates a new http.Server that wraps a Hertz instance for HTTP/2 support
func NewStdlibServer(h *server.Hertz, options *config.ServerOptions, tlsConfig *tls.Config) *http.Server {
	protocols := &http.Protocols{}
	protocols.SetHTTP1(true)
	protocols.SetHTTP2(true)
	protocols.SetUnencryptedHTTP2(true) // Support h2c

	srv := &http.Server{
		Handler: &HertzBridge{
			Hertz: h,
		},
		ReadTimeout:  options.Timeout.Read,
		WriteTimeout: options.Timeout.Write,
		IdleTimeout:  options.Timeout.Idle,
		Protocols:    protocols,
		TLSConfig:    tlsConfig,
	}

	return srv
}
