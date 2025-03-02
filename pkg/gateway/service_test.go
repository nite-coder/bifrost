package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/dns"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/stretchr/testify/assert"
)

const (
	backendResponse = "I am the backend"
)

func testServer() *server.Hertz {
	const backendResponse = "I am the backend"
	const backendStatus = 200
	h := server.Default(
		server.WithHostPorts(":8088"),
		server.WithExitWaitTime(1*time.Second),
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		server.WithSenseClientDisconnection(true),
	)

	h.GET("/proxy/backend", func(cc context.Context, ctx *app.RequestContext) {
		ctx.Data(backendStatus, "application/json", []byte(backendResponse))
	})

	h.GET("/proxy/long-task", func(cc context.Context, ctx *app.RequestContext) {
		time.Sleep(5 * time.Second)
		ctx.Data(backendStatus, "application/json", []byte(backendResponse))
	})

	go h.Spin()
	time.Sleep(time.Second)
	return h
}

func TestServices(t *testing.T) {
	h := testServer()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = h.Shutdown(ctx)
	}()

	dnsResolver, err := dns.NewResolver(dns.Options{})
	assert.NoError(t, err)

	bifrost := &Bifrost{
		resolver: dnsResolver,
		options: &config.Options{
			Services: map[string]config.ServiceOptions{
				"testService": {
					Url: "http://127.0.0.1:8088",
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
	assert.NoError(t, err)
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	service.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, backendResponse, string(hzCtx.Response.Body()))

	// exist upstream
	serviceOpts := bifrost.options.Services["testService"]
	serviceOpts.Url = "http://testUpstream"
	service, err = newService(bifrost, serviceOpts)
	assert.NoError(t, err)

	hzCtx = app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	service.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, backendResponse, string(hzCtx.Response.Body()))

	serviceOpts.Url = "http://test_upstream_no_port:8088"
	service, err = newService(bifrost, serviceOpts)
	assert.NoError(t, err)
	hzCtx = app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	service.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, backendResponse, string(hzCtx.Response.Body()))

	// dynamic upstream
	serviceOpts = bifrost.options.Services["testService"]
	serviceOpts.Url = "http://$test"
	service, err = newService(bifrost, serviceOpts)
	assert.NoError(t, err)

	hzCtx = app.NewContext(0)
	hzCtx.Set("$test", "testUpstream")
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	service.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, backendResponse, string(hzCtx.Response.Body()))
}

func TestDynamicService(t *testing.T) {
	h := testServer()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = h.Shutdown(ctx)
	}()

	dnsResolver, err := dns.NewResolver(dns.Options{})
	assert.NoError(t, err)

	bifrost := &Bifrost{
		resolver: dnsResolver,
		options: &config.Options{
			Services: map[string]config.ServiceOptions{
				"testService": {
					Url: "http://127.0.0.1:8088",
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
	assert.NoError(t, err)

	dynamicService := newDynamicService("$dd", services)

	hzCtx := app.NewContext(0)
	hzCtx.Set("$dd", "testService")
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	dynamicService.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, backendResponse, string(hzCtx.Response.Body()))
}

func TestDynamicServiceMiddleware(t *testing.T) {
	dnsResolver, err := dns.NewResolver(dns.Options{})
	assert.NoError(t, err)

	bifrost := &Bifrost{
		resolver: dnsResolver,
		options: &config.Options{
			Services: map[string]config.ServiceOptions{
				"testService": {
					Url: "http://127.0.0.1:8088",
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
		"testMiddleware": func(ctx context.Context, c *app.RequestContext) {
			hit++
			c.Abort()
		},
	}

	ctx := context.Background()
	services, err := loadServices(bifrost)
	assert.NoError(t, err)

	dynamicService := newDynamicService("$dd", services)
	hzCtx := app.NewContext(0)
	hzCtx.Set("$dd", "testService")
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	dynamicService.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, 1, hit)
}
