package gateway

import (
	"context"
	"http-benchmark/pkg/config"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/stretchr/testify/assert"
)

const (
	backendResponse = "I am the backend"
)

func testServer() {
	const backendResponse = "I am the backend"
	const backendStatus = 200
	r := server.New(server.WithHostPorts("127.0.0.1:9990"))

	r.GET("/proxy/backend", func(cc context.Context, ctx *app.RequestContext) {
		ctx.Data(backendStatus, "application/json", []byte(backendResponse))
	})

	r.GET("/proxy/long-task", func(cc context.Context, ctx *app.RequestContext) {
		time.Sleep(5 * time.Second)
		ctx.Data(backendStatus, "application/json", []byte(backendResponse))
	})

	go r.Spin()
	time.Sleep(time.Second)
}

func TestServeHTTP(t *testing.T) {
	testServer()

	bifrost := &Bifrost{
		opts: &config.Options{
			Services: map[string]config.ServiceOptions{
				"testService": {
					Url: "http://localhost:9990",
				},
			},
			Upstreams: map[string]config.UpstreamOptions{
				"testUpstream": {
					ID: "testUpstream",
					Targets: []config.TargetOptions{
						{
							Target: "127.0.0.1:9990",
						},
					},
				},
			},
		},
	}

	ctx := context.Background()

	// direct proxy
	service, err := newService(bifrost, bifrost.opts.Services["testService"])
	assert.NoError(t, err)
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://localhost:9990/proxy/backend")
	service.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, backendResponse, string(hzCtx.Response.Body()))

	// exist upstream
	serviceOpts := bifrost.opts.Services["testService"]
	serviceOpts.Url = "http://testUpstream"
	service, err = newService(bifrost, serviceOpts)
	assert.NoError(t, err)

	hzCtx = app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://localhost:9990/proxy/backend")
	service.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, backendResponse, string(hzCtx.Response.Body()))

	// dynamic service
	serviceOpts = bifrost.opts.Services["testService"]
	serviceOpts.Url = "http://$test"
	service, err = newService(bifrost, serviceOpts)
	assert.NoError(t, err)

	hzCtx = app.NewContext(0)
	hzCtx.Set("$test", "testUpstream")
	hzCtx.Request.SetRequestURI("http://localhost:9990/proxy/backend")
	service.ServeHTTP(ctx, hzCtx)
	assert.Equal(t, backendResponse, string(hzCtx.Response.Body()))
}
