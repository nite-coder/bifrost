package gateway

import (
	"context"
	"http-benchmark/pkg/log"
	"log/slog"

	"github.com/cloudwego/hertz/pkg/app"
)

type DynamicUpstream struct {
	upstreams map[string]*Upstream
	name      string
}

func NewDynamicUpstream(upstreams map[string]*Upstream, name string) *DynamicUpstream {
	return &DynamicUpstream{
		name:      name,
		upstreams: upstreams,
	}
}

func (u *DynamicUpstream) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	logger := log.FromContext(c)

	upstreamName := ctx.GetString(u.name)

	if len(upstreamName) == 0 {
		logger.Warn("upstream is not found", slog.String("name", u.name))
		ctx.Abort()
		return
	}

	upstream, found := u.upstreams[upstreamName]
	if !found {
		logger.Warn("upstream is not found", slog.String("name", upstreamName))
		ctx.Abort()
		return
	}

	upstream.ServeHTTP(c, ctx)
}
