package proxy

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"
)

// ErrMaxFailedCount is returned when the maximum number of failed proxy attempts is reached.
var ErrMaxFailedCount = errors.New("proxy: reach max failed count")

// Proxy defines the interface for a proxy service.
type Proxy interface {
	ID() string
	Target() string
	Endpoint() *Endpoint
	SetEndpoint(ep *Endpoint)
	ServeHTTP(c context.Context, ctx *app.RequestContext)
	Close() error
}
