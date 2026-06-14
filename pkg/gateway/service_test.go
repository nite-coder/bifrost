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

	"github.com/nite-coder/bifrost/pkg/balancer/weighted"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/proxy"
	"github.com/nite-coder/bifrost/pkg/resolver"
	"github.com/nite-coder/bifrost/pkg/target"
	"github.com/nite-coder/bifrost/pkg/variable"
)

const (
	backendResponse  = "I am the backend"
	testBackendAddr  = "127.0.0.1:8080"
	testBackendAddr2 = "127.0.0.1:8081"
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

	var foundP1, foundP2 bool
	require.Eventually(t, func() bool {
		foundP1 = false
		foundP2 = false
		service.proxyByAddress.Range(func(_, v any) bool {
			p, ok := v.(proxy.Proxy)
			if !ok {
				return true
			}
			if p.Target() == "p1/gpt-4" {
				assert.Equal(t, uint32(3), p.Endpoint().Weight)
				foundP1 = true
			} else if p.Target() == "p2/gpt-4" {
				assert.Equal(t, uint32(1), p.Endpoint().Weight)
				foundP2 = true
			}
			return true
		})
		return foundP1 && foundP2
	}, time.Second, 5*time.Millisecond, "expected proxies for p1/gpt-4 and p2/gpt-4")

	upstream2, exists2 := service.bifrost.upstreamManager.Get("ai:gpt-3.5-turbo")
	require.True(t, exists2)
	assert.Equal(t, "ai:gpt-3.5-turbo", upstream2.options.ID)
	assert.Equal(t, "weighted", upstream2.options.Balancer.Type)

	b2, exists3 := service.getBalancer("ai:gpt-3.5-turbo")
	require.True(t, exists3)
	require.NotNil(t, b2)

	var foundP3, foundP4 bool
	require.Eventually(t, func() bool {
		foundP3 = false
		foundP4 = false
		service.proxyByAddress.Range(func(_, v any) bool {
			p, ok := v.(proxy.Proxy)
			if !ok {
				return true
			}
			if p.Target() == "p1/gpt-3.5" {
				assert.Equal(t, uint32(1), p.Endpoint().Weight)
				foundP3 = true
			} else if p.Target() == "p2/gpt-3.5" {
				assert.Equal(t, uint32(1), p.Endpoint().Weight)
				foundP4 = true
			}
			return true
		})
		return foundP3 && foundP4
	}, time.Second, 5*time.Millisecond, "expected proxies for p1/gpt-3.5 and p2/gpt-3.5")
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

	assert.False(t, upstream.isExclusive.Load())
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

func TestService_UpdateEndpoints(t *testing.T) {
	upstreamID := "test_upstream"

	svc := &Service{
		options:           &config.ServiceOptions{ID: "testService"},
		upstreamAddresses: make(map[string]map[string]bool),
	}

	addr1 := testBackendAddr
	ep1 := &target.Endpoint{
		Address: addr1,
		State:   target.NewState(2, time.Second),
	}

	// Pre-populate mock proxy for addr1
	var closed bool
	p1 := &mockProxyForUpdate{
		id:     "proxy1",
		target: "http://" + addr1,
		ep:     ep1,
		onClose: func() {
			closed = true
		},
	}
	svc.proxyByAddress.Store(addr1, p1)
	svc.upstreamAddresses[upstreamID] = map[string]bool{addr1: true}

	// First update: same address, should reuse and call SetEndpoint
	svc.updateEndpoints(upstreamID, []*target.Endpoint{ep1})
	assert.Equal(t, 1, p1.setEpCount)
	assert.False(t, closed, "proxy should not be closed when address is reused")

	// Second update: new address, old should be closed
	addr2 := testBackendAddr2
	ep2 := &target.Endpoint{
		Address: addr2,
		State:   target.NewState(2, time.Second),
	}

	p2 := &mockProxyForUpdate{
		id:     "proxy2",
		target: "http://" + addr2,
		ep:     ep2,
	}
	svc.proxyByAddress.Store(addr2, p2)

	svc.updateEndpoints(upstreamID, []*target.Endpoint{ep2})
	assert.True(t, closed, "old proxy should be closed after address replacement")

	_, ok := svc.proxyByAddress.Load(addr1)
	assert.False(t, ok, "addr1 proxy should be removed from proxyByAddress")

	v, ok := svc.proxyByAddress.Load(addr2)
	assert.True(t, ok, "addr2 proxy should exist in proxyByAddress")
	assert.Equal(t, p2, v)
}

func TestService_UpdateEndpoints_SharedAddress(t *testing.T) {
	svc := &Service{
		options:           &config.ServiceOptions{ID: "testService"},
		upstreamAddresses: make(map[string]map[string]bool),
	}

	addr := testBackendAddr
	ep1 := &target.Endpoint{
		Address: addr,
		State:   target.NewState(2, time.Second),
	}

	p1 := &mockProxyForUpdate{
		id: "proxy1",
		ep: ep1,
		onClose: func() {
			t.Error("proxy should not be closed when still used by another upstream")
		},
	}
	svc.proxyByAddress.Store(addr, p1)
	svc.upstreamAddresses["upstream1"] = map[string]bool{addr: true}

	// Update upstream1 with same address
	svc.updateEndpoints("upstream1", []*target.Endpoint{ep1})

	// Now upstream2 gets the same address, then loses it
	svc.upstreamAddresses["upstream2"] = map[string]bool{addr: true}
	svc.updateEndpoints("upstream2", []*target.Endpoint{ep1})

	// Remove address from upstream2
	svc.updateEndpoints("upstream2", []*target.Endpoint{})

	_, ok := svc.proxyByAddress.Load(addr)
	assert.True(t, ok, "proxy should still exist when upstream1 still uses it")
}
