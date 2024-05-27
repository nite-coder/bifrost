package middleware

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

type TimingLoggerMiddleware struct {
}

func NewTimingLoggerMiddleware() *TimingLoggerMiddleware {
	return &TimingLoggerMiddleware{}
}

func (t *TimingLoggerMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	startTime := time.Now().UTC().UnixMicro()

	ctx.Next(c)

	endTime := time.Now().UTC().UnixMicro()

	ctx.Response.Header.Add("X-Time-In", strconv.FormatInt(startTime, 10))
	ctx.Response.Header.Add("X-Time-Out", strconv.FormatInt(endTime, 10))
}
