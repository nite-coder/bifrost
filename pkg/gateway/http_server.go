package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	bifrostConfig "http-benchmark/pkg/config"
	"io"
	"log/slog"
	"net"
	"os"
	"syscall"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/common/tracer"
	configHTTP2 "github.com/hertz-contrib/http2/config"
	"github.com/hertz-contrib/http2/factory"
	hertzslog "github.com/hertz-contrib/logger/slog"
	"github.com/hertz-contrib/pprof"
	"golang.org/x/sys/unix"
)

type HTTPServer struct {
	entryOpts bifrostConfig.EntryOptions
	switcher  *switcher
	server    *server.Hertz
}

func newHTTPServer(bifrost *Bifrost, entryOpts bifrostConfig.EntryOptions, tracers []tracer.Tracer) (*HTTPServer, error) {

	hzOpts := []config.Option{
		server.WithHostPorts(entryOpts.Bind),
		server.WithIdleTimeout(entryOpts.IdleTimeout),
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		server.WithSenseClientDisconnection(true),
		withDefaultServerHeader(true),
		server.WithALPN(true),
	}

	engine, err := newEngine(bifrost, entryOpts)
	if err != nil {
		return nil, err
	}

	switcher := newSwitcher(engine)

	// hertz server
	logger := hertzslog.NewLogger(hertzslog.WithOutput(io.Discard))
	hlog.SetLevel(hlog.LevelError)
	hlog.SetLogger(logger)
	hlog.SetSilentMode(true)

	hzOpts = append(hzOpts, engine.options...)

	for _, tracer := range tracers {
		hzOpts = append(hzOpts, server.WithTracer(tracer))
	}

	if entryOpts.HTTP2 && !entryOpts.TLS.Enabled {
		hzOpts = append(hzOpts, server.WithH2C(true))
	}

	if entryOpts.ReusePort {
		hzOpts = append(hzOpts, server.WithListenConfig(&net.ListenConfig{
			Control: func(network, address string, c syscall.RawConn) error {
				return c.Control(func(fd uintptr) {
					err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
					if err != nil {
						return
					}
				})
			},
		}))
	}

	var tlsConfig *tls.Config
	if entryOpts.TLS.Enabled {
		tlsConfig = &tls.Config{
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
		}

		if entryOpts.TLS.CertPEM == "" {
			return nil, fmt.Errorf("cert_pem can't be empty")
		}

		if entryOpts.TLS.KeyPEM == "" {
			return nil, fmt.Errorf("key_pem can't be empty")
		}

		certPEM, err := os.ReadFile(entryOpts.TLS.CertPEM)
		if err != nil {
			return nil, err
		}

		keyPEM, err := os.ReadFile(entryOpts.TLS.KeyPEM)
		if err != nil {
			return nil, err
		}

		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
		hzOpts = append(hzOpts, server.WithTLS(tlsConfig))
	}

	httpServer := &HTTPServer{
		entryOpts: entryOpts,
	}

	h := server.Default(hzOpts...)

	if entryOpts.HTTP2 && entryOpts.TLS.Enabled {
		// register http2 server factory
		h.AddProtocol("h2", factory.NewServerFactory(
			//configHTTP2.WithReadTimeout(time.Minute),
			configHTTP2.WithDisableKeepAlive(false)))

		tlsConfig.NextProtos = append(tlsConfig.NextProtos, "h2")
	}

	if entryOpts.HTTP2 && !entryOpts.TLS.Enabled {
		h.AddProtocol("h2", factory.NewServerFactory())
	}

	h.OnShutdown = append(h.OnShutdown, func(ctx context.Context) {
		// if accessLogTracer != nil {
		// 	accessLogTracer.Shutdown()
		// }

	})

	pprof.Register(h)

	h.Use(switcher.ServeHTTP)

	httpServer.switcher = switcher
	httpServer.server = h

	return httpServer, nil
}

func (s *HTTPServer) Run() {
	slog.Info("starting entry", "id", s.entryOpts.ID, "bind", s.entryOpts.Bind)
	s.server.Spin()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
