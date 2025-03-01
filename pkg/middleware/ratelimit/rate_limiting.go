package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
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
	LocalAsyncRedis StrategyMode = "local_async_redis" // nolint
)

type Options struct {
	Strategy         StrategyMode  `mapstructure:"strategy"`
	Limit            uint64        `mapstructure:"limit"`
	LimitBy          string        `mapstructure:"limit_by"`
	WindowSize       time.Duration `mapstructure:"window_size"`
	HeaderLimit      string        `mapstructure:"header_limit"`
	HeaderRemaining  string        `mapstructure:"header_remaining"`
	HeaderReset      string        `mapstructure:"header_reset"`
	HTTPStatusCode   int           `mapstructure:"http_status_code"`
	HTTPContentType  string        `mapstructure:"http_content_type"`
	HTTPResponseBody string        `mapstructure:"http_response_body"`
	RedisID          string        `mapstructure:"redis_id"`
}

type RateLimitingMiddleware struct {
	options *Options
	limter  Limiter
}

func NewMiddleware(options Options) (*RateLimitingMiddleware, error) {

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
			return nil, fmt.Errorf("redis id '%s' not found for rate_limit middleware", options.RedisID)
		}
		m.limter = NewRedisLimiter(client, options)
	default:
		return nil, fmt.Errorf("strategy '%s' is invalid for rate_limit middleware", options.Strategy)
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

			if len(m.options.HeaderLimit) > 0 {
				c.Response.Header.Set(m.options.HeaderLimit, strconv.FormatUint(result.Limit, 10))
			}

			if len(m.options.HeaderRemaining) > 0 {
				c.Response.Header.Set(m.options.HeaderRemaining, strconv.FormatUint(result.Remaining, 10))
			}

			if len(m.options.HeaderReset) > 0 {
				c.Response.Header.Set(m.options.HeaderReset, strconv.FormatInt(result.ResetTime.Unix(), 10))
			}

			return
		} else {
			if len(m.options.HeaderLimit) > 0 {
				c.Response.Header.Set(m.options.HeaderLimit, strconv.FormatUint(result.Limit, 10))
			}

			if len(m.options.HeaderRemaining) > 0 {
				c.Response.Header.Set(m.options.HeaderRemaining, strconv.FormatUint(result.Remaining, 10))
			}

			if len(m.options.HeaderReset) > 0 {
				c.Response.Header.Set(m.options.HeaderReset, strconv.FormatInt(result.ResetTime.Unix(), 10))
			}

			c.SetStatusCode(m.options.HTTPStatusCode)

			c.Response.Header.Set("Content-Type", "application/json; charset=utf8")
			if len(m.options.HTTPContentType) > 0 {
				c.Response.Header.Set("Content-Type", m.options.HTTPContentType)
			}

			if len(m.options.HTTPResponseBody) > 0 {
				c.Response.SetBody([]byte(m.options.HTTPResponseBody))
			}
			c.Abort()
			return
		}
	}

	c.Next(ctx)
}

func init() {
	_ = middleware.RegisterMiddleware("rate_limit", func(params any) (app.HandlerFunc, error) {
		if params == nil {
			return nil, errors.New("rate_limit middleware params is empty or invalid")
		}

		option := Options{}

		decoder, _ := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
			WeaklyTypedInput: true,
			Result:           &option,
		})

		err := decoder.Decode(params)
		if err != nil {
			return nil, fmt.Errorf("rate_limit middleware params is invalid: %w", err)
		}

		if len(option.LimitBy) == 0 {
			return nil, errors.New("limit_by can't be empty for rate_limit middleware")
		}

		switch option.Strategy {
		case Local, Redis:
		case "":
			option.Strategy = Local
		default:
			return nil, fmt.Errorf("strategy '%s' is invalid", option.Strategy)
		}

		if option.WindowSize == 0 {
			return nil, errors.New("window_size must be greater than 0 for rate_limit middleware")
		}

		m, err := NewMiddleware(option)
		if err != nil {
			return nil, err
		}
		return m.ServeHTTP, nil
	})
}
