package gateway

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
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
	hertzslog "github.com/hertz-contrib/logger/slog"
	"github.com/hertz-contrib/pprof"
	proxyproto "github.com/pires/go-proxyproto"
	prom "github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sys/unix"

	"github.com/nite-coder/bifrost/internal/pkg/infra"
	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/config"
)

var (
	httpServerOpenConnections *prom.GaugeVec
	initLoggerOnce            sync.Once
)

const (
	defaultHTTPReadTimeout       = 60 * time.Second
	defaultHTTPWriteTimeout      = 60 * time.Second
	defaultHTTPKeepAliveTimeout  = 60 * time.Second
	defaultHTTPExitWaitTime      = 10 * time.Second
	defaultMetricsTickerInterval = 10 * time.Second
)

// ListenerMode defines whether the server should listen on a network port.
type ListenerMode int

const (
	// ListenerEnabled indicates the server should listen on the configured address.
	ListenerEnabled ListenerMode = iota
	// ListenerDisabled indicates the server should not open a listener.
	ListenerDisabled
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

// HTTPServer represents an HTTP server instance in the gateway.
type HTTPServer struct {
	options          *config.ServerOptions
	switcher         *switcher
	server           *server.Hertz
	stdlibServer     *http.Server
	listener         net.Listener
	totalConnections atomic.Int64
	isActive         atomic.Bool
}

func newHTTPServer(
	bifrost *Bifrost,
	serverOptions config.ServerOptions,
	tracers []tracer.Tracer,
	listenerMode ListenerMode,
) (*HTTPServer, error) {
	ctx := context.Background()
	var err error
	httpServer := &HTTPServer{}
	httpServer.isActive.Store(true)
	hzOpts := []hzconfig.Option{
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		server.WithTraceLevel(stats.LevelBase),
		server.WithReadTimeout(defaultHTTPReadTimeout),
		server.WithWriteTimeout(defaultHTTPWriteTimeout),
		server.WithExitWaitTime(defaultHTTPExitWaitTime),
		server.WithKeepAliveTimeout(defaultHTTPKeepAliveTimeout),
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
							_ = netpollConn.AddCloseCallback(func(_ netpoll.Connection) error {
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
			ticker := time.NewTicker(defaultMetricsTickerInterval)
			defer ticker.Stop()
			for range ticker.C {
				if !httpServer.isActive.Load() {
					return
				}
				labels := make(prom.Labels)
				labels["server_id"] = serverOptions.ID
				totalConn := httpServer.totalConnections.Load()
				httpServerOpenConnections.With(labels).Set(float64(totalConn))
			}
		})
	}
	if listenerMode == ListenerEnabled {
		listenerConfig := &net.ListenConfig{
			Control: func(_, _ string, c syscall.RawConn) error {
				var opErr error
				err = c.Control(func(fd uintptr) {
					if serverOptions.ReusePort {
						err = setTCPReusePort(fd)
						if err != nil {
							opErr = err
							return
						}
					}
					if serverOptions.TCPQuickAck {
						err = setTCPQuickAck(fd)
						if err != nil {
							opErr = err
							return
						}
					}
					if serverOptions.TCPFastOpen {
						err = setTCPFastOpen(fd)
						if err != nil {
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
		listenerOptions := &infra.ListenerOptions{
			Network: "tcp",
			Address: serverOptions.Bind,
			Config:  listenerConfig,
		}
		if serverOptions.ProxyProtocol {
			listenerOptions.ProxyProtocol = true
		}
		var listener net.Listener
		listener, err = bifrost.zeroDownTime.Listener(ctx, listenerOptions)
		if err != nil {
			return nil, err
		}
		if serverOptions.Backlog > 0 && runtime.GOOS == "linux" {
			var tl *net.TCPListener
			proxylistener, ok := listener.(*proxyproto.Listener)
			if ok {
				tl, ok = proxylistener.Listener.(*net.TCPListener)
				if !ok {
					return nil, fmt.Errorf("only tcp listener supported, called with %#v", listener)
				}
			} else {
				tl, ok = listener.(*net.TCPListener)
				if !ok {
					return nil, fmt.Errorf("only tcp listener supported, called with %#v", listener)
				}
			}
			var file *os.File
			file, err = tl.File()
			if err != nil {
				return nil, err
			}
			fd := int(file.Fd())
			err = unix.Listen(fd, serverOptions.Backlog)
			if err != nil {
				return nil, err
			}
		}
		if !serverOptions.HTTP2 {
			hzOpts = append(hzOpts, server.WithListener(listener))
		}
		httpServer.listener = listener
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
	var engine *Engine
	engine, err = newEngine(bifrost, serverOptions)
	if err != nil {
		return nil, err
	}
	switcher := newSwitcher(engine)
	// hertz server
	initLoggerOnce.Do(func() {
		logger := hertzslog.NewLogger(hertzslog.WithOutput(io.Discard))
		hlog.SetLevel(hlog.LevelError)
		hlog.SetLogger(logger)
		hlog.SetSilentMode(true)
	})
	hzOpts = append(hzOpts, engine.hzOptions...)
	for _, tr := range tracers {
		hzOpts = append(hzOpts, server.WithTracer(tr))
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
			return nil, errors.New("cert_PEM cannot be empty")
		}
		if serverOptions.TLS.KeyPEM == "" {
			return nil, errors.New("key_PEM cannot be empty")
		}
		var certData []byte
		certData, err = os.ReadFile(serverOptions.TLS.CertPEM)
		if err != nil {
			return nil, err
		}
		var keyData []byte
		keyData, err = os.ReadFile(serverOptions.TLS.KeyPEM)
		if err != nil {
			return nil, err
		}
		var cert tls.Certificate
		cert, err = tls.X509KeyPair(certData, keyData)
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
		httpServer.stdlibServer = NewStdlibServer(h, &serverOptions, tlsConfig, tracers)
	}
	h.OnShutdown = append(h.OnShutdown, func(_ context.Context) {
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

// Run starts the HTTP server and blocks until it stops.
func (s *HTTPServer) Run() {
	slog.Info(
		"starting server",
		"id",
		s.options.ID,
		"bind",
		s.options.Bind,
		"transporter",
		s.server.GetTransporterName(),
	)
	if s.stdlibServer != nil {
		l := s.listener
		if s.stdlibServer.TLSConfig != nil {
			l = tls.NewListener(l, s.stdlibServer.TLSConfig)
		}
		_ = s.stdlibServer.Serve(l)
	} else {
		_ = s.server.Run()
	}
}

// Shutdown stops the HTTP server gracefully.
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	s.isActive.Store(false)
	var err error
	if s.stdlibServer != nil {
		err = s.stdlibServer.Shutdown(ctx)
		_ = s.server.Shutdown(ctx) // Ensure Hertz internals stop
	} else {
		err = s.server.Shutdown(ctx)
	}
	return err
}

// Bind returns the address the server is bound to.
func (s *HTTPServer) Bind() string {
	return s.options.Bind
}

// SetEngine updates the request processing engine for the server.
func (s *HTTPServer) SetEngine(engine *Engine) {
	s.switcher.SetEngine(engine)
}

// Engine returns the current request processing engine.
func (s *HTTPServer) Engine() *Engine {
	return s.switcher.Engine()
}
