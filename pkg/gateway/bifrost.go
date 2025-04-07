package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/tracer"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/dns"
	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/bifrost/pkg/tracer/accesslog"
	"github.com/nite-coder/bifrost/pkg/tracer/prometheus"
	"github.com/nite-coder/bifrost/pkg/zero"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Bifrost struct {
	state          uint32
	tracerProvider *sdktrace.TracerProvider
	options        *config.Options
	resolver       *dns.Resolver
	zero           *zero.ZeroDownTime
	middlewares    map[string]app.HandlerFunc
	services       map[string]*Service
	httpServers    map[string]*HTTPServer
}

// Run starts all HTTP servers in the Bifrost instance. The last server is started
// in the current goroutine, and the function will block until the last server
// stops. The other servers are started in separate goroutines.
func (b *Bifrost) Run() {
	i := 0
	for _, server := range b.httpServers {
		if i == len(b.httpServers)-1 {
			// last server need to blocked
			server.Run()
			return
		}
		go server.Run()
		i++
	}
}

func (b *Bifrost) ZeroDownTime() *zero.ZeroDownTime {
	return b.zero
}

// Shutdown shuts down all HTTP servers in the Bifrost instance gracefully.
// It blocks until all servers finish their shutdown process.
// The function returns an error if any server fails to shut down.
func (b *Bifrost) Shutdown(ctx context.Context) error {
	return b.shutdown(ctx, false)
}

func (b *Bifrost) ShutdownNow(ctx context.Context) error {
	return b.shutdown(ctx, true)
}

// Service retrieves a service by its ID from the Bifrost instance.
// It returns a pointer to the Service and a boolean indicating whether
// the service was found. If the serviceID does not exist, the returned
// Service pointer will be nil and the boolean will be false.
func (b *Bifrost) Service(serviceID string) (*Service, bool) {
	result, found := b.services[serviceID]
	return result, found
}

// IsActive returns whether the Bifrost is active or not. It returns true if the Bifrost is active, false otherwise.
func (b *Bifrost) IsActive() bool {
	return atomic.LoadUint32(&b.state) == 1
}

// SetActive sets whether the Bifrost is active or not. If the value is true, Bifrost will be set to active, otherwise it will be set to inactive.
func (b *Bifrost) SetActive(value bool) {
	if value {
		atomic.StoreUint32(&b.state, 1)
	} else {
		atomic.StoreUint32(&b.state, 0)
	}
}

func (b *Bifrost) shutdown(ctx context.Context, now bool) error {
	b.SetActive(false)

	if b.tracerProvider != nil {
		_ = b.tracerProvider.Shutdown(ctx)
	}

	wg := &sync.WaitGroup{}
	maxTimeout := 10 * time.Second

	for _, server := range b.httpServers {
		wg.Add(1)

		if server.options.Timeout.Graceful > 0 && server.options.Timeout.Graceful > maxTimeout {
			maxTimeout = server.options.Timeout.Graceful
		}

		go func(srv *HTTPServer) {
			defer wg.Done()

			if srv.options.Timeout.Graceful > 0 {
				ctx, cancel := context.WithTimeout(ctx, server.options.Timeout.Graceful)
				defer cancel()
				_ = srv.Shutdown(ctx)
			} else {
				_ = srv.Shutdown(ctx)
			}

			slog.Debug("http server shutdown", "id", srv.options.ID)
		}(server)
	}

	if now {
		maxTimeout = 500 * time.Millisecond
	}
	waitTimeout(wg, maxTimeout)
	return b.zero.Close(ctx)
}

// NewBifrost creates a new instance of Bifrost.
//
// It takes in two parameters: mainOptions of type config.Options and isReload of type bool.
// The mainOptions parameter contains the main configuration options for the Bifrost instance.
// The isReload parameter indicates whether the function is being called during a reload operation.
// The function returns a pointer to Bifrost and an error.
func NewBifrost(mainOptions config.Options, isReload bool) (*Bifrost, error) {
	tCache := timecache.New(mainOptions.TimerResolution)
	timecache.Set(tCache)

	// system logger
	logger, err := log.NewLogger(mainOptions.Logging)
	if err != nil {
		return nil, err
	}
	slog.SetDefault(logger)

	zeroOptions := zero.Options{
		UpgradeSock: mainOptions.UpgradeSock,
		PIDFile:     mainOptions.PIDFile,
	}

	resolveOptions := dns.Options{
		Servers:  mainOptions.Resolvers,
		SkipTest: mainOptions.SkipResolver,
	}

	dnsResolver, err := dns.NewResolver(resolveOptions)
	if err != nil {
		return nil, err
	}

	middlewares, err := loadMiddlewares(mainOptions.Middlewares)
	if err != nil {
		return nil, err
	}

	bifrost := &Bifrost{
		options:     &mainOptions,
		middlewares: middlewares,
		resolver:    dnsResolver,
		httpServers: make(map[string]*HTTPServer),
		zero:        zero.New(zeroOptions),
	}

	// services
	services, err := loadServices(bifrost)
	if err != nil {
		return nil, err
	}
	bifrost.services = services

	tracers := []tracer.Tracer{}

	// prometheus
	if mainOptions.Metrics.Prometheus.Enabled && !isReload {
		promOpts := []prometheus.Option{}

		if len(mainOptions.Metrics.Prometheus.Buckets) > 0 {
			promOpts = append(promOpts, prometheus.WithHistogramBuckets(mainOptions.Metrics.Prometheus.Buckets))
		}

		promTracer := prometheus.NewTracer(promOpts...)
		tracers = append(tracers, promTracer)
	}

	// access log
	accessLogTracers := map[string]*accesslog.Tracer{}
	if len(mainOptions.AccessLogs) > 0 && !isReload {

		for id, accessLogOptions := range mainOptions.AccessLogs {
			accessLogTracer, err := accesslog.NewTracer(accessLogOptions)
			if err != nil {
				return nil, err
			}

			if accessLogTracer != nil {
				accessLogTracers[id] = accessLogTracer
			}
		}
	}

	if mainOptions.Tracing.Enabled {
		// otel tracing
		tp, err := initTracerProvider(mainOptions.Tracing)
		if err != nil {
			return nil, err
		}
		bifrost.tracerProvider = tp
	}

	for id, server := range mainOptions.Servers {
		if id == "" {
			return nil, errors.New("http server id can't be empty")
		}

		server.ID = id

		if server.Bind == "" {
			return nil, errors.New("http server bind can't be empty")
		}

		_, found := bifrost.httpServers[id]
		if found {
			return nil, fmt.Errorf("http server '%s' already exists", id)
		}

		// deep copy; each server has only one access logger tracer
		myTracers := make([]tracer.Tracer, len(tracers))
		copy(myTracers, tracers)

		if len(server.AccessLogID) > 0 {
			_, found := mainOptions.AccessLogs[server.AccessLogID]
			if !found {
				return nil, fmt.Errorf("access log '%s' was not found in server '%s'", server.AccessLogID, server.ID)
			}

			accessLogTracer, found := accessLogTracers[server.AccessLogID]
			if found {
				myTracers = append(myTracers, accessLogTracer)
			}
		}

		httpServer, err := newHTTPServer(bifrost, server, myTracers, isReload)
		if err != nil {
			return nil, err
		}

		bifrost.httpServers[id] = httpServer
	}

	bifrost.SetActive(true)
	return bifrost, nil
}
