package main

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

type Engine struct {
	handlers        app.HandlersChain
	notFoundHandler app.HandlerFunc
}

func NewEngine() *Engine {
	return &Engine{
		handlers: make([]app.HandlerFunc, 0),
	}
}

func (e *Engine) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	ctx.SetIndex(-1)
	ctx.SetHandlers(e.handlers)
	ctx.Next(c)
}

func (e *Engine) Use(middleware ...app.HandlerFunc) {
	e.handlers = append(e.handlers, middleware...)
}
