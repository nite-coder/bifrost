package gateway

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/balancer"
	_ "github.com/nite-coder/bifrost/pkg/balancer/roundrobin"
	"github.com/nite-coder/bifrost/pkg/balancer/weighted"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/proxy"
	"github.com/nite-coder/bifrost/pkg/resolver"
	"github.com/nite-coder/bifrost/pkg/variable"
)

const (
	backendResponse = "I am the backend"
)

func testServer(t *testing.T) *server.Hertz {
	t.Helper()
	const backendResponse = "I am the backend"
	const backendStatus = 200
	h := server.Default(
		server.WithHostPorts(":8088"),
		server.WithExitWaitTime(1*time.Second),
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		server.WithSenseClientDisconnection(true),
	)

	h.GET("/proxy/backend", func(_ context.Context, ctx *app.RequestContext) {
		ctx.Data(backendStatus, "application/json", []byte(backendResponse))
	})

	go h.Spin()
	assert.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", "localhost:8088", 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return true
		}
		return false
	}, 10*time.Second, 100*time.Millisecond, "Server failed to start")
	return h
}

func TestClientCancelRequest(t *testing.T) {
	service := Service{}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	cancel()

	hzCtx := app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/test")
	service.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, int(499), hzCtx.Response.StatusCode())
}

func TestServices(t *testing.T) {
	h := testServer(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = h.Shutdown(ctx)
	}()

	options := config.Options{
		Services: map[string]config.ServiceOptions{
			"testService": {
				URL: "http://127.0.0.1:8088",
			},
		},
		Upstreams: map[string]config.UpstreamOptions{
			"testUpstream": {
				ID: "testUpstream",
				Targets: []config.TargetOptions{
					{
						Target: "127.0.0.1:8088",
					},
				},
			},

			"test_upstream_no_port": {
				ID: "test_upstream_no_port",
				Targets: []config.TargetOptions{
					{
						Target: "127.0.0.1",
					},
				},
			},
		},
	}

	bifrost, err := NewBifrost(options, ModeNormal)
	require.NoError(t, err)
	defer func() {
		_ = bifrost.Close()
	}()

	ctx := context.Background()

	// direct proxy
	service, err := newService(bifrost, bifrost.options.Services["testService"])
	require.NoError(t, err)

	hzCtx := app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	require.Eventually(t, func() bool {
		hzCtx.Response.Reset()
		service.ServeHTTP(ctx, hzCtx)
		return hzCtx.Response.StatusCode() == 200 && string(hzCtx.Response.Body()) == backendResponse
	}, time.Second, 5*time.Millisecond)

	// exist upstream
	serviceOpts := bifrost.options.Services["testService"]
	serviceOpts.URL = "http://testUpstream"
	service, err = newService(bifrost, serviceOpts)
	require.NoError(t, err)

	hzCtx = app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	require.Eventually(t, func() bool {
		hzCtx.Response.Reset()
		service.ServeHTTP(ctx, hzCtx)
		return hzCtx.Response.StatusCode() == 200 && string(hzCtx.Response.Body()) == backendResponse
	}, time.Second, 5*time.Millisecond)

	serviceOpts.URL = "http://test_upstream_no_port:8088"
	service, err = newService(bifrost, serviceOpts)
	require.NoError(t, err)

	hzCtx = app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	require.Eventually(t, func() bool {
		hzCtx.Response.Reset()
		service.ServeHTTP(ctx, hzCtx)
		return hzCtx.Response.StatusCode() == 200 && string(hzCtx.Response.Body()) == backendResponse
	}, time.Second, 5*time.Millisecond)

	// dynamic upstream
	serviceOpts = bifrost.options.Services["testService"]
	serviceOpts.URL = "http://$var.test"
	service, err = newService(bifrost, serviceOpts)
	require.NoError(t, err)

	hzCtx = app.NewContext(0)
	hzCtx.Set("test", "testUpstream")
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	require.Eventually(t, func() bool {
		hzCtx.Response.Reset()
		service.ServeHTTP(ctx, hzCtx)
		return hzCtx.Response.StatusCode() == 200 && string(hzCtx.Response.Body()) == backendResponse
	}, time.Second, 5*time.Millisecond)
}

