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
