package main

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

type TimingMiddleware struct {
}

func NewMiddleware() *TimingMiddleware {
	return &TimingMiddleware{}
}

func (t *TimingMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	startTime := time.Now().UTC().UnixMicro()

	ctx.Next(c)

	endTime := time.Now().UTC().UnixMicro()

	ctx.Response.Header.Add("x-time-in", strconv.FormatInt(startTime, 10))
	ctx.Response.Header.Add("x-time-out", strconv.FormatInt(endTime, 10))
}
