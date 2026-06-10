package gateway

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/nite-coder/bifrost/pkg/proxy"
	"github.com/nite-coder/bifrost/pkg/resolver"
)

func TestCreateUpstreamAndDnsRefresh(t *testing.T) {
	targetOptions := []config.TargetOptions{
		{
			Target: "127.0.0.1:1234",
			Weight: 1,
			Tags: map[string]string{
				"id": "123",
			},
		},
		{
			Target: "127.0.0.2:1235",
			Weight: 1,
		},
		{
			Target: "127.0.0.3:1236",
			Weight: 1,
		},
	}

	upstreamOptions := config.UpstreamOptions{
		ID: "test",
		Balancer: config.BalancerOptions{
			Type: "round_robin",
		},
		Targets: targetOptions,
	}

	dnsResolver, err := resolver.NewResolver(resolver.Options{})
	require.NoError(t, err)

	bifrost := &Bifrost{
		options: &config.Options{
			SkipResolver: true,
			Upstreams: map[string]config.UpstreamOptions{
				"test": upstreamOptions,
			},
		},
		resolver: dnsResolver,
	}

	upstream, err := newUpstream(
		bifrost,
		config.UpstreamOptions{
			ID: "test",
			Balancer: config.BalancerOptions{
				Type: "round_robin",
			},
			Targets: targetOptions,
		},
	)
	require.NoError(t, err)

	ch := upstream.Subscribe()
	select {
	case endpoints := <-ch:
		assert.Len(t, endpoints, 3)
		var foundID string
		found := false
		for _, ep := range endpoints {
			if id, ok := ep.Tags["id"]; ok {
				foundID = id
				found = true
				break
			}
		}
		assert.True(t, found, "Expected to find an endpoint with an 'id' tag")
		assert.Equal(t, "123", foundID, "Expected 'id' tag to be '123'")
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for endpoints")
	}
}

func TestRefreshEndpoints(t *testing.T) {
	t.Run("success with initial DNS instances", func(t *testing.T) {
		dnsResolver, err := resolver.NewResolver(resolver.Options{})
		require.NoError(t, err)

		upstream := &Upstream{
			bifrost: &Bifrost{
				options: &config.Options{
					SkipResolver: true,
				},
				resolver: dnsResolver,
			},
			options: &config.UpstreamOptions{
				ID: "test",
				Discovery: config.DiscoveryOptions{
					Type: "dns",
					Name: "test.service",
				},
			},
		}

		addr1, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
		require.NoError(t, err)
		ins1 := provider.NewInstance(addr1, 2)

		addr2, err := net.ResolveTCPAddr("tcp", "127.0.0.2:8080")
		require.NoError(t, err)
		ins2 := provider.NewInstance(addr2, 3)

		instances := []provider.Instancer{ins1, ins2}

		ch := upstream.Subscribe()

		err = upstream.refreshEndpoints(instances)
		require.NoError(t, err)

		select {
		case endpoints := <-ch:
			assert.Len(t, endpoints, 2)
			assert.Equal(t, "127.0.0.1:8080", endpoints[0].Address)
			assert.Equal(t, uint32(2), endpoints[0].Weight)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for endpoints")
		}
	})

	t.Run("success with updated tags", func(t *testing.T) {
		dnsResolver, err := resolver.NewResolver(resolver.Options{})
		require.NoError(t, err)

		upstream := &Upstream{
			bifrost: &Bifrost{
				options: &config.Options{
					SkipResolver: true,
				},
				resolver: dnsResolver,
			},
			options: &config.UpstreamOptions{
				ID: "test",
				Discovery: config.DiscoveryOptions{
					Type: "dns",
					Name: "test.service",
				},
			},
		}

		addr1, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
		require.NoError(t, err)
		ins1 := provider.NewInstance(addr1, 2)
		ins1.SetTag("version", "v1")

		addr2, err := net.ResolveTCPAddr("tcp", "127.0.0.2:8080")
		require.NoError(t, err)
		ins2 := provider.NewInstance(addr2, 3)

		ch := upstream.Subscribe()

		// first refresh
		instances1 := []provider.Instancer{ins1, ins2}
		err = upstream.refreshEndpoints(instances1)
		require.NoError(t, err)

		select {
		case endpoints := <-ch:
			assert.Len(t, endpoints, 2)
			assert.Equal(t, "v1", endpoints[0].Tags["version"])
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for endpoints")
		}

		// second refresh with updated tags
		ins1WithNewTags := provider.NewInstance(addr1, 2)
		ins1WithNewTags.SetTag("version", "v2")
		instances2 := []provider.Instancer{ins1WithNewTags, ins2}
		err = upstream.refreshEndpoints(instances2)
		require.NoError(t, err)

		select {
		case endpoints := <-ch:
			assert.Len(t, endpoints, 2)
			assert.Equal(t, "v2", endpoints[0].Tags["version"])
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for endpoints")
		}
	})

	t.Run("fail with no instances", func(t *testing.T) {
		dnsResolver, err := resolver.NewResolver(resolver.Options{})
		require.NoError(t, err)

		upstream := &Upstream{
			bifrost: &Bifrost{
				options: &config.Options{
					SkipResolver: true,
				},
				resolver: dnsResolver,
			},
			options: &config.UpstreamOptions{
				ID: "test",
				Discovery: config.DiscoveryOptions{
					Type: "dns",
					Name: "test.service",
				},
			},
		}

		err = upstream.refreshEndpoints([]provider.Instancer{})
		require.Error(t, err)
	})
}

