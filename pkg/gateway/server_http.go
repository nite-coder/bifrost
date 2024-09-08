package gateway

import (
	"context"
	"crypto/tls"
	"fmt"
	"http-benchmark/pkg/config"
	"io"
	"log/slog"
	"net"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/common/tracer"
	configHTTP2 "github.com/hertz-contrib/http2/config"
	"github.com/hertz-contrib/http2/factory"
	hertzslog "github.com/hertz-contrib/logger/slog"
	"github.com/hertz-contrib/pprof"
	"golang.org/x/sys/unix"
)

type HTTPServer struct {
	options  *config.ServerOptions
	switcher *switcher
	server   *server.Hertz
}

func newHTTPServer(bifrost *Bifrost, serverOpts config.ServerOptions, tracers []tracer.Tracer, disableListener bool) (*HTTPServer, error) {
	ctx := context.Background()

	hzOpts := []hzconfig.Option{
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		server.WithSenseClientDisconnection(true),
		server.WithReadTimeout(time.Second * 60),
		server.WithWriteTimeout(time.Second * 60),
		server.WithExitWaitTime(time.Second * 10),
		server.WithKeepAliveTimeout(time.Second * 60),
		server.WithKeepAlive(true),
		server.WithALPN(true),
		server.WithStreamBody(true),
		withDefaultServerHeader(true),
	}

	if !disableListener {
		listenerConfig := &net.ListenConfig{
			Control: func(network, address string, c syscall.RawConn) error {
				var opErr error
				err := c.Control(func(fd uintptr) {
					if serverOpts.ReusePort {
						if err := setTCPReusePort(fd); err != nil {
							opErr = err
							return
						}
					}

					if serverOpts.TCPQuickAck {
						if err := setTCPQuickAck(fd); err != nil {
							opErr = err
							return
						}
					}

					if serverOpts.TCPFastOpen {
						if err := setTCPFastOpen(fd); err != nil {
							opErr = err
							return
						}
					}
				})
				if err != nil {
					return err
				}
				return opErr
			},
		}

		listener, err := bifrost.zero.Listener(ctx, "tcp", serverOpts.Bind, listenerConfig)
		if err != nil {
			return nil, err
		}

		if serverOpts.Backlog > 0 && runtime.GOOS == "linux" {
			tl, ok := listener.(*net.TCPListener)
			if !ok {
				return nil, fmt.Errorf("only tcp listener supported, called with %#v", listener)
			}
			file, err := tl.File()
			if err != nil {
				return nil, err
			}
			fd := int(file.Fd())
			err = unix.Listen(fd, serverOpts.Backlog)
			if err != nil {
				return nil, err
			}
		}

		hzOpts = append(hzOpts, server.WithListener(listener))
	}

	if serverOpts.Timeout.KeepAlive.Seconds() > 0 {
		hzOpts = append(hzOpts, server.WithKeepAliveTimeout(serverOpts.Timeout.KeepAlive))
	}

	if serverOpts.Timeout.Idle.Seconds() > 0 {
		hzOpts = append(hzOpts, server.WithIdleTimeout(serverOpts.Timeout.Idle))
	}

	if serverOpts.Timeout.Read.Seconds() > 0 {
		hzOpts = append(hzOpts, server.WithReadTimeout(serverOpts.Timeout.Read))
	}

	if serverOpts.Timeout.Write.Seconds() > 0 {
		hzOpts = append(hzOpts, server.WithWriteTimeout(serverOpts.Timeout.Write))
	}

	if serverOpts.Timeout.Graceful.Seconds() > 0 {
		hzOpts = append(hzOpts, server.WithExitWaitTime(serverOpts.Timeout.Graceful))
	}

	if serverOpts.MaxRequestBodySize > 0 {
		hzOpts = append(hzOpts, server.WithMaxRequestBodySize(serverOpts.MaxRequestBodySize))
	}

	if serverOpts.ReadBufferSize > 0 {
		hzOpts = append(hzOpts, server.WithReadBufferSize(serverOpts.ReadBufferSize))
	}

	engine, err := newEngine(bifrost, serverOpts)
	if err != nil {
		return nil, err
	}

	switcher := newSwitcher(engine)

	// hertz server
	logger := hertzslog.NewLogger(hertzslog.WithOutput(io.Discard))
	hlog.SetLevel(hlog.LevelError)
	hlog.SetLogger(logger)
	hlog.SetSilentMode(true)

	hzOpts = append(hzOpts, engine.hzOptions...)

	for _, tracer := range tracers {
		hzOpts = append(hzOpts, server.WithTracer(tracer))
	}

	if serverOpts.HTTP2 && (len(serverOpts.TLS.CertPEM) == 0 || len(serverOpts.TLS.KeyPEM) == 0) {
		hzOpts = append(hzOpts, server.WithH2C(true))
	}

	var tlsConfig *tls.Config
	if len(serverOpts.TLS.CertPEM) > 0 || len(serverOpts.TLS.KeyPEM) > 0 {
		tlsConfig = &tls.Config{
			MinVersion:               tls.VersionTLS13,
			CurvePreferences:         []tls.CurveID{tls.X25519, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
		}

		if serverOpts.TLS.CertPEM == "" {
			return nil, fmt.Errorf("cert_pem can't be empty")
		}

		if serverOpts.TLS.KeyPEM == "" {
			return nil, fmt.Errorf("key_pem can't be empty")
		}

		certPEM, err := os.ReadFile(serverOpts.TLS.CertPEM)
		if err != nil {
			return nil, err
		}

		keyPEM, err := os.ReadFile(serverOpts.TLS.KeyPEM)
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
		options: &serverOpts,
	}

	h := server.Default(hzOpts...)

	if serverOpts.HTTP2 {
		http2opts := []configHTTP2.Option{}

		if serverOpts.Timeout.Idle.Seconds() > 0 {
			http2opts = append(http2opts, configHTTP2.WithIdleTimeout(serverOpts.Timeout.Idle))
		}

		if serverOpts.Timeout.Read.Seconds() > 0 {
			http2opts = append(http2opts, configHTTP2.WithReadTimeout(serverOpts.Timeout.Read))
		}

		if len(serverOpts.TLS.CertPEM) > 0 || len(serverOpts.TLS.KeyPEM) > 0 {
			h.AddProtocol("h2", factory.NewServerFactory(http2opts...))
			tlsConfig.NextProtos = append(tlsConfig.NextProtos, "h2")
		} else {
			h.AddProtocol("h2", factory.NewServerFactory(http2opts...))
		}
	}

	h.OnShutdown = append(h.OnShutdown, func(ctx context.Context) {
		// if accessLogTracer != nil {
		// 	accessLogTracer.Shutdown()
		// }

	})

	if serverOpts.PPROF {
		pprof.Register(h)
	}

	h.Use(switcher.ServeHTTP)

	httpServer.switcher = switcher
	httpServer.server = h

	return httpServer, nil
}

func (s *HTTPServer) Run() {
	slog.Info("starting server", "id", s.options.ID, "bind", s.options.Bind)
	s.server.Spin()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *HTTPServer) Bind() string {
	return s.options.Bind
}

func (s *HTTPServer) SetEngine(engine *Engine) {
	s.switcher.SetEngine(engine)
}

func (s *HTTPServer) Engine() *Engine {
	return s.switcher.Engine()
}