func TestDynamicService(t *testing.T) {
	h := testServer(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = h.Shutdown(ctx)
	}()

	options := config.Options{
		Services: map[string]config.ServiceOptions{
			"testService": {
				URL: "http://127.0.0.1:8088",
			},
		},
		Upstreams: map[string]config.UpstreamOptions{
			"testUpstream": {
				ID: "testUpstream",
				Targets: []config.TargetOptions{
					{
						Target: "127.0.0.1:8088",
					},
				},
			},

			"test_upstream_no_port": {
				ID: "test_upstream_no_port",
				Targets: []config.TargetOptions{
					{
						Target: "127.0.0.1",
					},
				},
			},
		},
	}

	bifrost, err := NewBifrost(options, ModeNormal)
	require.NoError(t, err)
	defer func() {
		_ = bifrost.Close()
	}()

	services := bifrost.services
	dynamicService := newDynamicService("$var.myservice", services)

	hzCtx := app.NewContext(0)
	hzCtx.Set("myservice", "testService")
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	require.Eventually(t, func() bool {
		hzCtx.Response.Reset()
		dynamicService.ServeHTTP(context.Background(), hzCtx)
		return hzCtx.Response.StatusCode() == 200 && string(hzCtx.Response.Body()) == backendResponse
	}, time.Second, 5*time.Millisecond)
}

func TestDynamicServiceMiddleware(t *testing.T) {
	options := config.Options{
		Services: map[string]config.ServiceOptions{
			"testService": {
				URL: "http://127.0.0.1:8088",
				Middlewares: []config.MiddlwareOptions{
					{
						Use: "testMiddleware",
					},
				},
			},
		},
	}

	dnsResolver, err := resolver.NewResolver(resolver.Options{})
	require.NoError(t, err)

	bifrost := &Bifrost{
		resolver: dnsResolver,
		options:  &options,
	}

	hit := 0
	bifrost.middlewares = map[string]app.HandlerFunc{
		"testMiddleware": func(_ context.Context, c *app.RequestContext) {
			hit++
			c.Abort()
		},
	}

	services, err := loadServices(bifrost)
	require.NoError(t, err)
	bifrost.services = services

	dynamicService := newDynamicService("$var.myservice", services)

	hzCtx := app.NewContext(0)
	hzCtx.Set("myservice", "testService")
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	dynamicService.ServeHTTP(context.Background(), hzCtx)
	assert.Equal(t, 1, hit)

	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	dynamicService.ServeHTTP(context.Background(), hzCtx)
	assert.Equal(t, 2, hit)

	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	dynamicService.ServeHTTP(context.Background(), hzCtx)
	assert.Equal(t, 3, hit)
}

func TestServiceNoUpstream(t *testing.T) {
	service := &Service{
		options:  &config.ServiceOptions{ID: "test-service"},
		upstream: nil,
	}

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/test")

	service.ServeHTTP(ctx, hzCtx)

	assert.Equal(t, 500, hzCtx.Response.StatusCode())
}

func TestServiceBalancerNil(t *testing.T) {
	upstream := &Upstream{
		options: &config.UpstreamOptions{
			ID: "test-upstream",
		},
	}

	service := &Service{
		options:  &config.ServiceOptions{ID: "test-service"},
		upstream: upstream,
	}

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/test")

	service.ServeHTTP(ctx, hzCtx)

	assert.Equal(t, 503, hzCtx.Response.StatusCode())
}

func TestServiceGetters(t *testing.T) {
	dnsResolver, err := resolver.NewResolver(resolver.Options{})
	require.NoError(t, err)

	bifrost := &Bifrost{
		resolver: dnsResolver,
		options: &config.Options{
			SkipResolver: true,
		},
	}

	upstream := &Upstream{
		bifrost: bifrost,
		options: &config.UpstreamOptions{
			ID: "test-upstream",
		},
	}

	mockHandler := func(_ context.Context, _ *app.RequestContext) {}

	service := &Service{
		options:     &config.ServiceOptions{ID: "test-service"},
		upstream:    upstream,
		middlewares: []app.HandlerFunc{mockHandler},
	}

	t.Run("Upstream getter", func(t *testing.T) {
		u := service.Upstream()
		assert.NotNil(t, u)
		assert.Equal(t, upstream, u)
	})

	t.Run("Middlewares getter", func(t *testing.T) {
		middlewares := service.Middlewares()
		assert.NotNil(t, middlewares)
		assert.Len(t, middlewares, 1)
	})
}

