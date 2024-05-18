package gateway

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/hlog"
)

type Engine struct {
	handlers        app.HandlersChain
	notFoundHandler app.HandlerFunc
}

func NewEngine() *Engine {
	return &Engine{
		handlers: make([]app.HandlerFunc, 0),
		notFoundHandler: func(c context.Context, ctx *app.RequestContext) {
			hlog.Info("bifrost: not found")
			//ctx.SetStatusCode(404)
			ctx.AbortWithStatus(404)
		},
	}
}

func (e *Engine) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	middlewares := append(e.handlers, e.notFoundHandler)
	ctx.SetIndex(-1)
	ctx.SetHandlers(middlewares)
	ctx.Next(c)
	ctx.Abort()
}

func (e *Engine) Use(middleware ...app.HandlerFunc) {
	e.handlers = append(e.handlers, middleware...)
}
