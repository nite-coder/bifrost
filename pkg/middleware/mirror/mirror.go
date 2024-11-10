package mirror

import (
	"context"
	"log/slog"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/gateway"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

func init() {
	_ = middleware.RegisterMiddleware("mirror", func(param map[string]any) (app.HandlerFunc, error) {

		m := NewMiddleware("xyz")
		return m.ServeHTTP, nil
	})
}

type MirrorMiddleware struct {
	serviceID string
	queue     chan *app.RequestContext
}

func NewMiddleware(serviceID string) *MirrorMiddleware {
	m := &MirrorMiddleware{
		serviceID: serviceID,
		queue:     make(chan *app.RequestContext, 1000),
	}

	go m.Run()
	return m
}

func (m *MirrorMiddleware) Run() {
	for c := range m.queue {
		bifrost := gateway.GetBifrost()

		if bifrost == nil {
			continue
		}

		if !bifrost.IsActive() {
			break
		}

		slog.Info("mirror service", "service_id", m.serviceID)
		svc, found := bifrost.Service(m.serviceID)

		if !found {
			slog.Warn("mirror: service not found", "service_id", m.serviceID)
			continue
		}

		svc.ServeHTTP(context.Background(), c)
		slog.Info("mirror resp", "req", c.Request.URI().FullURI(), "method", c.Request.Method(), "req_body", c.Request.Body(), "status", c.Response.StatusCode(), "content", c.Response.Body())
	}
}

func (m *MirrorMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	newC := c.Copy()
	m.queue <- newC
	c.Next(ctx)
}
