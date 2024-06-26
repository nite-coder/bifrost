package gateway

import (
	"hash/fnv"
	"math/rand"
	"testing"
	"time"
)

func TestUpstreamHashing(t *testing.T) {
	// Test when there is only one proxy
	u := &Upstream{
		proxies: []*ReverseProxy{
			{},
		},
		hasher: fnv.New32(),
	}
	result := u.hasing("key")
	if result != u.proxies[0] {
		t.Errorf("Expected %v, got %v", u.proxies[0], result)
	}

	// Test when there are multiple proxies
	u = &Upstream{
		proxies: []*ReverseProxy{
			{},
			{},
			{},
		},
		hasher: fnv.New32(),
	}
	result = u.hasing("key")
	if result != u.proxies[0] {
		t.Errorf("Expected %v, got %v", u.proxies[0], result)
	}

	result = u.hasing("another key")
	if result != u.proxies[1] {
		t.Errorf("Expected %v, got %v", u.proxies[1], result)
	}
}

func TestUpstreamRandom(t *testing.T) {
	// Test when there is only one proxy
	u := &Upstream{
		proxies: []*ReverseProxy{
			{},
		},
		rng: rand.New(rand.NewSource(time.Now().Unix())),
	}
	result := u.random()
	if result != u.proxies[0] {
		t.Errorf("Expected %v, got %v", u.proxies[0], result)
	}

	// Test when there are multiple proxies
	u = &Upstream{
		proxies: []*ReverseProxy{
			{},
			{},
			{},
		},
		rng: rand.New(rand.NewSource(time.Now().Unix())),
	}
	result = u.random()
	if result == nil {
		t.Errorf("Expected a non-nil result")
	}
}