func TestLoadModels(t *testing.T) {
	err := weighted.Init()
	require.NoError(t, err)

	dnsResolver, err := resolver.NewResolver(resolver.Options{})
	require.NoError(t, err)

	bifrost := &Bifrost{
		resolver: dnsResolver,
		options: &config.Options{
			SkipResolver: true,
			Models: map[string]*config.AIModelOptions{
				"gpt-4": {
					Balancer: &config.AIBalancerOptions{Type: "weighted"},
					Targets: []config.AITargetOptions{
						{Target: "p1/gpt-4", Weight: 3},
						{Target: "p2/gpt-4", Weight: 1},
					},
				},
				"gpt-3.5-turbo": {
					Targets: []config.AITargetOptions{
						{Target: "p1/gpt-3.5", Weight: 0},
						{Target: "p2/gpt-3.5"},
					},
				},
			},
		},
	}

	bifrost.upstreamManager = newUpstreamManager(bifrost)
	err = bifrost.upstreamManager.Start()
	require.NoError(t, err)
	defer func() {
		_ = bifrost.upstreamManager.Close()
	}()

	serviceOpts := config.ServiceOptions{
		ID:   "aiService",
		Type: "ai",
		URL:  "http://$ai_model_name",
	}

	service, err := newService(bifrost, serviceOpts)
	require.NoError(t, err)

	// Wait for subscriptions to propagate
	require.Eventually(t, func() bool {
		_, exists := service.getBalancer("ai:gpt-4")
		return exists
	}, time.Second, 5*time.Millisecond)

	assert.Equal(t, variable.Model, service.dynamicUpstream)

	upstream, exists := service.bifrost.upstreamManager.Get("ai:gpt-4")
	require.True(t, exists)
	assert.Equal(t, "ai:gpt-4", upstream.options.ID)

	require.Len(t, upstream.options.Targets, 2)
	assert.Equal(t, "p1/gpt-4", upstream.options.Targets[0].Target)
	assert.Equal(t, uint32(3), upstream.options.Targets[0].Weight)
	assert.Equal(t, "p2/gpt-4", upstream.options.Targets[1].Target)
	assert.Equal(t, uint32(1), upstream.options.Targets[1].Weight)

	b, exists := service.getBalancer("ai:gpt-4")
	require.True(t, exists)
	require.NotNil(t, b)
	proxies := b.Proxies()
	require.Len(t, proxies, 2)
	assert.Equal(t, "p1/gpt-4", proxies[0].Target())
	assert.Equal(t, uint32(3), proxies[0].Endpoint().Weight)
	assert.Equal(t, "p2/gpt-4", proxies[1].Target())
	assert.Equal(t, uint32(1), proxies[1].Endpoint().Weight)

	upstream2, exists2 := service.bifrost.upstreamManager.Get("ai:gpt-3.5-turbo")
	require.True(t, exists2)
	assert.Equal(t, "ai:gpt-3.5-turbo", upstream2.options.ID)
	assert.Equal(t, "weighted", upstream2.options.Balancer.Type)

	b2, exists3 := service.getBalancer("ai:gpt-3.5-turbo")
	require.True(t, exists3)
	require.NotNil(t, b2)
	proxies2 := b2.Proxies()
	require.Len(t, proxies2, 2)
	assert.Equal(t, "p1/gpt-3.5", proxies2[0].Target())
	assert.Equal(t, uint32(1), proxies2[0].Endpoint().Weight)
	assert.Equal(t, "p2/gpt-3.5", proxies2[1].Target())
	assert.Equal(t, uint32(1), proxies2[1].Endpoint().Weight)
}

func TestSharedUpstreamLifecycle(t *testing.T) {
	h := testServer(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = h.Shutdown(ctx)
	}()

	options := config.Options{
		Services: map[string]config.ServiceOptions{
			"service1": {
				ID:  "service1",
				URL: "http://testUpstream",
			},
			"service2": {
				ID:  "service2",
				URL: "http://testUpstream",
			},
		},
		Upstreams: map[string]config.UpstreamOptions{
			"testUpstream": {
				ID: "testUpstream",
				Targets: []config.TargetOptions{
					{
						Target: "127.0.0.1:8088",
					},
				},
			},
		},
	}

	bifrost, err := NewBifrost(options, ModeNormal)
	require.NoError(t, err)
	defer func() {
		_ = bifrost.Close()
	}()

	service1 := bifrost.services["service1"]
	require.NotNil(t, service1)

	service2 := bifrost.services["service2"]
	require.NotNil(t, service2)

	upstream, exists := bifrost.upstreamManager.Get("testUpstream")
	require.True(t, exists)

	upstream.mu.RLock()
	require.Len(t, upstream.subscribers, 2)
	upstream.mu.RUnlock()

	err = service1.Close()
	require.NoError(t, err)

	assert.False(t, upstream.isExclusive)
	assert.NotNil(t, upstream.cancel)

	upstream.mu.RLock()
	require.Len(t, upstream.subscribers, 1)
	upstream.mu.RUnlock()

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	require.Eventually(t, func() bool {
		hzCtx.Response.Reset()
		service2.ServeHTTP(ctx, hzCtx)
		return hzCtx.Response.StatusCode() == 200 && string(hzCtx.Response.Body()) == backendResponse
	}, time.Second, 5*time.Millisecond)

	_ = service2.Close()
}

type mockProxyForUpdate struct {
	id         string
	target     string
	ep         *proxy.Endpoint
	onClose    func()
	setEpCount int
}

