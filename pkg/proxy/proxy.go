package proxy

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/target"
)

// Proxy forwards HTTP requests to a backend endpoint.
type Proxy interface {
	ID() string
	Target() string
	Endpoint() *target.Endpoint
	SetEndpoint(ep *target.Endpoint)
	ServeHTTP(c context.Context, ctx *app.RequestContext)
	Close() error
}
