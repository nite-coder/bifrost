package consistent

import (
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/stathat/consistent"
)

// TestStathatDistribution tests the distribution of stathat/consistent package
// with 2 nodes and 160 virtual nodes to compare with our implementation.
func TestStathatDistribution(t *testing.T) {
	c := consistent.New()
	c.NumberOfReplicas = 160

	// Add 2 physical nodes
	c.Add("node1")
	c.Add("node2")

	// Test with a large number of keys
	numKeys := 100000
	distribution := make(map[string]int)

	for i := 0; i < numKeys; i++ {
		userID := uint32(i)
		key := strconv.FormatUint(uint64(userID), 10)
		node, err := c.Get(key)
		if err != nil {
			t.Fatalf("Failed to get node: %v", err)
		}
		distribution[node]++
	}

	// Calculate expected distribution (50% each for 2 nodes)
	expectedPerNode := numKeys / 2

	// Calculate bias for each node
	t.Logf("\nstathat/consistent distribution (160 replicas, 2 nodes, %d keys):", numKeys)
	for node, count := range distribution {
		bias := math.Abs(float64(count-expectedPerNode)) / float64(expectedPerNode) * 100
		t.Logf("  Node %s: %d keys (%.2f%% bias)", node, count, bias)
	}
}

// TestOurDistribution tests our implementation's distribution.
func TestOurDistribution(t *testing.T) {
	ring := New() // Default 160 replicas

	// Add 2 physical nodes
	_ = ring.Add("node1")
	_ = ring.Add("node2")

	// Test with a large number of keys
	numKeys := 100000
	distribution := make(map[string]int)

	for i := 0; i < numKeys; i++ {
		userID := uint32(i)
		key := strconv.FormatUint(uint64(userID), 10)
		node, err := ring.Get(key)
		if err != nil {
			t.Fatalf("Failed to get node: %v", err)
		}
		distribution[node]++
	}

	// Calculate expected distribution (50% each for 2 nodes)
	expectedPerNode := numKeys / 2

	// Calculate bias for each node
	t.Logf("\nOur implementation distribution (160 replicas, 2 nodes, %d keys):", numKeys)
	for node, count := range distribution {
		bias := math.Abs(float64(count-expectedPerNode)) / float64(expectedPerNode) * 100
		t.Logf("  Node %s: %d keys (%.2f%% bias)", node, count, bias)
	}
}

// BenchmarkStathatGet benchmarks stathat/consistent Get operation.
func BenchmarkStathatGet(b *testing.B) {
	c := consistent.New()
	c.NumberOfReplicas = 160
	c.Add("node1")
	c.Add("node2")
	c.Add("node3")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%10000)
		_, _ = c.Get(key)
	}
}

// BenchmarkOurGet benchmarks our implementation's Get operation.
func BenchmarkOurGet(b *testing.B) {
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

// BenchmarkStathatAdd benchmarks stathat/consistent Add operation.
func BenchmarkStathatAdd(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := consistent.New()
		c.NumberOfReplicas = 160
		c.Add(fmt.Sprintf("node%d", i))
	}
}

// BenchmarkOurAdd benchmarks our implementation's Add operation.
func BenchmarkOurAdd(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ring := New()
		_ = ring.Add(fmt.Sprintf("node%d", i))
	}
}

// BenchmarkStathatConcurrentGet benchmarks concurrent Get operations for stathat/consistent.
func BenchmarkStathatConcurrentGet(b *testing.B) {
	c := consistent.New()
	c.NumberOfReplicas = 160
	c.Add("node1")
	c.Add("node2")
	c.Add("node3")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			key := fmt.Sprintf("key%d", i%10000)
			_, _ = c.Get(key)
			i++
		}
	})
}

// BenchmarkOurConcurrentGet benchmarks concurrent Get operations for our implementation.
func BenchmarkOurConcurrentGet(b *testing.B) {
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
