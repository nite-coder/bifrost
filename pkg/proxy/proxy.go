package proxy

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/target"
)

// ErrMaxFailedCount is returned when the proxy has reached the max failed count.
var ErrMaxFailedCount = errors.New("proxy: reach max failed count")

// Proxy forwards HTTP requests to a backend endpoint.
type Proxy interface {
	ID() string
	Target() string
	Endpoint() *target.Endpoint
	SetEndpoint(ep *target.Endpoint)
	ServeHTTP(c context.Context, ctx *app.RequestContext)
	Close() error
}