// mockErrorDiscovery is a mock service discovery that always returns an error.
type mockErrorDiscovery struct{}

func (m *mockErrorDiscovery) GetInstances(
	_ context.Context,
	_ provider.GetInstanceOptions,
) ([]provider.Instancer, error) {
	return nil, assert.AnError
}

func (m *mockErrorDiscovery) Watch(
	_ context.Context,
	_ provider.GetInstanceOptions,
) (<-chan []provider.Instancer, error) {
	return nil, assert.AnError
}

func (m *mockErrorDiscovery) Close() error {
	return nil
}

// TestWatchErrorHandling verifies that watch() returns early when Watch() fails.
func TestWatchErrorHandling(t *testing.T) {
	dnsResolver, err := resolver.NewResolver(resolver.Options{})
	require.NoError(t, err)

	upstream := &Upstream{
		bifrost: &Bifrost{
			options: &config.Options{
				SkipResolver: true,
			},
			resolver: dnsResolver,
		},
		options: &config.UpstreamOptions{
			ID: "test-upstream",
			Discovery: config.DiscoveryOptions{
				Type: "dns",
				Name: "test.service",
			},
		},
		discovery: &mockErrorDiscovery{},
	}

	// This should not panic or block indefinitely
	// The watch() should return early after logging the error
	upstream.watch()
}

