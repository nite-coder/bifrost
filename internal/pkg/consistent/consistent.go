package consistent

import (
	"errors"
	"hash/crc32"
	"slices"
	"strconv"
	"sync"
)

const (
	// DefaultReplicas is the default number of virtual nodes per physical node.
	// This value is based on Kong Gateway's configuration for achieving <1% distribution bias.
	DefaultReplicas = 160
)

var (
	// ErrEmptyRing is returned when trying to get a node from an empty ring.
	ErrEmptyRing = errors.New("consistent: hash ring is empty")
	// ErrNodeNotFound is returned when trying to remove a node that doesn't exist.
	ErrNodeNotFound = errors.New("consistent: node not found")
	// ErrNodeExists is returned when trying to add a node that already exists.
	ErrNodeExists = errors.New("consistent: node already exists")
)

// Consistent represents a consistent hashing ring with virtual nodes.
// It provides a thread-safe implementation of consistent hashing algorithm.
type Consistent struct {
	replicas    int                 // Number of virtual nodes per physical node (default 160)
	hashFunc    func([]byte) uint32 // Hash function (default FNV-1a)
	sortedNodes []uint32            // Sorted virtual node hash values
	ring        map[uint32]string   // Hash value to node ID mapping
	nodes       map[string]int      // Track added nodes and their replica counts
	mu          sync.RWMutex        // Concurrent safety
}

// New creates a new Consistent hash ring with default configuration:
// - 160 virtual nodes per physical node
// - CRC32 hash function.
func New() *Consistent {
	return &Consistent{
		replicas: DefaultReplicas,
		hashFunc: crc32hash,
		ring:     make(map[uint32]string),
		nodes:    make(map[string]int),
	}
}

// SetReplicas sets the number of virtual nodes per physical node.
// This method can be chained with other configuration methods.
// Returns the Consistent instance for method chaining.
func (c *Consistent) SetReplicas(replicas int) *Consistent {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.replicas = replicas
	return c
}

// SetHashFunc sets a custom hash function.
// This method can be chained with other configuration methods.
// Returns the Consistent instance for method chaining.
func (c *Consistent) SetHashFunc(hashFunc func([]byte) uint32) *Consistent {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hashFunc = hashFunc
	return c
}

// Add adds a physical node to the hash ring using the default replica count.
// It creates virtual nodes for the physical node and distributes them on the ring.
// Returns ErrNodeExists if the node already exists.
func (c *Consistent) Add(nodeID string) error {
	return c.AddWithReplicas(nodeID, c.replicas)
}

// AddWithReplicas adds or updates a physical node in the hash ring with a specific replica count.
// If the node already exists, it will be removed and re-added with the new replica count.
// This allows for weight-based consistent hashing by giving more virtual nodes to certain physical nodes.
func (c *Consistent) AddWithReplicas(nodeID string, replicas int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.nodes[nodeID]; exists {
		c.removeUnsafe(nodeID)
	}

	// Add virtual nodes for this physical node
	for i := 0; i < replicas; i++ {
		// Virtual node IDs are hashed to distribute them along the ring
		// Using idx+nodeID pattern for better distribution (same as stathat/consistent)
		virtualNodeKey := strconv.Itoa(i) + nodeID
		hash := c.hashFunc([]byte(virtualNodeKey))
		c.ring[hash] = nodeID
		c.sortedNodes = append(c.sortedNodes, hash)
	}

	// Sort the ring after adding new nodes
	slices.Sort(c.sortedNodes)
	c.nodes[nodeID] = replicas

	return nil
}

// Remove removes a physical node from the hash ring.
// It removes all virtual nodes associated with the physical node.
// Returns ErrNodeNotFound if the node doesn't exist.
func (c *Consistent) Remove(nodeID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.nodes[nodeID]; !exists {
		return ErrNodeNotFound
	}

	c.removeUnsafe(nodeID)
	return nil
}

// Get returns the physical node for the given key.
// It uses consistent hashing to find the closest node on the ring.
// Returns ErrEmptyRing if the ring is empty.
func (c *Consistent) Get(key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.sortedNodes) == 0 {
		return "", ErrEmptyRing
	}

	hash := c.hashFunc([]byte(key))

	// Binary search to find the first virtual node >= hash
	idx, _ := slices.BinarySearch(c.sortedNodes, hash)

	// If exact match not found, idx is the insertion point (first element > hash)
	// If not found and idx >= len, wrap around to the first node
	if idx >= len(c.sortedNodes) {
		idx = 0
	}

	return c.ring[c.sortedNodes[idx]], nil
}

// GetN returns the top N unique physical nodes in clockwise order starting from the hash of the key.
// This is used for consistent failover - if the first node is unavailable, the next nodes on the ring are used.
// Returns ErrEmptyRing if the ring is empty.
func (c *Consistent) GetN(key string, n int) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.sortedNodes) == 0 {
		return nil, ErrEmptyRing
	}

	if n <= 0 {
		return nil, nil
	}

	// Cannot return more nodes than existing physical nodes
	if n > len(c.nodes) {
		n = len(c.nodes)
	}

	hash := c.hashFunc([]byte(key))

	// Binary search to find the start point
	idx, _ := slices.BinarySearch(c.sortedNodes, hash)
	if idx >= len(c.sortedNodes) {
		idx = 0
	}

	res := make([]string, 0, n)
	seen := make(map[string]bool)

	// Traverse the ring clockwise to find unique physical nodes
	for i := 0; i < len(c.sortedNodes); i++ {
		currIdx := (idx + i) % len(c.sortedNodes)
		nodeID := c.ring[c.sortedNodes[currIdx]]
		if !seen[nodeID] {
			res = append(res, nodeID)
			seen[nodeID] = true
			if len(res) == n {
				break
			}
		}
	}

	return res, nil
}

// Nodes returns a list of all physical nodes in the ring.
func (c *Consistent) Nodes() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	nodes := make([]string, 0, len(c.nodes))
	for node := range c.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// IsEmpty returns true if the ring has no nodes.
func (c *Consistent) IsEmpty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.nodes) == 0
}

// removeUnsafe removes a node from the ring without locking.
// The caller must hold the lock.
func (c *Consistent) removeUnsafe(nodeID string) {
	replicas := c.nodes[nodeID]

	// Remove all virtual nodes for this physical node
	for i := 0; i < replicas; i++ {
		virtualNodeKey := strconv.Itoa(i) + nodeID
		hash := c.hashFunc([]byte(virtualNodeKey))
		delete(c.ring, hash)
	}

	// Rebuild sorted nodes list without the removed node's virtual nodes
	newSortedNodes := make([]uint32, 0, len(c.sortedNodes)-replicas)
	for _, hash := range c.sortedNodes {
		if _, exists := c.ring[hash]; exists {
			newSortedNodes = append(newSortedNodes, hash)
		}
	}

	c.sortedNodes = newSortedNodes
	delete(c.nodes, nodeID)
}

// crc32hash is a CRC32 hash function.
// It provides excellent distribution for consistent hashing.
func crc32hash(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
}
