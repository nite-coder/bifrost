package gateway

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

type LoggerTracer struct {
	template string
}

func NewLoggerTracer(template string) *LoggerTracer {
	return &LoggerTracer{
		template: strings.ReplaceAll(template, "\n", " "),
	}
}

func (t *LoggerTracer) Start(ctx context.Context, c *app.RequestContext) context.Context {
	time := time.Now()
	c.Set(REQUEST_START_TIME, time)
	return ctx
}

func (t *LoggerTracer) Finish(ctx context.Context, c *app.RequestContext) {
	val, found := c.Get(REQUEST_START_TIME)

	if !found {
		return
	}

	startTime := val.(time.Time)
	dur := time.Since(startTime)
	mic := dur.Microseconds()
	duration := float64(mic) / 1e6
	requestTime := strconv.FormatFloat(duration, 'f', -1, 64)

	replacer := strings.NewReplacer(
		"$time", startTime.Format("2006-01-02 15:04:05"),
		"$request", b2s(c.Request.Path()),
		"$upstream", "default",
		"$request_time", requestTime,
	)

	result := replacer.Replace(t.template)
	fmt.Println(result)
}