func TestNewUpstreamValidation(t *testing.T) {
	dnsResolver, err := resolver.NewResolver(resolver.Options{})
	require.NoError(t, err)

	bifrost := &Bifrost{
		options: &config.Options{
			SkipResolver: true,
		},
		resolver: dnsResolver,
	}

	t.Run("empty upstream ID", func(t *testing.T) {
		_, err := newUpstream(
			bifrost,
			config.UpstreamOptions{
				ID:      "",
				Targets: []config.TargetOptions{{Target: "127.0.0.1:8080"}},
			},
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upstream ID cannot be empty")
	})

	t.Run("empty targets without discovery", func(t *testing.T) {
		_, err := newUpstream(
			bifrost,
			config.UpstreamOptions{
				ID:      "test",
				Targets: []config.TargetOptions{},
			},
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "targets cannot be empty")
	})

	t.Run("DNS discovery disabled", func(t *testing.T) {
		bifrostWithDNSDisabled := &Bifrost{
			options: &config.Options{
				SkipResolver: true,
				Providers: config.ProviderOptions{
					DNS: config.DNSProviderOptions{Enabled: false},
				},
			},
			resolver: dnsResolver,
		}

		_, err := newUpstream(
			bifrostWithDNSDisabled,
			config.UpstreamOptions{
				ID: "test",
				Discovery: config.DiscoveryOptions{
					Type: "dns",
					Name: "test.service",
				},
			},
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "dns provider is disabled")
	})

	t.Run("Nacos discovery disabled", func(t *testing.T) {
		bifrostWithNacosDisabled := &Bifrost{
			options: &config.Options{
				SkipResolver: true,
				Providers: config.ProviderOptions{
					Nacos: config.NacosProviderOptions{
						Discovery: config.NacosDiscoveryOptions{Enabled: false},
					},
				},
			},
			resolver: dnsResolver,
		}

		_, err := newUpstream(
			bifrostWithNacosDisabled,
			config.UpstreamOptions{
				ID: "test",
				Discovery: config.DiscoveryOptions{
					Type: "nacos",
					Name: "test.service",
				},
			},
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nacos discovery provider is disabled")
	})

	t.Run("K8S discovery disabled", func(t *testing.T) {
		bifrostWithK8SDisabled := &Bifrost{
			options: &config.Options{
				SkipResolver: true,
				Providers: config.ProviderOptions{
					K8S: config.K8SProviderOptions{Enabled: false},
				},
			},
			resolver: dnsResolver,
		}

		_, err := newUpstream(
			bifrostWithK8SDisabled,
			config.UpstreamOptions{
				ID: "test",
				Discovery: config.DiscoveryOptions{
					Type: "k8s",
					Name: "test.service",
				},
			},
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "k8s provider is disabled")
	})
}

func TestUpstream_TargetStatePersistence(t *testing.T) {
	bifrost := &Bifrost{
		options: &config.Options{},
	}

	upstreamOpts1 := config.UpstreamOptions{
		ID: "upstream1",
		Targets: []config.TargetOptions{
			{Target: "127.0.0.1:8080"},
		},
	}
	u1, err := newUpstream(bifrost, upstreamOpts1)
	require.NoError(t, err)
	defer func() {
		_ = u1.Close()
	}()

	// 1. Same upstream, same address -> same target state pointer across refreshes
	var state1 *proxy.TargetState
	u1.mu.Lock()
	state1 = u1.targets["127.0.0.1:8080"]
	u1.mu.Unlock()
	require.NotNil(t, state1)

	// Trigger a manual refresh with the same instance
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
	require.NoError(t, err)
	ins := provider.NewInstance(addr, 1)
	err = u1.refreshEndpoints([]provider.Instancer{ins})
	require.NoError(t, err)

	var state2 *proxy.TargetState
	u1.mu.Lock()
	state2 = u1.targets["127.0.0.1:8080"]
	u1.mu.Unlock()
	assert.Same(t, state1, state2, "Expected target state to persist across refreshes on the same Upstream")

	// 2. Different upstream, same address -> different target state (isolation)
	upstreamOpts2 := config.UpstreamOptions{
		ID: "upstream2",
		Targets: []config.TargetOptions{
			{Target: "127.0.0.1:8080"},
		},
	}
	u2, err := newUpstream(bifrost, upstreamOpts2)
	require.NoError(t, err)
	defer func() {
		_ = u2.Close()
	}()

	var state3 *proxy.TargetState
	u2.mu.Lock()
	state3 = u2.targets["127.0.0.1:8080"]
	u2.mu.Unlock()
	require.NotNil(t, state3)
	assert.NotSame(
		t,
		state1,
		state3,
		"Expected different Upstream instances to have isolated target states for the same address",
	)

	// 3. Different address, same upstream -> different target state
	upstreamOpts3 := config.UpstreamOptions{
		ID: "upstream3",
		Targets: []config.TargetOptions{
			{Target: "127.0.0.1:8080"},
			{Target: "127.0.0.1:8081"},
		},
	}
	u3, err := newUpstream(bifrost, upstreamOpts3)
	require.NoError(t, err)
	defer func() {
		_ = u3.Close()
	}()

	var state4, state5 *proxy.TargetState
	u3.mu.Lock()
	state4 = u3.targets["127.0.0.1:8080"]
	state5 = u3.targets["127.0.0.1:8081"]
	u3.mu.Unlock()
	require.NotNil(t, state4)
	require.NotNil(t, state5)
	assert.NotSame(
		t,
		state4,
		state5,
		"Expected different target addresses in the same Upstream to have distinct target states",
	)
}
