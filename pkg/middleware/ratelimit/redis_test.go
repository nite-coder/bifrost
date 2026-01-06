package ratelimit

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	redisMod "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

func startRedis(t *testing.T) (string, func()) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	redisContainer, err := redisMod.Run(ctx,
		"redis:7.4",
		redisMod.WithSnapshotting(10, 1),
		redisMod.WithLogLevel(redisMod.LogLevelVerbose),
	)
	if err != nil {
		t.Fatalf("failed to start container: %s", err)
	}

	endpoint, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %s", err)
	}
	// Strip redis:// prefix if present, as go-redis Options.Addr expects host:port
	endpoint = strings.TrimPrefix(endpoint, "redis://")

	return endpoint, func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}
}

func startRedisCluster(t *testing.T) ([]string, map[string]string, func()) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	// 1. Create a network
	newNetwork, err := network.New(ctx, network.WithCheckDuplicate())
	if err != nil {
		t.Fatal(err)
	}
	networkName := newNetwork.Name

	// 2. Start 3 Redis nodes
	var containers []testcontainers.Container
	var nodeIPs []string
	var hostAddrs []string

	for i := 0; i < 3; i++ {
		req := testcontainers.ContainerRequest{
			Image:        "redis:7.4",
			Cmd:          []string{"redis-server", "--cluster-enabled", "yes", "--cluster-config-file", "nodes.conf", "--cluster-node-timeout", "5000", "--appendonly", "yes", "--requirepass", "bitnami"},
			ExposedPorts: []string{"6379/tcp"},
			Networks:     []string{networkName},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		}

		container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			t.Fatalf("failed to start redis node %d: %v", i, err)
		}
		containers = append(containers, container)

		// Get internal IP
		ip, err := container.ContainerIP(ctx)
		if err != nil {
			t.Fatalf("failed to get container IP: %v", err)
		}
		nodeIPs = append(nodeIPs, ip)

		// Get mapped host port
		endpoint, err := container.PortEndpoint(ctx, "6379/tcp", "")
		if err != nil {
			t.Fatalf("failed to get endpoint: %v", err)
		}
		hostAddrs = append(hostAddrs, endpoint)
	}

	// 3. Create Cluster
	// We run redis-cli --cluster create on the first node
	clusterCmd := []string{"redis-cli", "-a", "bitnami", "--cluster", "create"}
	for _, ip := range nodeIPs {
		clusterCmd = append(clusterCmd, fmt.Sprintf("%s:6379", ip))
	}
	clusterCmd = append(clusterCmd, "--cluster-replicas", "0", "--cluster-yes")

	exitCode, _, err := containers[0].Exec(ctx, clusterCmd)
	if err != nil {
		t.Fatalf("failed to execute cluster create: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("cluster create failed with exit code %d", exitCode)
	}

	// 4. Wait for Cluster state OK
	// Collect Node ID -> HostAddr mapping
	idMap := make(map[string]string)

	assert.Eventually(t, func() bool {
		// Verify with single node connection first
		for _, addr := range hostAddrs {
			rdb := redis.NewClient(&redis.Options{
				Addr:     addr,
				Password: "bitnami",
			})
			info, err := rdb.ClusterInfo(ctx).Result()
			if err != nil {
				_ = rdb.Close()
				return false
			}
			if !strings.Contains(info, "cluster_state:ok") {
				_ = rdb.Close()
				return false
			}

			// Get Node ID
			myID, err := rdb.ClusterMyID(ctx).Result()
			_ = rdb.Close()
			if err != nil {
				return false
			}
			idMap[myID] = addr
		}
		return true
	}, 60*time.Second, 1*time.Second, "Cluster state should be OK")

	destroyFunc := func() {
		for _, c := range containers {
			_ = c.Terminate(ctx)
		}
		_ = newNetwork.Remove(ctx)
	}

	return hostAddrs, idMap, destroyFunc
}

func TestRedis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

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
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	addrs, idMap, destroyRedis := startRedisCluster(t)
	defer destroyRedis()

	ctx := context.Background()
	client := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:        addrs,
		Password:     "bitnami",
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		// Custom ClusterSlots to rewrite internal container IPs to localhost mapped ports
		ClusterSlots: func(ctx context.Context) ([]redis.ClusterSlot, error) {
			// Create a temporary single-node client to fetch slots
			// usage of any node is fine
			rdb := redis.NewClient(&redis.Options{
				Addr:     addrs[0],
				Password: "bitnami",
			})
			defer rdb.Close()

			slots, err := rdb.ClusterSlots(ctx).Result()
			if err != nil {
				return nil, err
			}

			// Rewrite addrs
			for i := range slots {
				for j := range slots[i].Nodes {
					nodeID := slots[i].Nodes[j].ID
					if mappedAddr, ok := idMap[nodeID]; ok {
						slots[i].Nodes[j].Addr = mappedAddr
					}
				}
			}
			return slots, nil
		},
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

		assert.Eventually(t, func() bool {
			return limiter.Allow(ctx, key).Allow
		}, options.WindowSize+1*time.Second, 100*time.Millisecond, "Request after reset should be allowed")
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
