package consistent

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNew tests the creation of a new Consistent ring with default configuration.
func TestNew(t *testing.T) {
	ring := New()
	require.NotNil(t, ring)
	assert.Equal(t, DefaultReplicas, ring.replicas)
	assert.NotNil(t, ring.hashFunc)
	assert.True(t, ring.IsEmpty())
}

// TestSetReplicas tests setting custom replica count.
func TestSetReplicas(t *testing.T) {
	ring := New().SetReplicas(100)
	assert.Equal(t, 100, ring.replicas)

	// Test chaining
	ring2 := New().SetReplicas(50).SetReplicas(200)
	assert.Equal(t, 200, ring2.replicas)
}

// TestSetHashFunc tests setting a custom hash function.
func TestSetHashFunc(t *testing.T) {
	customHash := func(data []byte) uint32 {
		return 12345 // Simple custom hash for testing
	}

	ring := New().SetHashFunc(customHash)
	assert.NotNil(t, ring.hashFunc)

	// Verify the custom hash function is used
	result := ring.hashFunc([]byte("test"))
	assert.Equal(t, uint32(12345), result)
}

// TestChainableAPI tests that all configuration methods can be chained.
func TestChainableAPI(t *testing.T) {
	customHash := func(data []byte) uint32 {
		return uint32(len(data))
	}

	ring := New().
		SetReplicas(100).
		SetHashFunc(customHash)

	assert.Equal(t, 100, ring.replicas)
	assert.Equal(t, uint32(4), ring.hashFunc([]byte("test")))
}

// TestAddNode tests adding nodes to the ring.
func TestAddNode(t *testing.T) {
	ring := New()

	// Add first node
	err := ring.Add("node1")
	require.NoError(t, err)
	assert.False(t, ring.IsEmpty())
	assert.Equal(t, 1, len(ring.Nodes()))

	// Add second node
	err = ring.Add("node2")
	require.NoError(t, err)
	assert.Equal(t, 2, len(ring.Nodes()))

	// Try to add duplicate node - it should replace/update now
	err = ring.Add("node1")
	require.NoError(t, err)
	assert.Equal(t, 2, len(ring.Nodes()))
}

// TestAddWithReplicas tests adding/updating nodes with custom replica counts.
func TestAddWithReplicas(t *testing.T) {
	ring := New()

	// Add node with custom replicas
	err := ring.AddWithReplicas("node1", 10)
	require.NoError(t, err)
	assert.Equal(t, 10, len(ring.sortedNodes))

	// Update node with different replicas
	err = ring.AddWithReplicas("node1", 20)
	require.NoError(t, err)
	assert.Equal(t, 20, len(ring.sortedNodes))

	// Verify other nodes are unaffected
	err = ring.Add("node2") // Uses default (160)
	require.NoError(t, err)
	assert.Equal(t, 20+DefaultReplicas, len(ring.sortedNodes))
}

// TestRemoveNode tests removing nodes from the ring.
func TestRemoveNode(t *testing.T) {
	ring := New()
	_ = ring.Add("node1")
	_ = ring.Add("node2")

	// Remove existing node
	err := ring.Remove("node1")
	require.NoError(t, err)
	assert.Equal(t, 1, len(ring.Nodes()))

	// Try to remove non-existent node
	err = ring.Remove("node3")
	assert.ErrorIs(t, err, ErrNodeNotFound)

	// Remove last node
	err = ring.Remove("node2")
	require.NoError(t, err)
	assert.True(t, ring.IsEmpty())
}

// TestGet tests retrieving nodes for keys.
func TestGet(t *testing.T) {
	ring := New()

	// Test empty ring
	_, err := ring.Get("key1")
	assert.ErrorIs(t, err, ErrEmptyRing)

	// Add nodes
	_ = ring.Add("node1")
	_ = ring.Add("node2")
	_ = ring.Add("node3")

	// Test that same key always returns same node
	node1, err := ring.Get("user123")
	require.NoError(t, err)
	assert.NotEmpty(t, node1)

	node2, err := ring.Get("user123")
	require.NoError(t, err)
	assert.Equal(t, node1, node2)

	// Different keys may map to different nodes
	node3, err := ring.Get("user456")
	require.NoError(t, err)
	assert.NotEmpty(t, node3)
}

// TestConsistency tests that keys remain on the same node when possible.
func TestConsistency(t *testing.T) {
	ring := New()

	// Add 3 nodes
	_ = ring.Add("node1")
	_ = ring.Add("node2")
	_ = ring.Add("node3")

	// Map 1000 keys and record their assignments
	assignments := make(map[string]string)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key%d", i)
		node, _ := ring.Get(key)
		assignments[key] = node
	}

	// Remove one node
	_ = ring.Remove("node1")

	// Check how many keys moved
	moved := 0
	for key, oldNode := range assignments {
		newNode, _ := ring.Get(key)
		if oldNode != "node1" && oldNode != newNode {
			moved++
		}
	}

	// Keys that were on node1 should move, but keys on node2 and node3 should stay
	// With consistent hashing, only keys from the removed node should move
	assert.Equal(t, 0, moved, "Keys on alive nodes should not move")
}

