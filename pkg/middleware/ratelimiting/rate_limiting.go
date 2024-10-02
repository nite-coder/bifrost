package ratelimiting

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/nite-coder/blackbear/pkg/cast"
)

type Limter interface {
	Allow(key string) AllowResult
}

type AllowResult struct {
	Allow     bool
	Limit     uint64
	Remaining uint64
	ResetTime time.Time
}

type Options struct {
	Strategy         string
	Limit            uint64
	LimitBy          string
	WindowSize       time.Duration
	HeaderLimit      string
	HeaderRemaining  string
	HeaderReset      string
	HTTPStatus       int
	HTTPContentType  string
	HTTPResponseBody string
}

type RateLimitingMiddleware struct {
	options *Options
	limter  Limter
}

func NewMiddleware(options Options) (*RateLimitingMiddleware, error) {
	strategy := strings.ToLower(strings.TrimSpace(options.Strategy))

	if options.HeaderLimit == "" {
		options.HeaderLimit = "X-RateLimit-Limit"
	}

	if options.HeaderRemaining == "" {
		options.HeaderRemaining = "X-RateLimit-Remaining"
	}

	if options.HeaderReset == "" {
		options.HeaderReset = "X-RateLimit-Reset"
	}

	if options.HTTPStatus == 0 {
		options.HTTPStatus = 429
	}

	if options.HTTPContentType == "" {
		options.HeaderReset = "application/json; charset=utf8"
	}

	m := &RateLimitingMiddleware{
		options: &options,
	}

	switch strategy {
	case "local":
		m.limter = NewLocalLimiter(options)
	case "redis":
	default:
		return nil, fmt.Errorf("strategy '%s' is invalid", strategy)
	}

	if options.WindowSize == 0 {
		return nil, fmt.Errorf("window_size must be greater than 0")
	}

	return m, nil
}

func (m *RateLimitingMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	isAllow := c.GetBool(config.ALLOW)
	if isAllow {
		c.Next(ctx)
		return
	}

	key, found := variable.Get(m.options.LimitBy, c)
	keyStr, _ := cast.ToString(key)

	builder := strings.Builder{}
	builder.WriteString(m.options.LimitBy)
	builder.WriteString(":")
	builder.WriteString(keyStr)
	namespace := builder.String()

	if found {
		result := m.limter.Allow(namespace)

		if result.Allow {
			c.Next(ctx)
			c.Response.Header.Set(m.options.HeaderLimit, strconv.FormatUint(result.Limit, 10))
			c.Response.Header.Set(m.options.HeaderRemaining, strconv.FormatUint(result.Remaining, 10))
			c.Response.Header.Set(m.options.HeaderReset, strconv.FormatInt(result.ResetTime.Unix(), 10))
			return
		} else {
			c.Response.Header.Set(m.options.HeaderLimit, strconv.FormatUint(result.Limit, 10))
			c.Response.Header.Set(m.options.HeaderRemaining, strconv.FormatUint(result.Remaining, 10))
			c.Response.Header.Set(m.options.HeaderReset, strconv.FormatInt(result.ResetTime.Unix(), 10))

			c.SetStatusCode(m.options.HTTPStatus)
			c.Response.Header.Set("Content-Type", "application/json; charset=utf8")
			if m.options.HTTPResponseBody != "" {
				c.Response.SetBody([]byte(m.options.HTTPResponseBody))
			}
			c.Abort()
			return
		}
	}

	c.Next(ctx)
}
