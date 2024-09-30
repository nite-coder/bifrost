package ratelimiting

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
)

type Limter interface {
	Allow(key string) bool
}

type Options struct {
	Strategy   string
	Limit      int64
	LimitBy    string
	WindowSize time.Duration
}

type RateLimitingMiddleware struct {
	options *Options
	limter  Limter
}

func NewMiddleware(options Options) (*RateLimitingMiddleware, error) {

	strategy := strings.ToLower(strings.TrimSpace(options.Strategy))

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

	return m, nil
}

func (m *RateLimitingMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	isAllow := c.GetBool(config.ALLOW)
	if isAllow {
		c.Next(ctx)
		return
	}

	key, found := c.Get(m.options.LimitBy)
	if found {
		if m.limter.Allow(key.(string)) {
			c.Next(ctx)
			return
		} else {
			c.SetStatusCode(429)
			return
		}
	}

	c.Next(ctx)
}