func (m *mockProxyForUpdate) ID() string                                         { return m.id }
func (m *mockProxyForUpdate) Target() string                                     { return m.target }
func (m *mockProxyForUpdate) Endpoint() *proxy.Endpoint                          { return m.ep }
func (m *mockProxyForUpdate) SetEndpoint(ep *proxy.Endpoint)                     { m.ep = ep; m.setEpCount++ }
func (m *mockProxyForUpdate) ServeHTTP(_ context.Context, _ *app.RequestContext) {}
func (m *mockProxyForUpdate) Close() error {
	if m.onClose != nil {
		m.onClose()
	}
	return nil
}

type mockBalancer struct {
	proxies []proxy.Proxy
}

func (b *mockBalancer) Proxies() []proxy.Proxy { return b.proxies }
func (b *mockBalancer) Select(_ context.Context, _ *app.RequestContext) (proxy.Proxy, error) {
	if len(b.proxies) == 0 {
		return nil, balancer.ErrNotAvailable
	}
	return b.proxies[0], nil
}

func TestService_AtomicBalancerUpdate(t *testing.T) {
	// 1. Register a mock balancer handler
	err := balancer.Register([]string{"mock_atomic"}, func(proxies []proxy.Proxy, _ any) (balancer.Balancer, error) {
		return &mockBalancer{proxies: proxies}, nil
	})
	require.NoError(t, err)

	// 2. Setup a Service with mock balancer type
	bifrost := &Bifrost{
		options: &config.Options{},
	}

	upstreamID := "test_atomic_upstream"
	upstreamOpts := config.UpstreamOptions{
		ID: upstreamID,
		Balancer: config.BalancerOptions{
			Type: "mock_atomic",
		},
		Targets: []config.TargetOptions{
			{Target: "127.0.0.1:8080"},
		},
	}
	u, err := newUpstream(bifrost, upstreamOpts)
	require.NoError(t, err)
	defer func() {
		_ = u.Close()
	}()

	svc := &Service{
		bifrost:         bifrost,
		options:         &config.ServiceOptions{ID: "testService", URL: "http://" + upstreamID},
		upstream:        u,
		balancers:       make(map[string]balancer.Balancer),
		activeProxies:   make(map[string]proxy.Proxy),
		upstreamProxies: make(map[string]map[string]proxy.Proxy),
	}

	// First update: initial endpoints
	ep1 := &proxy.Endpoint{
		Address:     "127.0.0.1:8080",
		HealthState: proxy.NewTargetState(2, time.Second),
	}

	// Create mock proxy for ep1
	targetURL1 := "http://127.0.0.1:8080"
	hash1 := generateServiceProxyHash(targetURL1, nil)
	p1 := &mockProxyForUpdate{
		id:     "proxy1",
		target: targetURL1,
		ep:     ep1,
	}
	svc.activeProxies[hash1] = p1
	svc.upstreamProxies[upstreamID] = map[string]proxy.Proxy{
		hash1: p1,
	}

	// Call updateEndpoints (this should trigger creation of mockBalancer containing p1)
	svc.updateEndpoints(upstreamID, []*proxy.Endpoint{ep1})

	bal1Val := svc.balancer.Load()
	require.NotNil(t, bal1Val)
	bal1, ok := bal1Val.(balancer.Balancer)
	require.True(t, ok)
	assert.Same(t, p1, bal1.Proxies()[0])

	// Second update: ep1 is replaced by ep2
	ep2 := &proxy.Endpoint{
		Address:     "127.0.0.1:8081",
		HealthState: proxy.NewTargetState(2, time.Second),
	}

	targetURL2 := "http://127.0.0.1:8081"
	hash2 := generateServiceProxyHash(targetURL2, nil)
	p2 := &mockProxyForUpdate{
		id:     "proxy2",
		target: targetURL2,
		ep:     ep2,
	}
	// Pre-populate p2 so updateEndpoints doesn't try to create a real ReverseProxy
	svc.activeProxies[hash2] = p2

	// Set onClose assertion: when p1 is closed, the active balancer must already be bal2
	var closedAsserted bool
	p1.onClose = func() {
		currentBalVal := svc.balancer.Load()
		require.NotNil(t, currentBalVal)
		currentBal, ok := currentBalVal.(balancer.Balancer)
		require.True(t, ok)
		// It must be bal2 (which contains p2, not p1)
		assert.Same(t, p2, currentBal.Proxies()[0])
		closedAsserted = true
	}

	svc.updateEndpoints(upstreamID, []*proxy.Endpoint{ep2})

	assert.True(t, closedAsserted, "Expected onClose to be called and assert the new balancer was already stored")
}
