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

	_ "github.com/nite-coder/bifrost/pkg/balancer/roundrobin"
	"github.com/nite-coder/bifrost/pkg/balancer/weighted"
	"github.com/nite-coder/bifrost/pkg/config"
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

	h.GET("/proxy/long-task", func(_ context.Context, ctx *app.RequestContext) {
		time.Sleep(5 * time.Second)
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

	dnsResolver, err := resolver.NewResolver(resolver.Options{})
	require.NoError(t, err)

	bifrost := &Bifrost{
		resolver: dnsResolver,
		options: &config.Options{
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
		},
	}

	ctx := context.Background()

	// direct proxy
	service, err := newService(bifrost, bifrost.options.Services["testService"])
	require.NoError(t, err)
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	service.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, backendResponse, string(hzCtx.Response.Body()))

	// exist upstream
	serviceOpts := bifrost.options.Services["testService"]
	serviceOpts.URL = "http://testUpstream"
	service, err = newService(bifrost, serviceOpts)
	require.NoError(t, err)

	hzCtx = app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	service.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, backendResponse, string(hzCtx.Response.Body()))

	serviceOpts.URL = "http://test_upstream_no_port:8088"
	service, err = newService(bifrost, serviceOpts)
	require.NoError(t, err)
	hzCtx = app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	service.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, backendResponse, string(hzCtx.Response.Body()))

	// dynamic upstream
	serviceOpts = bifrost.options.Services["testService"]
	serviceOpts.URL = "http://$var.test"
	service, err = newService(bifrost, serviceOpts)
	require.NoError(t, err)

	hzCtx = app.NewContext(0)
	hzCtx.Set("test", "testUpstream")
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	service.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, backendResponse, string(hzCtx.Response.Body()))
}

func TestDynamicService(t *testing.T) {
	h := testServer(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = h.Shutdown(ctx)
	}()

	dnsResolver, err := resolver.NewResolver(resolver.Options{})
	require.NoError(t, err)

	bifrost := &Bifrost{
		resolver: dnsResolver,
		options: &config.Options{
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
		},
	}

	ctx := context.Background()
	services, err := loadServices(bifrost)
	require.NoError(t, err)

	dynamicService := newDynamicService("$var.myservice", services)

	hzCtx := app.NewContext(0)
	hzCtx.Set("myservice", "testService")
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	dynamicService.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, backendResponse, string(hzCtx.Response.Body()))
}

func TestDynamicServiceMiddleware(t *testing.T) {
	dnsResolver, err := resolver.NewResolver(resolver.Options{})
	require.NoError(t, err)

	bifrost := &Bifrost{
		resolver: dnsResolver,
		options: &config.Options{
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
		},
	}

	hit := 0
	bifrost.middlewares = map[string]app.HandlerFunc{
		"testMiddleware": func(_ context.Context, c *app.RequestContext) {
			hit++
			c.Abort()
		},
	}

	ctx := context.Background()
	services, err := loadServices(bifrost)
	require.NoError(t, err)

	dynamicService := newDynamicService("$var.myservice", services)

	hzCtx := app.NewContext(0)
	hzCtx.Set("myservice", "testService")
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	dynamicService.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, 1, hit)

	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	dynamicService.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, 2, hit)

	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	dynamicService.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, 3, hit)
}

// TestServiceNoUpstream verifies that the service handles nil upstream gracefully
// When upstream is nil, the code path hits c.Error(nil) which panics.
// The panic is recovered and returns 500.
func TestServiceNoUpstream(t *testing.T) {
	service := &Service{
		options:  &config.ServiceOptions{ID: "test-service"},
		upstream: nil, // No upstream configured
	}

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/test")

	service.ServeHTTP(ctx, hzCtx)

	// When upstream is nil, proxy is nil, c.Error(nil) causes panic which is recovered as 500
	assert.Equal(t, 500, hzCtx.Response.StatusCode())
}

// TestServiceBalancerNil verifies that the service returns 503 when balancer is nil.
func TestServiceBalancerNil(t *testing.T) {
	dnsResolver, err := resolver.NewResolver(resolver.Options{})
	require.NoError(t, err)

	// Create upstream with nil balancer
	upstream := &Upstream{
		bifrost: &Bifrost{
			options:  &config.Options{SkipResolver: true},
			resolver: dnsResolver,
		},
		options: &config.UpstreamOptions{
			ID: "test-upstream",
		},
		serviceOptions: &config.ServiceOptions{
			ID:  "test-service",
			URL: "http://test",
		},
		// balancer is not set (nil)
	}

	service := &Service{
		options:  &config.ServiceOptions{ID: "test-service"},
		upstream: upstream,
	}

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/test")

	service.ServeHTTP(ctx, hzCtx)

	// Should return 503 when balancer is nil
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
		serviceOptions: &config.ServiceOptions{
			URL: "http://test",
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

	serviceOpts := config.ServiceOptions{
		ID:   "aiService",
		Type: "ai",
		URL:  "http://$ai_model_name",
	}

	service, err := newService(bifrost, serviceOpts)
	require.NoError(t, err)

	// Verify type: ai handles dynamic upstream correctly
	assert.Equal(t, variable.ModelName, service.dynamicUpstream)

	// Verify that the virtual upstream was created with "ai:" prefix
	upstream, exists := service.upstreams["ai:gpt-4"]
	require.True(t, exists)
	assert.Equal(t, "ai:gpt-4", upstream.options.ID)

	// Verify targets and weights in the upstream options
	require.Len(t, upstream.options.Targets, 2)
	assert.Equal(t, "p1/gpt-4", upstream.options.Targets[0].Target)
	assert.Equal(t, uint32(3), upstream.options.Targets[0].Weight)
	assert.Equal(t, "p2/gpt-4", upstream.options.Targets[1].Target)
	assert.Equal(t, uint32(1), upstream.options.Targets[1].Weight)

	// Verify balancer exists and has the correct proxies
	b := upstream.Balancer()
	require.NotNil(t, b)
	proxies := b.Proxies()
	require.Len(t, proxies, 2)
	assert.Equal(t, "p1/gpt-4", proxies[0].Target())
	assert.Equal(t, uint32(3), proxies[0].Weight())
	assert.Equal(t, "p2/gpt-4", proxies[1].Target())
	assert.Equal(t, uint32(1), proxies[1].Weight())

	// Verify that the virtual upstream was created with "ai:" prefix and defaults to weighted balancer
	upstream2, exists2 := service.upstreams["ai:gpt-3.5-turbo"]
	require.True(t, exists2)
	assert.Equal(t, "ai:gpt-3.5-turbo", upstream2.options.ID)
	assert.Equal(t, "weighted", upstream2.options.Balancer.Type)

	b2 := upstream2.Balancer()
	require.NotNil(t, b2)
	proxies2 := b2.Proxies()
	require.Len(t, proxies2, 2)
	assert.Equal(t, "p1/gpt-3.5", proxies2[0].Target())
	assert.Equal(t, uint32(1), proxies2[0].Weight())
	assert.Equal(t, "p2/gpt-3.5", proxies2[1].Target())
	assert.Equal(t, uint32(1), proxies2[1].Weight())
}
