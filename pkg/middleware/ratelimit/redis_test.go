package ratelimit

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func startRedis(t *testing.T) (string, func()) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Failed to start Dockertest: %+v", err)
	}

	resource, err := pool.Run("redis", "7-alpine", nil)
	if err != nil {
		t.Fatalf("Failed to start redis: %+v", err)
	}

	// determine the port the container is listening on
	addr := net.JoinHostPort("localhost", resource.GetPort("6379/tcp"))

	// wait for the container to be ready
	err = pool.Retry(func() error {
		var e error
		ctx := context.Background()
		client := redis.NewClient(&redis.Options{Addr: addr})
		defer client.Close()

		_, e = client.Ping(ctx).Result()
		return e
	})

	if err != nil {
		t.Fatalf("Failed to ping Redis: %+v", err)
	}

	destroyFunc := func() {
		_ = pool.Purge(resource)
	}

	return addr, destroyFunc
}

func startRedisCluster(t *testing.T) func() {
	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Failed to start Dockertest: %+v", err)
	}

	networks, err := pool.NetworksByName("redis-cluster-network")
	if err != nil {
		t.Fatalf("Could not find docker network: %s", err)
	}

	var network *dockertest.Network
	if len(networks) > 0 {
		network = &networks[0]
	} else {
		network, err = pool.CreateNetwork("redis-cluster-network")
		if err != nil {
			t.Fatalf("Could not create docker network: %s", err)
		}
	}

	node0, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:       "redis-node-0",
		Repository: "bitnami/redis-cluster",
		Tag:        "latest",
		Env: []string{
			"REDIS_PASSWORD=bitnami",
			"REDISCLI_AUTH=bitnami",
			"REDIS_NODES=redis-node-0 redis-node-1 redis-node-2",
			"REDIS_CLUSTER_CREATOR=yes",
			"REDIS_CLUSTER_REPLICAS=0",
		},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"6379/tcp": {{HostPort: "7000"}},
		},
		Networks: []*dockertest.Network{network},
	})
	if err != nil {
		t.Fatalf("Failed to start redis: %+v", err)
	}

	node1, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:       "redis-node-1",
		Repository: "bitnami/redis-cluster",
		Tag:        "latest",
		Env: []string{
			"REDIS_PASSWORD=bitnami",
			"REDIS_NODES=redis-node-0 redis-node-1 redis-node-2",
			"REDIS_CLUSTER_CREATOR=no",
			"REDIS_CLUSTER_REPLICAS=0",
		},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"6379/tcp": {{HostPort: "7001"}},
		},
		Networks: []*dockertest.Network{network},
	})
	if err != nil {
		t.Fatalf("Failed to start redis: %+v", err)
	}

	node2, err := pool.RunWithOptions(&dockertest.RunOptions{
		Name:       "redis-node-2",
		Repository: "bitnami/redis-cluster",
		Tag:        "latest",
		Env: []string{
			"REDIS_PASSWORD=bitnami",
			"REDIS_NODES=redis-node-0 redis-node-1 redis-node-2",
			"REDIS_CLUSTER_CREATOR=no",
			"REDIS_CLUSTER_REPLICAS=0",
		},
		PortBindings: map[docker.Port][]docker.PortBinding{
			"6379/tcp": {{HostPort: "7002"}},
		},
		Networks: []*dockertest.Network{network},
	})
	if err != nil {
		t.Fatalf("Failed to start redis: %+v", err)
	}

	destroyFunc := func() {
		_ = pool.Purge(node0)
		_ = pool.Purge(node1)
		_ = pool.Purge(node2)
		network.Close()
	}

	// wait for the container to be ready
	err = pool.Retry(func() error {
		ctx := context.Background()
		client := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    []string{"localhost:7000", "localhost:7001", "localhost:7002"},
			Password: "bitnami",
		})
		defer client.Close()

		// ping can't be used to check if the redis cluster is ready
		info, err := client.ClusterInfo(ctx).Result()
		if err != nil {
			return err
		}

		if strings.Contains(info, "cluster_state:ok") {
			return nil
		} else {
			return fmt.Errorf("cluster state is not ok")
		}
	})
	if err != nil {
		t.Fatalf("Failed to connect redis cluster: %+v", err)
	}

	time.Sleep(2 * time.Second)

	return destroyFunc
}

func TestRedis(t *testing.T) {
	addr, destroyRedis := startRedis(t)
	defer destroyRedis()

	t.Log("addr:", addr)
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{Addr: addr})
	_, e := client.Ping(ctx).Result()
	assert.NoError(t, e)

	options := Options{
		Limit:      5,
		WindowSize: time.Second,
	}

	limiter := NewRedisLimiter(client, options)
	testLimiter(t, limiter, options)
}

func TestRedisCluster(t *testing.T) {
	destroyRedis := startRedisCluster(t)
	defer destroyRedis()

	ctx := context.Background()
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        []string{"localhost:7000", "localhost:7001", "localhost:7002"},
		Password:     "bitnami",
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	_, e := client.Ping(ctx).Result()
	assert.NoError(t, e)

	options := Options{
		Limit:      5,
		WindowSize: time.Second,
	}

	limiter := NewRedisLimiter(client, options)
	testLimiter(t, limiter, options)
}

func testLimiter(t *testing.T, limiter Limiter, options Options) {

	t.Run("Basic functionality", func(t *testing.T) {
		key := "test_key"
		ctx := context.Background()

		for i := 1; i < 6; i++ {
			now := time.Now()
			result := limiter.Allow(ctx, key)
			if !result.Allow {
				t.Errorf("Request %d should be allowed", i+1)
			}
			assert.Equal(t, options.Limit, result.Limit)
			assert.Equal(t, uint64(5-i), result.Remaining) // nolint
			assert.LessOrEqual(t, result.ResetTime.Sub(now).Seconds(), float64(1.1))
			time.Sleep(100 * time.Millisecond)
		}
		result := limiter.Allow(ctx, key)
		if result.Allow {
			t.Error("6th request should be denied")
		}
	})

	t.Run("Different keys", func(t *testing.T) {
		ctx := context.Background()

		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("key_%d", i)
			result := limiter.Allow(ctx, key)
			if !result.Allow {
				t.Errorf("Request for key %s should be allowed", key)
			}
		}
	})

	t.Run("Window reset", func(t *testing.T) {
		ctx := context.Background()

		key := "reset_key"
		for i := 0; i < 5; i++ {
			result := limiter.Allow(ctx, key)
			if !result.Allow {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}

		result := limiter.Allow(ctx, key)
		if result.Allow {
			t.Error("6th request should be denied")
		}

		time.Sleep(options.WindowSize)

		result = limiter.Allow(ctx, key)
		if !result.Allow {
			t.Error("Request after reset should be allowed")
		}
	})

	t.Run("Concurrent requests", func(t *testing.T) {
		ctx := context.Background()

		key := "concurrent_key"
		concurrentRequests := 100
		allowedCount := atomic.Uint64{}
		var wg sync.WaitGroup

		wg.Add(concurrentRequests)
		for i := 0; i < concurrentRequests; i++ {
			go func() {
				defer wg.Done()

				result := limiter.Allow(ctx, key)
				if result.Allow {
					allowedCount.Add(1)
				}
			}()
		}
		wg.Wait()

		if allowedCount.Load() != options.Limit {
			t.Errorf("Expected %d requests to be allowed, but got %d", options.Limit, allowedCount.Load())
		}
	})
}
