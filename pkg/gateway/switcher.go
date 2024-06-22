package gateway

import (
	"context"
	"sync/atomic"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/config"
)

type switcher struct {
	engine atomic.Value
}

func newSwitcher(engine *Engine) *switcher {
	s := &switcher{}
	s.SetEngine(engine)
	return s
}

func (s *switcher) Engine() *Engine {
	return s.engine.Load().(*Engine)
}

func (s *switcher) SetEngine(engine *Engine) {
	s.engine.Store(engine)
}

func (s *switcher) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	s.Engine().ServeHTTP(c, ctx)
	ctx.Abort()
}

func withDefaultServerHeader(disable bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.NoDefaultServerHeader = disable
	}}
}
