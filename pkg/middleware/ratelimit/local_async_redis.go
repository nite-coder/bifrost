package ratelimit

import (
	"context"
	"math"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
)

type LocalAsyncRedisLimiter struct {
	options       *Options
	currentTokens int64
	client        redis.UniversalClient
}

func NewLocalAsyncRedisLimiter(client redis.UniversalClient, options Options) *LocalAsyncRedisLimiter {
	limiter := LocalAsyncRedisLimiter{
		options: &options,
		client:  client,
	}

	if options.Limit > math.MaxInt64 {
		limiter.currentTokens = math.MaxInt64
	} else {
		limiter.currentTokens = int64(options.Limit)
	}

	return &limiter
}

const rateLimitLuaScript = `
local key = KEYS[1]
local max_tokens = tonumber(ARGV[1])
local ttl = tonumber(ARGV[2])

local current = redis.call("INCRBY", key, max_tokens)

if current_count == max_tokens then
    redis.call('EXPIRE', key, ttl)
end

return current_count
`

func (r *LocalAsyncRedisLimiter) TryAcquire(token int64) bool {
	if token <= 0 {
		return false
	}

	go r.asyncSyncWithRedis()
	return atomic.AddInt64(&r.currentTokens, -token) >= 0
}

func (r *LocalAsyncRedisLimiter) asyncSyncWithRedis() {
	ctx, cancel := context.WithTimeout(context.Background(), r.options.WindowSize)
	defer cancel()

	result, err := r.client.Eval(
		ctx,
		rateLimitLuaScript,
		[]string{r.options.LimitBy},
		r.options.Limit,
		int(r.options.WindowSize.Seconds()),
	).Result()

	if err != nil {
		// TODO: log error
		return
	}

	count, ok := result.(int64)
	if ok {
		maxTokens := int64(0)
		if r.options.Limit > math.MaxInt64 {
			maxTokens = math.MaxInt64
		} else {
			maxTokens = int64(r.options.Limit) // nolint
		}

		if count <= maxTokens {
			atomic.StoreInt64(&r.currentTokens, maxTokens-count)
		} else {
			atomic.StoreInt64(&r.currentTokens, 0)
		}
	}
}
