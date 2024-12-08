package ratelimiting

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config/redisclient"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/nite-coder/bifrost/pkg/variable"
	"github.com/nite-coder/blackbear/pkg/cast"
)

type Limiter interface {
	Allow(ctx context.Context, key string) *AllowResult
}

type AllowResult struct {
	Allow     bool
	Limit     uint64
	Remaining uint64
	ResetTime time.Time
}

type StrategyMode string

const (
	Local           StrategyMode = "local"
	Redis           StrategyMode = "redis"
	LocalAsyncRedis StrategyMode = "local-async-redis" // nolint
)

type Options struct {
	Strategy         StrategyMode
	Limit            uint64
	LimitBy          string
	WindowSize       time.Duration
	HeaderLimit      string
	HeaderRemaining  string
	HeaderReset      string
	HTTPStatusCode   int
	HTTPContentType  string
	HTTPResponseBody string
	RedisID          string
}

type RateLimitingMiddleware struct {
	options *Options
	limter  Limiter
}

func NewMiddleware(options Options) (*RateLimitingMiddleware, error) {

	if options.HeaderLimit == "" {
		options.HeaderLimit = "X-RateLimit-Limit"
	}

	if options.HeaderRemaining == "" {
		options.HeaderRemaining = "X-RateLimit-Remaining"
	}

	if options.HeaderReset == "" {
		options.HeaderReset = "X-RateLimit-Reset"
	}

	if options.HTTPStatusCode == 0 {
		options.HTTPStatusCode = 429
	}

	if options.HTTPContentType == "" {
		options.HeaderReset = "application/json; charset=utf8"
	}

	if options.WindowSize == 0 {
		return nil, errors.New("window_size must be greater than 0")
	}

	m := &RateLimitingMiddleware{
		options: &options,
	}

	switch options.Strategy {
	case Local:
		m.limter = NewLocalLimiter(options)
	case Redis:
		client, found := redisclient.Get(options.RedisID)
		if !found {
			return nil, fmt.Errorf("redis id '%s' not found in ratelimiting middleware", options.RedisID)
		}
		m.limter = NewRedisLimiter(client, options)
	default:
		return nil, fmt.Errorf("strategy '%s' is invalid", options.Strategy)
	}

	return m, nil
}

func (m *RateLimitingMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	isAllow := c.GetBool(variable.Allow)
	if isAllow {
		c.Next(ctx)
		return
	}

	key := variable.GetString(m.options.LimitBy, c)

	if len(key) > 0 {
		val, _ := cast.ToString(key)

		builder := strings.Builder{}
		builder.WriteString("rate_limit:")
		builder.WriteString(m.options.LimitBy)
		builder.WriteString(":")
		builder.WriteString(val)
		key := builder.String()

		result := m.limter.Allow(ctx, key)

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

			c.SetStatusCode(m.options.HTTPStatusCode)
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

func init() {
	_ = middleware.RegisterMiddleware("rate-limiting", func(param map[string]any) (app.HandlerFunc, error) {
		option := Options{}

		// strategy
		strategyVal, found := param["strategy"]
		if !found {
			return nil, errors.New("strategy is not found in rate-limiting middleware")
		}
		strategy, err := cast.ToString(strategyVal)
		if err != nil {
			return nil, errors.New("strategy is invalid in rate-limiting middleware")
		}
		option.Strategy = StrategyMode(strategy)

		// limit
		limitVal, found := param["limit"]
		if !found {
			return nil, errors.New("limit is not found in rate-limiting middleware1")
		}
		limit, err := cast.ToUint64(limitVal)
		if err != nil {
			return nil, errors.New("limit is invalid in rate-limiting middleware")
		}
		option.Limit = limit

		// limit_by
		limitByVal, found := param["limit_by"]
		if !found {
			return nil, errors.New("limit_by is not found in rate-limiting middleware")
		}
		limitBy, err := cast.ToString(limitByVal)
		if err != nil {
			return nil, errors.New("limit_by is invalid in rate-limiting middleware")
		}
		option.LimitBy = limitBy

		// window_size
		windowSizeVal, found := param["window_size"]
		if !found {
			return nil, errors.New("window_size is not found in rate-limiting middleware")
		}
		s, _ := cast.ToString(windowSizeVal)
		windowSize, err := time.ParseDuration(s)
		if err != nil {
			return nil, errors.New("window_size is invalid in rate-limiting middleware")
		}
		option.WindowSize = windowSize

		// http status
		statusVal, found := param["http_status"]
		if found {
			status, err := cast.ToInt(statusVal)
			if err != nil {
				return nil, errors.New("http_status is invalid in rate-limiting middleware")
			}
			option.HTTPStatusCode = status
		}

		// http content type
		contentTypeVal, found := param["http_content_type"]
		if found {
			contentType, err := cast.ToString(contentTypeVal)
			if err != nil {
				return nil, errors.New("http_content_type is invalid in rate-limiting middleware")
			}
			option.HTTPContentType = contentType
		}

		// http body
		bodyVal, found := param["http_response_body"]
		if found {
			body, err := cast.ToString(bodyVal)
			if err != nil {
				return nil, errors.New("http_response_body is invalid in rate-limiting middleware")
			}
			option.HTTPResponseBody = body
		}

		// redis id
		if option.Strategy == Redis {
			redisIDVal, found := param["redis_id"]
			if found {
				redisID, err := cast.ToString(redisIDVal)
				if err != nil {
					return nil, errors.New("redis_id is invalid in rate-limiting middleware")
				}
				option.RedisID = redisID
			} else {
				return nil, errors.New("redis_id is not found in rate-limiting middleware")
			}
		}

		m, err := NewMiddleware(option)
		if err != nil {
			return nil, err
		}
		return m.ServeHTTP, nil
	})

}
