package redis

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/redis/go-redis/v9"
)

var (
	clients = make(map[string]redis.UniversalClient)
	mu      = sync.RWMutex{}
)

func Get(id string) (redis.UniversalClient, bool) {
	mu.RLock()
	defer mu.RUnlock()

	client, ok := clients[id]
	if ok {
		return client, true
	}

	return nil, false
}

func Initialize(ctx context.Context, options []config.RedisOptions) error {
	if len(options) == 0 {
		return nil
	}

	mu.Lock()
	defer mu.Unlock()

	for _, option := range options {
		if len(option.Addrs) == 0 {
			return errors.New("redis options addrs can't be empty")
		}

		if option.ID == "" {
			return errors.New("redis options id can't be empty")
		}

		if len(option.Addrs) == 1 {
			// signle redis
			addr := strings.TrimSpace(option.Addrs[0])
			if addr == "" {
				return errors.New("redis addrs can't be empty")
			}

			client := redis.NewClient(&redis.Options{
				Addr:     addr,
				Username: option.Username,
				Password: option.Password,
				DB:       option.DB,
			})

			if !option.SkipPing {
				_, err := client.Ping(ctx).Result()
				if err != nil {
					return err
				}
			}

			clients[option.ID] = client
		} else {
			// multi redis
			addrs := []string{}

			for _, addr := range option.Addrs {
				addr := strings.TrimSpace(addr)
				if addr == "" {
					return errors.New("redis addrs can't be empty")
				}
				addrs = append(addrs, addr)
			}

			client := redis.NewClusterClient(&redis.ClusterOptions{
				Addrs:    addrs,
				Username: option.Username,
				Password: option.Password,
			})

			if !option.SkipPing {
				_, err := client.Ping(ctx).Result()
				if err != nil {
					return err
				}
			}

			clients[option.ID] = client
		}

	}

	return nil
}
