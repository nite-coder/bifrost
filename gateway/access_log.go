package gateway

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
)

var spaceByte = []byte{byte(' ')}

type LoggerTracer struct {
	template string
	logChan  chan string
}

func NewLoggerTracer(template string) *LoggerTracer {
	words := strings.Fields(template)

	tracer := &LoggerTracer{
		template: strings.Join(words, " "),
		logChan:  make(chan string, 10000),
	}

	go func() {
		for {
			select {
			case <-tracer.logChan:

			}
		}
	}()

	return tracer
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

	builder := strings.Builder{}
	builder.Write(c.Request.Method())
	builder.Write(spaceByte)
	builder.Write(c.Request.Path())
	builder.Write(spaceByte)
	builder.WriteString(c.Request.Header.GetProtocol())

	upstreamAddr := c.GetString(UPSTREAM_ADDR)
	upstreamRespTime := c.GetString(UPSTREAM_RESPONSE_TIME)

	var body string
	if c.Request.Body() != nil {
		body = b2s(c.Request.Body())
	}

	replacer := strings.NewReplacer(
		UPSTREAM_RESPONSE_TIME, upstreamRespTime,
		UPSTREAM_STATUS, strconv.Itoa(c.GetInt(UPSTREAM_STATUS)),
		REMOTE_ADDR, c.RemoteAddr().String(),
		STATUS, strconv.Itoa(c.Response.StatusCode()),
		REQUEST_BODY, body,
		UPSTREAM_ADDR, upstreamAddr,
		REQUEST_TIME, requestTime,
		REQUEST, builder.String(),
		TIME, startTime.Format("2006-01-02 15:04:05"),
	)

	result := replacer.Replace(t.template)
	t.logChan <- result
}