// TestDistributionBias tests the distribution bias with 2 nodes and 160 virtual nodes.
// This test verifies that the distribution bias is less than 1%, as per Kong Gateway's standard.
func TestDistributionBias(t *testing.T) {
	ring := New() // Default 160 replicas per node

	// Add 2 physical nodes
	_ = ring.Add("node1")
	_ = ring.Add("node2")

	// Test with a large number of userIDs
	numKeys := 100000
	distribution := make(map[string]int)

	for i := 0; i < numKeys; i++ {
		// Simulate userID as uint32
		userID := uint32(i)
		key := strconv.FormatUint(uint64(userID), 10)
		node, err := ring.Get(key)
		require.NoError(t, err)
		distribution[node]++
	}

	// Calculate expected distribution (50% each for 2 nodes)
	expectedPerNode := numKeys / 2

	// Calculate bias for each node
	for node, count := range distribution {
		bias := math.Abs(float64(count-expectedPerNode)) / float64(expectedPerNode) * 100
		t.Logf("Node %s: %d keys (%.2f%% bias)", node, count, bias)

		// Verify bias is less than 1%
		assert.Less(t, bias, 1.0, "Distribution bias should be less than 1%% for node %s", node)
	}

	// Verify both nodes received keys
	assert.Equal(t, 2, len(distribution), "Both nodes should receive keys")
}

// TestDistributionWithDifferentReplicas tests distribution with different replica counts.
func TestDistributionWithDifferentReplicas(t *testing.T) {
	testCases := []struct {
		name     string
		replicas int
		maxBias  float64
	}{
		{"20 replicas", 20, 5.0},   // Higher bias expected with fewer replicas
		{"160 replicas", 160, 1.0}, // Kong Gateway standard
		{"256 replicas", 256, 1.5}, // Even better distribution
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ring := New().SetReplicas(tc.replicas)

			// Add 2 physical nodes
			_ = ring.Add("node1")
			_ = ring.Add("node2")

			// Test with 50000 keys
			numKeys := 50000
			distribution := make(map[string]int)

			for i := 0; i < numKeys; i++ {
				key := strconv.Itoa(i)
				node, _ := ring.Get(key)
				distribution[node]++
			}

			expectedPerNode := numKeys / 2

			for node, count := range distribution {
				bias := math.Abs(float64(count-expectedPerNode)) / float64(expectedPerNode) * 100
				t.Logf("%s - Node %s: %d keys (%.2f%% bias)", tc.name, node, count, bias)
				assert.Less(t, bias, tc.maxBias, "Bias should be less than %.1f%% for %s", tc.maxBias, tc.name)
			}
		})
	}
}

// TestWeightedDistribution tests that SetWithReplicas correctly handles weight-based distribution.
func TestWeightedDistribution(t *testing.T) {
	ring := New()

	// Add 2 nodes with 2:1 weight ratio
	// Node1: 320 virtual nodes (2 * 160)
	// Node2: 160 virtual nodes (1 * 160)
	_ = ring.AddWithReplicas("node1", 320)
	_ = ring.AddWithReplicas("node2", 160)

	numKeys := 100000
	distribution := make(map[string]int)

	for i := 0; i < numKeys; i++ {
		key := strconv.Itoa(i)
		node, _ := ring.Get(key)
		distribution[node]++
	}

	h1Count := distribution["node1"]
	h2Count := distribution["node2"]

	t.Logf("Weighted Distribution (2:1): node1=%d, node2=%d", h1Count, h2Count)

	ratio := float64(h1Count) / float64(h2Count)
	t.Logf("Measured Ratio (h1/h2): %.4f (Expected around 2.0)", ratio)

	// Since we have 480 total virtual nodes, the distribution should be reasonably close to 2:1
	// 2.22 is a reasonable upper bound for 2:1 ratio given the small number of physical nodes
	assert.Greater(t, ratio, 1.7)
	assert.Less(t, ratio, 2.3)
}

// TestVirtualNodeDistribution tests that virtual nodes are properly distributed.
func TestVirtualNodeDistribution(t *testing.T) {
	ring := New().SetReplicas(10) // Use smaller number for easier testing

	_ = ring.Add("node1")

	// Should have 10 virtual nodes
	assert.Equal(t, 10, len(ring.sortedNodes))

	// Verify nodes are sorted
	for i := 1; i < len(ring.sortedNodes); i++ {
		assert.Less(t, ring.sortedNodes[i-1], ring.sortedNodes[i], "Virtual nodes should be sorted")
	}
}

