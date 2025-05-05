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
	script  *redis.Script
}

func NewRedisLimiter(client redis.UniversalClient, options Options) *RedisLimiter {
	return &RedisLimiter{
		client:  client,
		options: &options,
		script:  redis.NewScript(luaScript),
	}
}

const (
	luaScript = `
    local key = KEYS[1]
	local tokens, limit, window = tonumber(ARGV[1]), tonumber(ARGV[2]), tonumber(ARGV[3])
	local now = redis.call("TIME")[1] * 1000

    local current = redis.call("INCRBY", key, tokens)
    local pttl = redis.call("PTTL", key)

	if pttl < 0 then
		redis.call("PEXPIRE", key, window)
		pttl = window
	end

    local remaining = limit - current
    if remaining < 0 then remaining = 0 end

    return {current, limit, remaining, now + pttl}
    `
)

func (l *RedisLimiter) Allow(ctx context.Context, key string) *AllowResult {
	logger := log.FromContext(ctx)

	var err error

	tracer := otel.Tracer("bifrost")
	if tracer != nil {
		spanOptions := []trace.SpanStartOption{
			trace.WithSpanKind(trace.SpanKindClient),
		}

		_, span := tracer.Start(ctx, "ratelimit_redis", spanOptions...)

		defer func() {
			if err != nil {
				span.SetStatus(otelcodes.Error, "")
			}

			span.End()
		}()
	}

	keys := []string{key}
	args := []any{1, l.options.Limit, int(l.options.WindowSize.Milliseconds())}

	result, err := l.script.Run(
		ctx,
		l.client,
		keys,
		args,
	).Result()
	if err != nil {
		// downgrade
		logger.Warn("ratelimit: redis eval error", "error", err)
		now := timecache.Now()
		return &AllowResult{
			Allow:     true,
			Limit:     l.options.Limit,
			Remaining: l.options.Limit,
			ResetTime: now.Add(l.options.WindowSize),
		}
	}

	resultArray := result.([]any)

	current, _ := cast.ToUint64(resultArray[0])
	remaining, _ := cast.ToUint64(resultArray[2])
	resetTime := time.UnixMilli(resultArray[3].(int64))

	allowResult := GetAllowResult()
	allowResult.Allow = current <= l.options.Limit
	allowResult.Limit = l.options.Limit
	allowResult.Remaining = remaining
	allowResult.ResetTime = resetTime
	return allowResult
}
