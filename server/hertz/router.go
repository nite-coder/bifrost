package main

import (
	"context"
	"regexp"

	"github.com/cloudwego/hertz/pkg/app"
)

type Router struct {
	handlers app.HandlersChain
}

func NewRouter() *Router {
	return &Router{
		handlers: make([]app.HandlerFunc, 0),
	}
}

func (r *Router) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	ctx.SetIndex(-1)
	ctx.SetHandlers(r.handlers)
	ctx.Next(c)
}

func (r *Router) Use(middleware ...app.HandlerFunc) {
	r.handlers = append(r.handlers, middleware...)
}

func (r *Router) Regexp(expr string, middleware ...app.HandlerFunc) {
	regx, err := regexp.Compile(expr)
	if err != nil {
		panic(err)
	}

	r.Use(func(c context.Context, ctx *app.RequestContext) {
		if !regx.MatchString(string(ctx.Request.Path())) {
			return
		}

		ctx.SetIndex(-1)
		ctx.SetHandlers(middleware)
		ctx.Next(c)
		ctx.Abort()
	})
}