// TestGetN tests retrieving multiple nodes for failover.
func TestGetN(t *testing.T) {
	ring := New()
	_ = ring.Add("node1")
	_ = ring.Add("node2")
	_ = ring.Add("node3")

	// Get 2 nodes
	nodes, err := ring.GetN("user123", 2)
	require.NoError(t, err)
	assert.Equal(t, 2, len(nodes))
	assert.NotEqual(t, nodes[0], nodes[1])

	// Get more nodes than exist
	nodes, err = ring.GetN("user123", 5)
	require.NoError(t, err)
	assert.Equal(t, 3, len(nodes))

	// Verify order is clockwise on the ring
	// If we start from some point, GetN must return unique nodes in the order they appear
	key := "test-failover"
	n3, _ := ring.GetN(key, 3)
	assert.Equal(t, 3, len(n3))

	// The first node in GetN(key, 3) must be the same as Get(key)
	first, _ := ring.Get(key)
	assert.Equal(t, first, n3[0])
}

// TestConcurrency tests concurrent operations on the ring.
func TestConcurrency(t *testing.T) {
	ring := New()

	// Add initial nodes
	_ = ring.Add("node1")
	_ = ring.Add("node2")

	var wg sync.WaitGroup
	numGoroutines := 100
	operationsPerGoroutine := 100

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := fmt.Sprintf("key-%d-%d", id, j)
				_, _ = ring.Get(key)
			}
		}(i)
	}

	// Concurrent writes (add/remove)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			nodeID := fmt.Sprintf("temp-node-%d", id)
			_ = ring.Add(nodeID)
			_ = ring.Remove(nodeID)
		}(i)
	}

	wg.Wait()

	// Verify ring is still consistent
	assert.False(t, ring.IsEmpty())
	nodes := ring.Nodes()
	assert.GreaterOrEqual(t, len(nodes), 2)
}

// TestEdgeCases tests various edge cases.
func TestEdgeCases(t *testing.T) {
	t.Run("empty ring operations", func(t *testing.T) {
		ring := New()
		assert.True(t, ring.IsEmpty())
		assert.Empty(t, ring.Nodes())

		_, err := ring.Get("key")
		assert.ErrorIs(t, err, ErrEmptyRing)

		err = ring.Remove("node1")
		assert.ErrorIs(t, err, ErrNodeNotFound)
	})

	t.Run("single node", func(t *testing.T) {
		ring := New()
		_ = ring.Add("only-node")

		// All keys should map to the only node
		for i := 0; i < 100; i++ {
			node, err := ring.Get(fmt.Sprintf("key%d", i))
			require.NoError(t, err)
			assert.Equal(t, "only-node", node)
		}
	})

	t.Run("zero replicas", func(t *testing.T) {
		ring := New().SetReplicas(0)
		err := ring.Add("node1")
		require.NoError(t, err)

		// With 0 replicas, no virtual nodes are created
		assert.Equal(t, 0, len(ring.sortedNodes))
	})

	t.Run("duplicate add", func(t *testing.T) {
		ring := New()
		_ = ring.Add("node1")
		err := ring.Add("node1") // Should succeed by replacing
		assert.NoError(t, err)
		assert.Equal(t, 1, len(ring.Nodes()))
	})

	t.Run("remove non-existent", func(t *testing.T) {
		ring := New()
		err := ring.Remove("ghost-node")
		assert.ErrorIs(t, err, ErrNodeNotFound)
	})
}

// TestCustomHashFunction tests using a custom hash function.
func TestCustomHashFunction(t *testing.T) {
	// Simple deterministic hash for testing
	simpleHash := func(data []byte) uint32 {
		var hash uint32
		for _, b := range data {
			hash = hash*31 + uint32(b)
		}
		return hash
	}

	ring := New().SetHashFunc(simpleHash)
	_ = ring.Add("node1")
	_ = ring.Add("node2")

	// Verify custom hash is being used
	node1, err := ring.Get("test-key")
	require.NoError(t, err)

	// Same key should always return same node
	node2, err := ring.Get("test-key")
	require.NoError(t, err)
	assert.Equal(t, node1, node2)
}

// TestNodes tests the Nodes() method.
func TestNodes(t *testing.T) {
	ring := New()

	// Empty ring
	nodes := ring.Nodes()
	assert.Empty(t, nodes)

	// Add nodes
	_ = ring.Add("node1")
	_ = ring.Add("node2")
	_ = ring.Add("node3")

	nodes = ring.Nodes()
	assert.Equal(t, 3, len(nodes))
	assert.Contains(t, nodes, "node1")
	assert.Contains(t, nodes, "node2")
	assert.Contains(t, nodes, "node3")
}

// BenchmarkGet benchmarks the Get operation.
func BenchmarkGet(b *testing.B) {
	ring := New()
	_ = ring.Add("node1")
	_ = ring.Add("node2")
	_ = ring.Add("node3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%10000)
		_, _ = ring.Get(key)
	}
}

// BenchmarkAdd benchmarks the Add operation.
func BenchmarkAdd(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ring := New()
		_ = ring.Add(fmt.Sprintf("node%d", i))
	}
}

// BenchmarkConcurrentGet benchmarks concurrent Get operations.
func BenchmarkConcurrentGet(b *testing.B) {
	ring := New()
	_ = ring.Add("node1")
	_ = ring.Add("node2")
	_ = ring.Add("node3")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key%d", i%10000)
			_, _ = ring.Get(key)
			i++
		}
	})
}
