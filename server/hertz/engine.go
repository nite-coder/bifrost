package main

import (
	"context"
	"sync/atomic"

	"github.com/cloudwego/hertz/pkg/app"
)

type Switcher struct {
	router atomic.Value
}

func NewSwitcher(router *Router) *Switcher {
	s := &Switcher{}
	s.SetRouter(router)
	return s
}

func (s *Switcher) Router() *Router {
	return s.router.Load().(*Router)
}

func (s *Switcher) SetRouter(router *Router) {
	s.router.Store(router)
}

func (s *Switcher) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	s.Router().ServeHTTP(c, ctx)
}
