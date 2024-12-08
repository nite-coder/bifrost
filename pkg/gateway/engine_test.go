package gateway

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/stretchr/testify/assert"
)

func TestMiddlewarePipeline(t *testing.T) {

	bifrost := &Bifrost{
		options: &config.Options{
			Servers: map[string]config.ServerOptions{
				"testServer": {
					ID: "testServer",
					Middlewares: []config.MiddlwareOptions{
						{
							Use: "testMiddleware1",
						},
					},
				},
			},

			Routes: map[string]config.RouteOptions{
				"testRoute": {
					Paths: []string{"/test"},
					Middlewares: []config.MiddlwareOptions{
						{
							Use: "testMiddleware2",
						},
					},
					ServiceID: "testService",
				},
			},

			Services: map[string]config.ServiceOptions{
				"testService": {
					Url: "http://127.0.0.1:8088",
					Middlewares: []config.MiddlwareOptions{
						{
							Use: "testMiddleware3",
						},
					},
				},
			},
		},
	}

	hit1 := 0
	hit2 := 0
	hit3 := 0
	bifrost.middlewares = map[string]app.HandlerFunc{
		"testMiddleware1": func(ctx context.Context, c *app.RequestContext) {
			hit1++
		},
		"testMiddleware2": func(ctx context.Context, c *app.RequestContext) {
			hit2++
		},
		"testMiddleware3": func(ctx context.Context, c *app.RequestContext) {
			hit3++
			c.Abort()
		},
	}

	// services
	services, err := loadServices(bifrost)
	assert.NoError(t, err)
	bifrost.services = services

	NewBifrost(*bifrost.options, false)

	engine, err := newEngine(bifrost, bifrost.options.Servers["testServer"])
	assert.NoError(t, err)

	hzCtx := app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/test")
	engine.ServeHTTP(context.Background(), hzCtx)
	assert.Equal(t, 1, hit1)
	assert.Equal(t, 1, hit2)
	assert.Equal(t, 1, hit3)

	// get variables
	serverID := variable.GetString(variable.ServerID, hzCtx)
	assert.Equal(t, "testServer", serverID)

	routeID := variable.GetString(variable.RouteID, hzCtx)
	assert.Equal(t, "testRoute", routeID)

	serviceID := variable.GetString(variable.ServiceID, hzCtx)
	assert.Equal(t, "testService", serviceID)
}
