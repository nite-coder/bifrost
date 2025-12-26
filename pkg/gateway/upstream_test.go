package gateway

import (
	"context"
	"net"
	"testing"

	_ "github.com/nite-coder/bifrost/pkg/balancer/roundrobin"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/nite-coder/bifrost/pkg/resolver"
	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err)

	bifrost := &Bifrost{
		options: &config.Options{
			SkipResolver: true,
			Upstreams: map[string]config.UpstreamOptions{
				"test": upstreamOptions,
			},
		},
		resolver: dnsResolver,
	}

	testService := config.ServiceOptions{
		Protocol: config.ProtocolHTTP,
		URL:      "http://test",
	}

	upstreamsMap, err := loadUpstreams(bifrost, testService)
	assert.NoError(t, err)
	assert.Len(t, upstreamsMap, 1)

	upstream, err := newUpstream(
		bifrost,
		testService,
		config.UpstreamOptions{
			ID: "test",
			Balancer: config.BalancerOptions{
				Type: "round_robin",
			},
			Targets: targetOptions,
		},
	)

	assert.NoError(t, err)
	proxiies := upstream.Balancer().Proxies()
	assert.Len(t, proxiies, 3)

	var foundID string
	found := false
	for _, proxy := range proxiies {
		if id, ok := proxy.Tag("id"); ok {
			foundID = id
			found = true
			break
		}
	}
	assert.True(t, found, "Expected to find a proxy with an 'id' tag")
	assert.Equal(t, "123", foundID, "Expected 'id' tag to be '123'")

	upstream, err = newUpstream(
		bifrost,
		config.ServiceOptions{
			Protocol: config.ProtocolGRPC,
			URL:      "http://test",
		},
		config.UpstreamOptions{
			ID: "test",
			Balancer: config.BalancerOptions{
				Type: "round_robin",
			},
			Targets: targetOptions,
		},
	)
	assert.NoError(t, err)
	proxiies = upstream.Balancer().Proxies()
	assert.Len(t, proxiies, 3)
}

func TestRefreshProxies(t *testing.T) {
	t.Run("success with initial DNS instances", func(t *testing.T) {
		dnsResolver, err := resolver.NewResolver(resolver.Options{})
		assert.NoError(t, err)

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
			serviceOptions: &config.ServiceOptions{
				Protocol: config.ProtocolHTTP,
				URL:      "http://test.service",
			},
		}

		addr1, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
		assert.NoError(t, err)
		ins1 := provider.NewInstance(addr1, 2)

		addr2, err := net.ResolveTCPAddr("tcp", "127.0.0.2:8080")
		assert.NoError(t, err)
		ins2 := provider.NewInstance(addr2, 3)

		instances := []provider.Instancer{ins1, ins2}

		err = upstream.refreshProxies(instances)
		assert.NoError(t, err)

		plist1 := upstream.Balancer().Proxies()
		assert.Len(t, plist1, 2)

		// should be no update
		err = upstream.refreshProxies(instances)
		assert.NoError(t, err)

		plist2 := upstream.Balancer().Proxies()
		assert.Len(t, plist2, 2)

		plist1IDs := make(map[string]struct{})
		for _, p := range plist1 {
			plist1IDs[p.ID()] = struct{}{}
		}

		plist2IDs := make(map[string]struct{})
		for _, p := range plist2 {
			plist2IDs[p.ID()] = struct{}{}
		}

		assert.Equal(t, plist1IDs, plist2IDs, "Expected proxy IDs to be the same regardless of order")

	})

	t.Run("success with updated tags", func(t *testing.T) {
		dnsResolver, err := resolver.NewResolver(resolver.Options{})
		assert.NoError(t, err)

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
			serviceOptions: &config.ServiceOptions{
				Protocol: config.ProtocolHTTP,
				URL:      "http://test.service",
			},
		}

		addr1, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
		assert.NoError(t, err)
		ins1 := provider.NewInstance(addr1, 2)
		ins1.SetTag("version", "v1")

		addr2, err := net.ResolveTCPAddr("tcp", "127.0.0.2:8080")
		assert.NoError(t, err)
		ins2 := provider.NewInstance(addr2, 3)

		// first refresh
		instances1 := []provider.Instancer{ins1, ins2}
		err = upstream.refreshProxies(instances1)
		assert.NoError(t, err)
		plist1 := upstream.Balancer().Proxies()
		assert.Len(t, plist1, 2)

		// second refresh with updated tags
		ins1WithNewTags := provider.NewInstance(addr1, 2)
		ins1WithNewTags.SetTag("version", "v2")
		instances2 := []provider.Instancer{ins1WithNewTags, ins2}
		err = upstream.refreshProxies(instances2)
		assert.NoError(t, err)
		plist2 := upstream.Balancer().Proxies()
		assert.Len(t, plist2, 2)

		// The proxy with the same target but different tags should be replaced, so their IDs should not be equal.
		// The other proxy should remain the same.
		plist1Map := make(map[string]string)
		for _, p := range plist1 {
			plist1Map[p.Target()] = p.ID()
		}

		plist2Map := make(map[string]string)
		for _, p := range plist2 {
			plist2Map[p.Target()] = p.ID()
		}

		assert.NotEqual(t, plist1Map["http://127.0.0.1:8080"], plist2Map["http://127.0.0.1:8080"], "proxy should be updated due to tag changes")
		assert.Equal(t, plist1Map["http://127.0.0.2:8080"], plist2Map["http://127.0.0.2:8080"], "proxy without tag changes should remain the same")
	})

	t.Run("fail with no instances", func(t *testing.T) {
		dnsResolver, err := resolver.NewResolver(resolver.Options{})
		assert.NoError(t, err)

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
			serviceOptions: &config.ServiceOptions{
				Protocol: config.ProtocolHTTP,
				URL:      "http://test.service",
			},
		}

		err = upstream.refreshProxies([]provider.Instancer{})
		assert.Error(t, err)
	})

}

// mockErrorDiscovery is a mock service discovery that always returns an error
type mockErrorDiscovery struct{}

func (m *mockErrorDiscovery) GetInstances(ctx context.Context, opts provider.GetInstanceOptions) ([]provider.Instancer, error) {
	return nil, assert.AnError
}

func (m *mockErrorDiscovery) Watch(ctx context.Context, opts provider.GetInstanceOptions) (<-chan []provider.Instancer, error) {
	return nil, assert.AnError
}

// TestWatchErrorHandling verifies that watch() returns early when Watch() fails
func TestWatchErrorHandling(t *testing.T) {
	dnsResolver, err := resolver.NewResolver(resolver.Options{})
	assert.NoError(t, err)

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
		serviceOptions: &config.ServiceOptions{
			Protocol: config.ProtocolHTTP,
			URL:      "http://test.service",
		},
		discovery: &mockErrorDiscovery{},
	}

	// This should not panic or block indefinitely
	// The watch() should return early after logging the error
	upstream.watch()

	// If we reach here without blocking, the test passes
	assert.True(t, true, "watch() should return early on error")
}
