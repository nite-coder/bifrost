package main

import (
	"context"
	"sync/atomic"

	"github.com/cloudwego/hertz/pkg/app"
)

type Switcher struct {
	engine atomic.Value
}

func NewSwitcher(engine *Engine) *Switcher {
	s := &Switcher{}
	s.SetEngine(engine)
	return s
}

func (s *Switcher) Engine() *Engine {
	return s.engine.Load().(*Engine)
}

func (s *Switcher) SetEngine(engine *Engine) {
	s.engine.Store(engine)
}

func (s *Switcher) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	s.Engine().ServeHTTP(c, ctx)
}
