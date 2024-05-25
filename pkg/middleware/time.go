package middleware

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

type TimeMiddleware struct {
}

func NewTimeMiddleware() *TimeMiddleware {
	return &TimeMiddleware{}
}

func (t *TimeMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	startTime := time.Now().UTC().UnixMicro()

	ctx.Next(c)

	endTime := time.Now().UTC().UnixMicro()

	ctx.Response.Header.Add("X-In-Time", strconv.FormatInt(startTime, 10))
	ctx.Response.Header.Add("X-In-Out", strconv.FormatInt(endTime, 10))
}
