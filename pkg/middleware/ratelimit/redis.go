package ratelimit

import (
	"context"
	"time"

	"github.com/nite-coder/bifrost/pkg/log"
	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/blackbear/pkg/cast"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type RedisLimiter struct {
	options *Options
	client  redis.UniversalClient
}

func NewRedisLimiter(client redis.UniversalClient, options Options) *RedisLimiter {
	return &RedisLimiter{
		client:  client,
		options: &options,
	}
}

const (
	luaScript = `
    local key = KEYS[1]
    local tokens = tonumber(ARGV[1])
    local limit = tonumber(ARGV[2])
    local window = tonumber(ARGV[3])
    local now = tonumber(ARGV[4])

    local current = redis.call("INCRBY", key, tokens)
    local ttl = redis.call("TTL", key)

    if ttl == -1 then
        redis.call("EXPIRE", key, window)
        ttl = window
    end

    local resetTime = now + ttl * 1000  -- Convert to milliseconds
    local remaining = limit - current
    if remaining < 0 then
        remaining = 0
    end

    return {current, limit, remaining, resetTime}
    `
)

func (l *RedisLimiter) Allow(ctx context.Context, key string) *AllowResult {
	logger := log.FromContext(ctx)

	tokens := 1

	now := timecache.Now()
	t := now.UnixNano() / int64(time.Millisecond)

	tracer := otel.Tracer("bifrost")
	var span trace.Span
	if tracer != nil {
		spanOptions := []trace.SpanStartOption{
			trace.WithSpanKind(trace.SpanKindClient),
		}

		ctx, span = tracer.Start(ctx, "ratelimiting_redis", spanOptions...)
	}

	result, err := l.client.Eval(ctx, luaScript, []string{key}, tokens, l.options.Limit, int(l.options.WindowSize.Seconds()), t).Result()
	if err != nil {
		logger.Error("ratelimiting: redis eval error", "error", err)
		span.SetStatus(otelcodes.Error, err.Error())
		span.End()
		return &AllowResult{
			Allow:     true,
			Limit:     l.options.Limit,
			Remaining: l.options.Limit,
			ResetTime: now.Add(l.options.WindowSize),
		}
	}

	span.SetStatus(otelcodes.Ok, "")
	span.End()

	resultArray := result.([]any)

	current, _ := cast.ToUint64(resultArray[0])
	remaining, _ := cast.ToUint64(resultArray[2])
	resetTime := time.UnixMilli(resultArray[3].(int64) * int64(time.Millisecond))

	return &AllowResult{
		Allow:     current <= l.options.Limit,
		Limit:     l.options.Limit,
		Remaining: remaining,
		ResetTime: resetTime,
	}
}
