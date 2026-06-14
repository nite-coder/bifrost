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
	"github.com/nite-coder/bifrost/pkg/target"
)

const testAddr = "127.0.0.1:8080"

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

	dnsResolver, err := resolver.NewResolver(resolver.Options{SkipTest: true})
	require.NoError(t, err)

	bifrost := &Bifrost{
		options: &config.Options{
			SkipResolver: true,
			Default: config.DefaultOptions{
				Upstream: config.DefaultUpstreamOptions{
					MaxFails:    1,
					FailTimeout: time.Second,
				},
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

	endpoints := upstream.Endpoints()
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
}

func TestRefreshEndpoints(t *testing.T) {
	t.Run("success with initial DNS instances", func(t *testing.T) {
		dnsResolver, err := resolver.NewResolver(resolver.Options{SkipTest: true})
		require.NoError(t, err)

		upstream := &Upstream{
			bifrost: &Bifrost{
				options: &config.Options{
					SkipResolver: true,
					Default: config.DefaultOptions{
						Upstream: config.DefaultUpstreamOptions{
							MaxFails:    1,
							FailTimeout: time.Second,
						},
					},
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
			targets: make(map[string]*target.Target),
		}

		addr1, err := net.ResolveTCPAddr("tcp", testAddr)
		require.NoError(t, err)
		ins1 := provider.NewInstance(addr1, 2)

		addr2, err := net.ResolveTCPAddr("tcp", "127.0.0.2:8080")
		require.NoError(t, err)
		ins2 := provider.NewInstance(addr2, 3)

		results := []provider.DiscoveryResult{
			{Target: testAddr, Nodes: []provider.Instancer{ins1}},
			{Target: "127.0.0.2:8080", Nodes: []provider.Instancer{ins2}},
		}

		ch := upstream.Subscribe()

		err = upstream.refreshEndpoints(results)
		require.NoError(t, err)

		select {
		case endpoints := <-ch:
			assert.Len(t, endpoints, 2)
			var foundEP *target.Endpoint
			for _, ep := range endpoints {
				if ep.Address == testAddr {
					foundEP = ep
					break
				}
			}
			require.NotNil(t, foundEP)
			assert.Equal(t, uint32(2), foundEP.Weight)
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for endpoints")
		}
	})

	t.Run("success with updated tags", func(t *testing.T) {
		dnsResolver, err := resolver.NewResolver(resolver.Options{SkipTest: true})
		require.NoError(t, err)

		upstream := &Upstream{
			bifrost: &Bifrost{
				options: &config.Options{
					SkipResolver: true,
					Default: config.DefaultOptions{
						Upstream: config.DefaultUpstreamOptions{
							MaxFails:    1,
							FailTimeout: time.Second,
						},
					},
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
			targets: make(map[string]*target.Target),
		}

		addr1, err := net.ResolveTCPAddr("tcp", testAddr)
		require.NoError(t, err)
		ins1 := provider.NewInstance(addr1, 2)
		ins1.SetTag("version", "v1")

		addr2, err := net.ResolveTCPAddr("tcp", "127.0.0.2:8080")
		require.NoError(t, err)
		ins2 := provider.NewInstance(addr2, 3)

		ch := upstream.Subscribe()

		// first refresh
		results1 := []provider.DiscoveryResult{
			{Target: testAddr, Nodes: []provider.Instancer{ins1}},
			{Target: "127.0.0.2:8080", Nodes: []provider.Instancer{ins2}},
		}
		err = upstream.refreshEndpoints(results1)
		require.NoError(t, err)

		select {
		case endpoints := <-ch:
			assert.Len(t, endpoints, 2)
			var foundEP *target.Endpoint
			for _, ep := range endpoints {
				if ep.Address == testAddr {
					foundEP = ep
					break
				}
			}
			require.NotNil(t, foundEP)
			assert.Equal(t, "v1", foundEP.Tags["version"])
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for endpoints")
		}

		// second refresh with updated tags
		ins1WithNewTags := provider.NewInstance(addr1, 2)
		ins1WithNewTags.SetTag("version", "v2")
		results2 := []provider.DiscoveryResult{
			{Target: testAddr, Nodes: []provider.Instancer{ins1WithNewTags}},
			{Target: "127.0.0.2:8080", Nodes: []provider.Instancer{ins2}},
		}
		err = upstream.refreshEndpoints(results2)
		require.NoError(t, err)

		select {
		case endpoints := <-ch:
			assert.Len(t, endpoints, 2)
			var foundEP *target.Endpoint
			for _, ep := range endpoints {
				if ep.Address == testAddr {
					foundEP = ep
					break
				}
			}
			require.NotNil(t, foundEP)
			assert.Equal(t, "v2", foundEP.Tags["version"])
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for endpoints")
		}
	})

	t.Run("fail with no instances", func(t *testing.T) {
		dnsResolver, err := resolver.NewResolver(resolver.Options{SkipTest: true})
		require.NoError(t, err)

		upstream := &Upstream{
			bifrost: &Bifrost{
				options: &config.Options{
					SkipResolver: true,
					Default: config.DefaultOptions{
						Upstream: config.DefaultUpstreamOptions{
							MaxFails:    1,
							FailTimeout: time.Second,
						},
					},
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

		err = upstream.refreshEndpoints([]provider.DiscoveryResult{})
		require.Error(t, err)
	})
}

// mockErrorDiscovery is a mock service discovery that always returns an error.
type mockErrorDiscovery struct{}

func (m *mockErrorDiscovery) GetInstances(
	_ context.Context,
	_ provider.GetInstanceOptions,
) ([]provider.DiscoveryResult, error) {
	return nil, assert.AnError
}

func (m *mockErrorDiscovery) Watch(
	_ context.Context,
	_ provider.GetInstanceOptions,
) (<-chan []provider.DiscoveryResult, error) {
	return nil, assert.AnError
}

func (m *mockErrorDiscovery) Close() error {
	return nil
}

// TestWatchErrorHandling verifies that watch() returns early when Watch() fails.
func TestWatchErrorHandling(t *testing.T) {
	dnsResolver, err := resolver.NewResolver(resolver.Options{SkipTest: true})
	require.NoError(t, err)

	upstream := &Upstream{
		bifrost: &Bifrost{
			options: &config.Options{
				SkipResolver: true,
				Default: config.DefaultOptions{
					Upstream: config.DefaultUpstreamOptions{
						MaxFails:    1,
						FailTimeout: time.Second,
					},
				},
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
	dnsResolver, err := resolver.NewResolver(resolver.Options{SkipTest: true})
	require.NoError(t, err)

	bifrost := &Bifrost{
		options: &config.Options{
			SkipResolver: true,
			Default: config.DefaultOptions{
				Upstream: config.DefaultUpstreamOptions{
					MaxFails:    1,
					FailTimeout: time.Second,
				},
			},
		},
		resolver: dnsResolver,
	}

	t.Run("empty upstream ID", func(t *testing.T) {
		_, err := newUpstream(
			bifrost,
			config.UpstreamOptions{
				ID:      "",
				Targets: []config.TargetOptions{{Target: testAddr}},
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
				Default: config.DefaultOptions{
					Upstream: config.DefaultUpstreamOptions{
						MaxFails:    1,
						FailTimeout: time.Second,
					},
				},
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
				Default: config.DefaultOptions{
					Upstream: config.DefaultUpstreamOptions{
						MaxFails:    1,
						FailTimeout: time.Second,
					},
				},
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
				Default: config.DefaultOptions{
					Upstream: config.DefaultUpstreamOptions{
						MaxFails:    1,
						FailTimeout: time.Second,
					},
				},
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
	dnsResolver, err := resolver.NewResolver(resolver.Options{SkipTest: true})
	require.NoError(t, err)

	bifrost := &Bifrost{
		options: &config.Options{
			SkipResolver: true,
			Default: config.DefaultOptions{
				Upstream: config.DefaultUpstreamOptions{
					MaxFails:    1,
					FailTimeout: time.Second,
				},
			},
		},
		resolver: dnsResolver,
	}

	upstreamOpts1 := config.UpstreamOptions{
		ID: "upstream1",
		Balancer: config.BalancerOptions{
			Type: "round_robin",
		},
		Targets: []config.TargetOptions{
			{Target: testAddr},
		},
	}
	u1, err := newUpstream(bifrost, upstreamOpts1)
	require.NoError(t, err)
	defer func() {
		_ = u1.Close()
	}()

	// 1. Same upstream, same address -> same target state pointer across refreshes
	var state1 *target.State
	u1.mu.Lock()
	state1 = u1.targets[testAddr].Endpoints[testAddr].State
	u1.mu.Unlock()
	require.NotNil(t, state1)

	// Trigger a manual refresh with the same instance
	addr, err := net.ResolveTCPAddr("tcp", testAddr)
	require.NoError(t, err)
	ins := provider.NewInstance(addr, 1)
	results := []provider.DiscoveryResult{
		{Target: testAddr, Nodes: []provider.Instancer{ins}},
	}
	err = u1.refreshEndpoints(results)
	require.NoError(t, err)

	var state2 *target.State
	u1.mu.Lock()
	state2 = u1.targets[testAddr].Endpoints[testAddr].State
	u1.mu.Unlock()
	assert.Same(t, state1, state2, "Expected target state to persist across refreshes on the same Upstream")

	// 2. Different upstream, same address -> different target state (isolation)
	upstreamOpts2 := config.UpstreamOptions{
		ID: "upstream2",
		Balancer: config.BalancerOptions{
			Type: "round_robin",
		},
		Targets: []config.TargetOptions{
			{Target: testAddr},
		},
	}
	u2, err := newUpstream(bifrost, upstreamOpts2)
	require.NoError(t, err)
	defer func() {
		_ = u2.Close()
	}()

	var state3 *target.State
	u2.mu.Lock()
	state3 = u2.targets[testAddr].Endpoints[testAddr].State
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
		Balancer: config.BalancerOptions{
			Type: "round_robin",
		},
		Targets: []config.TargetOptions{
			{Target: testAddr},
			{Target: "127.0.0.1:8081"},
		},
	}
	u3, err := newUpstream(bifrost, upstreamOpts3)
	require.NoError(t, err)
	defer func() {
		_ = u3.Close()
	}()

	var state4, state5 *target.State
	u3.mu.Lock()
	state4 = u3.targets[testAddr].Endpoints[testAddr].State
	state5 = u3.targets["127.0.0.1:8081"].Endpoints["127.0.0.1:8081"].State
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

func TestUpstream_HoldsBalancer(t *testing.T) {
	dnsResolver, err := resolver.NewResolver(resolver.Options{SkipTest: true})
	require.NoError(t, err)

	bifrost := &Bifrost{
		options: &config.Options{
			SkipResolver: true,
			Default: config.DefaultOptions{
				Upstream: config.DefaultUpstreamOptions{
					MaxFails:    1,
					FailTimeout: time.Second,
				},
			},
		},
		resolver: dnsResolver,
	}

	upstream, err := newUpstream(bifrost, config.UpstreamOptions{
		ID: "test",
		Balancer: config.BalancerOptions{
			Type: "round_robin",
		},
		Targets: []config.TargetOptions{
			{Target: "127.0.0.1:1234", Weight: 1},
			{Target: "127.0.0.2:1235", Weight: 1},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, upstream.Balancer(), "upstream should have a balancer after creation")

	ep, err := upstream.Balancer().Select(context.Background(), nil)
	require.NoError(t, err)
	assert.Contains(t, []string{"127.0.0.1:1234", "127.0.0.2:1235"}, ep.Address)
}

func TestUpstream_TargetGrouping(t *testing.T) {
	t.Run("targets from config are pre-populated", func(t *testing.T) {
		dnsResolver, err := resolver.NewResolver(resolver.Options{SkipTest: true})
		require.NoError(t, err)

		bifrost := &Bifrost{
			options: &config.Options{
				SkipResolver: true,
				Default: config.DefaultOptions{
					Upstream: config.DefaultUpstreamOptions{
						MaxFails:    1,
						FailTimeout: time.Second,
					},
				},
			},
			resolver: dnsResolver,
		}

		upstream, err := newUpstream(bifrost, config.UpstreamOptions{
			ID: "test",
			Targets: []config.TargetOptions{
				{Target: "10.0.1.1:80", Weight: 100, Tags: map[string]string{"region": "us"}},
				{Target: "10.0.1.5:8080", Weight: 50},
			},
			Balancer: config.BalancerOptions{Type: "round_robin"},
		})
		require.NoError(t, err)
		require.Len(t, upstream.targets, 2)
		assert.Equal(t, uint32(100), upstream.targets["10.0.1.1:80"].Weight)
		assert.Equal(t, "us", upstream.targets["10.0.1.1:80"].Tags["region"])
		assert.Equal(t, uint32(50), upstream.targets["10.0.1.5:8080"].Weight)
	})

	t.Run("endpoints are grouped under correct target", func(t *testing.T) {
		dnsResolver, err := resolver.NewResolver(resolver.Options{SkipTest: true})
		require.NoError(t, err)

		bifrost := &Bifrost{
			options: &config.Options{
				SkipResolver: true,
				Default: config.DefaultOptions{
					Upstream: config.DefaultUpstreamOptions{
						MaxFails:    1,
						FailTimeout: time.Second,
					},
				},
			},
			resolver: dnsResolver,
		}

		upstream, err := newUpstream(bifrost, config.UpstreamOptions{
			ID: "test",
			Targets: []config.TargetOptions{
				{Target: "127.0.0.1:1234", Weight: 1},
				{Target: "127.0.0.2:1235", Weight: 2},
			},
			Balancer: config.BalancerOptions{Type: "round_robin"},
		})
		require.NoError(t, err)

		assert.Len(t, upstream.targets["127.0.0.1:1234"].Endpoints, 1)
		assert.Equal(t, uint32(1), upstream.targets["127.0.0.1:1234"].Endpoints["127.0.0.1:1234"].Weight)

		assert.Len(t, upstream.targets["127.0.0.2:1235"].Endpoints, 1)
		assert.Equal(t, uint32(2), upstream.targets["127.0.0.2:1235"].Endpoints["127.0.0.2:1235"].Weight)
	})

	t.Run("flattenEndpoints returns all endpoints from all targets", func(t *testing.T) {
		dnsResolver, err := resolver.NewResolver(resolver.Options{SkipTest: true})
		require.NoError(t, err)

		bifrost := &Bifrost{
			options: &config.Options{
				SkipResolver: true,
				Default: config.DefaultOptions{
					Upstream: config.DefaultUpstreamOptions{
						MaxFails:    1,
						FailTimeout: time.Second,
					},
				},
			},
			resolver: dnsResolver,
		}

		upstream, err := newUpstream(bifrost, config.UpstreamOptions{
			ID: "test",
			Targets: []config.TargetOptions{
				{Target: "127.0.0.1:1234", Weight: 1},
				{Target: "127.0.0.2:1235", Weight: 2},
			},
			Balancer: config.BalancerOptions{Type: "round_robin"},
		})
		require.NoError(t, err)

		flat := upstream.flattenEndpoints()
		require.Len(t, flat, 2)

		addrs := make([]string, len(flat))
		for i, ep := range flat {
			addrs[i] = ep.Address
		}
		assert.Contains(t, addrs, "127.0.0.1:1234")
		assert.Contains(t, addrs, "127.0.0.2:1235")
	})
}
