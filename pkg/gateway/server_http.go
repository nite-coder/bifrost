package gateway

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/common/tracer"
	"github.com/cloudwego/hertz/pkg/common/tracer/stats"
	hznetpoll "github.com/cloudwego/hertz/pkg/network/netpoll"
	"github.com/cloudwego/netpoll"
	configHTTP2 "github.com/hertz-contrib/http2/config"
	"github.com/hertz-contrib/http2/factory"
	hertzslog "github.com/hertz-contrib/logger/slog"
	"github.com/hertz-contrib/pprof"
	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/zero"
	prom "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sys/unix"
)

var (
	httpServerOpenConnections *prom.GaugeVec
)

func init() {
	httpServerOpenConnections = prom.NewGaugeVec(
		prom.GaugeOpts{
			Name: "http_server_open_connections",
			Help: "Number of open connections for servers",
		},
		[]string{"server_id"},
	)

	prom.MustRegister(httpServerOpenConnections)
}

type HTTPServer struct {
	isActive         atomic.Bool
	options          *config.ServerOptions
	switcher         *switcher
	server           *server.Hertz
	totalConnections atomic.Int64
}

func newHTTPServer(bifrost *Bifrost, serverOptions config.ServerOptions, tracers []tracer.Tracer, disableListener bool) (*HTTPServer, error) {
	ctx := context.Background()

	httpServer := &HTTPServer{}
	httpServer.isActive.Store(true)

	hzOpts := []hzconfig.Option{
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		server.WithTraceLevel(stats.LevelBase),
		server.WithReadTimeout(time.Second * 60),
		server.WithWriteTimeout(time.Second * 60),
		server.WithExitWaitTime(time.Second * 10),
		server.WithKeepAliveTimeout(time.Second * 60),
		server.WithKeepAlive(true),
		server.WithALPN(true),
		server.WithStreamBody(true),
		server.WithOnAccept(func(conn net.Conn) context.Context {
			if conn != nil {
				hzConn, ok := conn.(*hznetpoll.Conn)
				if ok {
					nc, ok := hzConn.Conn.(netpoll.Conn)
					if ok {
						_ = setCloExec(nc.Fd())
					}

					if bifrost.options.Metrics.Prometheus.Enabled {
						netpollConn, ok := hzConn.Conn.(netpoll.Connection)
						if ok {
							httpServer.totalConnections.Add(1)
							_ = netpollConn.AddCloseCallback(func(connection netpoll.Connection) error {
								httpServer.totalConnections.Add(-1)
								return nil
							})
						}
					}
				}
			}

			return context.Background()
		}),
		withDefaultServerHeader(true),
	}

	if bifrost.options.Metrics.Prometheus.Enabled {
		go safety.Go(context.Background(), func() {
			ticker := time.NewTicker(time.Second * 10)

			for range ticker.C {
				if !httpServer.isActive.Load() {
					break
				}

				labels := make(prom.Labels)
				labels["server_id"] = serverOptions.ID
				totalConn := httpServer.totalConnections.Load()

				httpServerOpenConnections.With(labels).Set(float64(totalConn))
			}
		})
	}

	if !disableListener {
		listenerConfig := &net.ListenConfig{
			Control: func(network, address string, c syscall.RawConn) error {
				var opErr error
				err := c.Control(func(fd uintptr) {
					if serverOptions.ReusePort {
						if err := setTCPReusePort(fd); err != nil {
							opErr = err
							return
						}
					}

					if serverOptions.TCPQuickAck {
						if err := setTCPQuickAck(fd); err != nil {
							opErr = err
							return
						}
					}

					if serverOptions.TCPFastOpen {
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

		listenerOptions := &zero.ListenerOptions{
			Network: "tcp",
			Address: serverOptions.Bind,
			Config:  listenerConfig,
		}

		if serverOptions.ProxyProtocol {
			listenerOptions.ProxyProtocol = true
		}

		listener, err := bifrost.zero.Listener(ctx, listenerOptions)
		if err != nil {
			return nil, err
		}

		if serverOptions.Backlog > 0 && runtime.GOOS == "linux" {
			tl, ok := listener.(*net.TCPListener)
			if !ok {
				return nil, fmt.Errorf("only tcp listener supported, called with %#v", listener)
			}
			file, err := tl.File()
			if err != nil {
				return nil, err
			}
			fd := int(file.Fd())
			err = unix.Listen(fd, serverOptions.Backlog)
			if err != nil {
				return nil, err
			}
		}

		hzOpts = append(hzOpts, server.WithListener(listener))
	}

	if serverOptions.Timeout.KeepAlive > 0 {
		hzOpts = append(hzOpts, server.WithKeepAliveTimeout(serverOptions.Timeout.KeepAlive))
	}

	if serverOptions.Timeout.Idle > 0 {
		hzOpts = append(hzOpts, server.WithIdleTimeout(serverOptions.Timeout.Idle))
	}

	if serverOptions.Timeout.Read > 0 {
		hzOpts = append(hzOpts, server.WithReadTimeout(serverOptions.Timeout.Read))
	}

	if serverOptions.Timeout.Write > 0 {
		hzOpts = append(hzOpts, server.WithWriteTimeout(serverOptions.Timeout.Write))
	}

	if serverOptions.Timeout.Graceful > 0 {
		hzOpts = append(hzOpts, server.WithExitWaitTime(serverOptions.Timeout.Graceful))
	}

	if serverOptions.MaxRequestBodySize > 0 {
		hzOpts = append(hzOpts, server.WithMaxRequestBodySize(serverOptions.MaxRequestBodySize))
	}

	if serverOptions.ReadBufferSize > 0 {
		hzOpts = append(hzOpts, server.WithReadBufferSize(serverOptions.ReadBufferSize))
	}

	engine, err := newEngine(bifrost, serverOptions)
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

	if serverOptions.HTTP2 && (len(serverOptions.TLS.CertPEM) == 0 || len(serverOptions.TLS.KeyPEM) == 0) {
		hzOpts = append(hzOpts, server.WithH2C(true))
	}

	var tlsConfig *tls.Config
	if len(serverOptions.TLS.CertPEM) > 0 || len(serverOptions.TLS.KeyPEM) > 0 {
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

		if serverOptions.TLS.CertPEM == "" {
			return nil, errors.New("cert_pem can't be empty")
		}

		if serverOptions.TLS.KeyPEM == "" {
			return nil, errors.New("key_pem can't be empty")
		}

		certPEM, err := os.ReadFile(serverOptions.TLS.CertPEM)
		if err != nil {
			return nil, err
		}

		keyPEM, err := os.ReadFile(serverOptions.TLS.KeyPEM)
		if err != nil {
			return nil, err
		}

		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
		hzOpts = append(hzOpts, server.WithTLS(tlsConfig))
	} else {
		hzOpts = append(hzOpts, server.WithSenseClientDisconnection(true))
	}

	httpServer.options = &serverOptions

	h := server.New(hzOpts...)

	if len(serverOptions.RemoteIPHeaders) > 0 || len(serverOptions.TrustedCIDRS) > 0 {
		clientIPOptions := app.ClientIPOptions{
			RemoteIPHeaders: []string{"X-Forwarded-For", "X-Real-IP"},
			TrustedCIDRs:    defaultTrustedCIDRs,
		}

		if len(serverOptions.RemoteIPHeaders) > 0 {
			clientIPOptions.RemoteIPHeaders = serverOptions.RemoteIPHeaders
		}

		if len(serverOptions.TrustedCIDRS) > 0 {
			clientIPOptions.TrustedCIDRs = make([]*net.IPNet, len(serverOptions.TrustedCIDRS))
			for i, ip := range serverOptions.TrustedCIDRS {
				_, clientIPOptions.TrustedCIDRs[i], err = net.ParseCIDR(ip)
				if err != nil {
					return nil, err
				}
			}
		}

		h.SetClientIPFunc(app.ClientIPWithOption(clientIPOptions))
	}

	if serverOptions.HTTP2 {
		http2opts := []configHTTP2.Option{}

		if serverOptions.Timeout.Idle > 0 {
			http2opts = append(http2opts, configHTTP2.WithIdleTimeout(serverOptions.Timeout.Idle))
		}

		if serverOptions.Timeout.Read > 0 {
			http2opts = append(http2opts, configHTTP2.WithReadTimeout(serverOptions.Timeout.Read))
		}

		if len(serverOptions.TLS.CertPEM) > 0 || len(serverOptions.TLS.KeyPEM) > 0 {
			h.AddProtocol("h2", factory.NewServerFactory(http2opts...))
			tlsConfig.NextProtos = append(tlsConfig.NextProtos, "h2")
		} else {
			h.AddProtocol("h2", factory.NewServerFactory(http2opts...))
		}
	}

	h.OnShutdown = append(h.OnShutdown, func(ctx context.Context) {
		for _, tracer := range tracers {
			if closer, ok := tracer.(io.Closer); ok {
				_ = closer.Close()
			}
		}
	})

	if serverOptions.PPROF {
		pprof.Register(h)
	}

	h.Use(switcher.ServeHTTP)

	httpServer.switcher = switcher
	httpServer.server = h

	return httpServer, nil
}

func (s *HTTPServer) Run() {
	slog.Info("starting server", "id", s.options.ID, "bind", s.options.Bind, "transporter", s.server.GetTransporterName())
	_ = s.server.Run()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	s.isActive.Store(false)
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
