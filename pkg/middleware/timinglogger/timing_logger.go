package timinglogger

import (
	"context"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

func init() {
	_ = middleware.RegisterMiddleware("timing_logger", func(param map[string]any) (app.HandlerFunc, error) {
		m := NewMiddleware()
		return m.ServeHTTP, nil
	})
}

type TimingLoggerMiddleware struct {
}

func NewMiddleware() *TimingLoggerMiddleware {
	return &TimingLoggerMiddleware{}
}

func (t *TimingLoggerMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	startTime := time.Now().UTC().UnixMicro()

	ctx.Next(c)

	endTime := time.Now().UTC().UnixMicro()

	ctx.Response.Header.Add("X-Time-In", strconv.FormatInt(startTime, 10))
	ctx.Response.Header.Add("X-Time-Out", strconv.FormatInt(endTime, 10))
}
