package chash

import (
	"context"
	"errors"
	"hash/fnv"
	"slices"
	"sort"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/proxy"
	"github.com/nite-coder/bifrost/pkg/variable"
)

const (
	defaultReplicas = 256
)

func Init() error {
	return balancer.Register([]string{"hashing", "chash"}, func(proxies []proxy.Proxy, params any) (balancer.Balancer, error) {
		if params == nil {
			return nil, errors.New("params cannot be empty")
		}
		var hashon string
		if val, ok := params.(map[string]any); ok {
			hashon, ok = val["hash_on"].(string)
			if !ok {
				return nil, errors.New("hash_on is required and must be a string")
			}
		}

		b := NewBalancer(proxies, hashon)
		return b, nil
	})
}

// HashingBalancer implements a consistent hashing balancer with virtual nodes.
type HashingBalancer struct {
	hashon      string
	proxies     []proxy.Proxy
	sortedNodes []uint32
	ring        map[uint32]proxy.Proxy
}

// NewBalancer creates a new HashingBalancer instance.
// It initializes the hash ring with virtual nodes for each proxy.
func NewBalancer(proxies []proxy.Proxy, hashon string) *HashingBalancer {
	b := &HashingBalancer{
		proxies: proxies,
		hashon:  hashon,
		ring:    make(map[uint32]proxy.Proxy),
	}

	for _, p := range proxies {
		for i := range defaultReplicas {
			// Virtual node IDs are hashed to distribute them along the ring
			hash := fnv32a(p.ID() + "-" + strconv.Itoa(i))
			b.ring[hash] = p
			b.sortedNodes = append(b.sortedNodes, hash)
		}
	}
	slices.Sort(b.sortedNodes)

	return b
}

// Proxies returns the list of proxies managed by the balancer.
func (b *HashingBalancer) Proxies() []proxy.Proxy {
	return b.proxies
}

// Select picks a proxy from the hash ring based on a hashed value from the request.
// If the selected proxy is unavailable, it moves clockwise along the ring to find the next available one.
func (b *HashingBalancer) Select(ctx context.Context, c *app.RequestContext) (proxy.Proxy, error) {
	if len(b.proxies) == 0 {
		return nil, balancer.ErrNotAvailable
	}

	val := variable.GetString(b.hashon, c)
	// Use stateless fnv32a to avoid per-request allocations
	hashValue := fnv32a(val)

	// Find the first virtual node >= hashValue
	idx := sort.Search(len(b.sortedNodes), func(i int) bool {
		return b.sortedNodes[i] >= hashValue
	})

	failedCount := 0
	for failedCount < len(b.sortedNodes) {
		// Wrap around the ring
		if idx >= len(b.sortedNodes) {
			idx = 0
		}

		targetProxy := b.ring[b.sortedNodes[idx]]
		if targetProxy.IsAvailable() {
			return targetProxy, nil
		}

		// Try next node on the ring
		idx++
		failedCount++
	}

	return nil, balancer.ErrNotAvailable
}

func fnv32a(s string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return h.Sum32()
}
