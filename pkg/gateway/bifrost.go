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
	"github.com/nite-coder/bifrost/internal/pkg/runtime"
	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/resolver"
	"github.com/nite-coder/bifrost/pkg/telemetry/metrics"
	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/bifrost/pkg/tracer/accesslog"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type Bifrost struct {
	tracerProvider  *sdktrace.TracerProvider
	metricsProvider *metrics.Provider
	options         *config.Options
	resolver        *resolver.Resolver
	runtime         *runtime.ZeroDownTime
	middlewares     map[string]app.HandlerFunc
	services        map[string]*Service
	httpServers     map[string]*HTTPServer
	state           uint32
}

// Run starts all HTTP servers in the Bifrost instance. The last server is started
// in the current goroutine, and the function will block until the last server
// stops. The other servers are started in separate goroutines.
func (b *Bifrost) Run() {
	i := 0
	for _, server := range b.httpServers {
		if i == len(b.httpServers)-1 {
			// last server need to blocked
			safety.Go(context.Background(), server.Run)
			return
		}
		go safety.Go(context.Background(), server.Run)
		i++
	}
}
func (b *Bifrost) ZeroDownTime() *runtime.ZeroDownTime {
	return b.runtime
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
	wg := &sync.WaitGroup{}
	maxTimeout := 10 * time.Second
	for _, server := range b.httpServers {
		wg.Add(1)
		if server.options.Timeout.Graceful > 0 && server.options.Timeout.Graceful > maxTimeout {
			maxTimeout = server.options.Timeout.Graceful
		}
		go func(srv *HTTPServer) {
			safety.Go(ctx, func() {
				defer wg.Done()
				if srv.options.Timeout.Graceful > 0 {
					ctx, cancel := context.WithTimeout(ctx, server.options.Timeout.Graceful)
					defer cancel()
					_ = srv.Shutdown(ctx)
				} else {
					_ = srv.Shutdown(ctx)
				}

				slog.Debug("http server shutdown", "id", srv.options.ID)
			})
		}(server)
	}

	if now {
		maxTimeout = 500 * time.Millisecond
	}
	waitTimeout(wg, maxTimeout)

	if b.tracerProvider != nil {
		_ = b.tracerProvider.Shutdown(ctx)
	}

	if b.metricsProvider != nil {
		_ = b.metricsProvider.Shutdown(ctx)
	}

	return b.runtime.Close(ctx)
}

func (b *Bifrost) Close() error {
	for _, service := range b.services {
		_ = service.Close()
	}
	if b.resolver != nil {
		b.resolver.Close()
	}
	return nil
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

	zeroOptions := runtime.Options{}
	resolveOptions := resolver.Options{
		Servers:  mainOptions.Resolver.Servers,
		SkipTest: mainOptions.SkipResolver,
	}
	dnsResolver, err := resolver.NewResolver(resolveOptions)
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
		runtime:     runtime.New(zeroOptions),
	}
	// services
	services, err := loadServices(bifrost)
	if err != nil {
		return nil, err
	}
	bifrost.services = services
	tracers := []tracer.Tracer{}
	// metrics (OTel with Prometheus and/or OTLP exporters)
	if (mainOptions.Metrics.Prometheus.Enabled || mainOptions.Metrics.OTLP.Enabled) && !isReload {
		metricsProvider, err := metrics.NewProvider(context.Background(), mainOptions.Metrics)
		if err != nil {
			return nil, fmt.Errorf("failed to create metrics provider: %w", err)
		}
		bifrost.metricsProvider = metricsProvider

		// Phase 1: Keep using legacy Prometheus tracer for core metrics
		// This ensures all existing metrics (request count, latency) are recorded
		// to the DefaultRegistry as before.
		// Since our Provider uses a Bridge (for Push) and MergedGatherer (for Pull)
		// that both read from DefaultGatherer, these metrics will be visible everywhere.
		promOpts := []metrics.Option{}
		if len(mainOptions.Metrics.Prometheus.Buckets) > 0 {
			promOpts = append(promOpts, metrics.WithHistogramBuckets(mainOptions.Metrics.Prometheus.Buckets))
		}
		promTracer := metrics.NewTracer(promOpts...)
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
			return nil, errors.New("HTTP server ID cannot be empty")
		}
		server.ID = id
		if server.Bind == "" {
			return nil, errors.New("HTTP server bind cannot be empty")
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
