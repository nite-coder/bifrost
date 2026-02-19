package consistent

import (
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/stathat/consistent"
)

// TestDistributionWith5Nodes tests distribution with 5 physical nodes.
func TestDistributionWith5Nodes(t *testing.T) {
	testCases := []struct {
		name     string
		replicas int
	}{
		{"20 replicas", 20},
		{"160 replicas", 160},
		{"256 replicas", 256},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test our implementation
			ring := New().SetReplicas(tc.replicas)
			for i := 1; i <= 5; i++ {
				_ = ring.Add(fmt.Sprintf("node%d", i))
			}

			numKeys := 100000
			distribution := make(map[string]int)

			for i := 0; i < numKeys; i++ {
				key := strconv.Itoa(i)
				node, _ := ring.Get(key)
				distribution[node]++
			}

			expectedPerNode := numKeys / 5
			t.Logf("\n%s - Our implementation (5 nodes, %d keys):", tc.name, numKeys)

			var maxBias float64
			for node, count := range distribution {
				bias := math.Abs(float64(count-expectedPerNode)) / float64(expectedPerNode) * 100
				t.Logf("  %s: %d keys (%.2f%% bias)", node, count, bias)
				if bias > maxBias {
					maxBias = bias
				}
			}
			t.Logf("  Max bias: %.2f%%", maxBias)

			// Test stathat/consistent for comparison
			c := consistent.New()
			c.NumberOfReplicas = tc.replicas
			for i := 1; i <= 5; i++ {
				c.Add(fmt.Sprintf("node%d", i))
			}

			distribution2 := make(map[string]int)
			for i := 0; i < numKeys; i++ {
				key := strconv.Itoa(i)
				node, _ := c.Get(key)
				distribution2[node]++
			}

			t.Logf("\n%s - stathat/consistent (5 nodes, %d keys):", tc.name, numKeys)
			var maxBias2 float64
			for node, count := range distribution2 {
				bias := math.Abs(float64(count-expectedPerNode)) / float64(expectedPerNode) * 100
				t.Logf("  %s: %d keys (%.2f%% bias)", node, count, bias)
				if bias > maxBias2 {
					maxBias2 = bias
				}
			}
			t.Logf("  Max bias: %.2f%%", maxBias2)
		})
	}
}

// TestDistributionWith10Nodes tests distribution with 10 physical nodes.
func TestDistributionWith10Nodes(t *testing.T) {
	testCases := []struct {
		name     string
		replicas int
	}{
		{"20 replicas", 20},
		{"160 replicas", 160},
		{"256 replicas", 256},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test our implementation
			ring := New().SetReplicas(tc.replicas)
			for i := 1; i <= 10; i++ {
				_ = ring.Add(fmt.Sprintf("node%d", i))
			}

			numKeys := 100000
			distribution := make(map[string]int)

			for i := 0; i < numKeys; i++ {
				key := strconv.Itoa(i)
				node, _ := ring.Get(key)
				distribution[node]++
			}

			expectedPerNode := numKeys / 10
			t.Logf("\n%s - Our implementation (10 nodes, %d keys):", tc.name, numKeys)

			var maxBias float64
			for node, count := range distribution {
				bias := math.Abs(float64(count-expectedPerNode)) / float64(expectedPerNode) * 100
				t.Logf("  %s: %d keys (%.2f%% bias)", node, count, bias)
				if bias > maxBias {
					maxBias = bias
				}
			}
			t.Logf("  Max bias: %.2f%%", maxBias)

			// Test stathat/consistent for comparison
			c := consistent.New()
			c.NumberOfReplicas = tc.replicas
			for i := 1; i <= 10; i++ {
				c.Add(fmt.Sprintf("node%d", i))
			}

			distribution2 := make(map[string]int)
			for i := 0; i < numKeys; i++ {
				key := strconv.Itoa(i)
				node, _ := c.Get(key)
				distribution2[node]++
			}

			t.Logf("\n%s - stathat/consistent (10 nodes, %d keys):", tc.name, numKeys)
			var maxBias2 float64
			for node, count := range distribution2 {
				bias := math.Abs(float64(count-expectedPerNode)) / float64(expectedPerNode) * 100
				t.Logf("  %s: %d keys (%.2f%% bias)", node, count, bias)
				if bias > maxBias2 {
					maxBias2 = bias
				}
			}
			t.Logf("  Max bias: %.2f%%", maxBias2)
		})
	}
}
