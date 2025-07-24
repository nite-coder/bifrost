package gateway

import (
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
		ID:       "test",
		Strategy: "round_robin",
		Targets:  targetOptions,
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
			ID:       "test",
			Strategy: "round_robin",
			Targets:  targetOptions,
		},
	)

	assert.NoError(t, err)
	assert.Len(t, upstream.Balancer().Proxies(), 3)

	upstream, err = newUpstream(
		bifrost,
		config.ServiceOptions{
			Protocol: config.ProtocolGRPC,
			URL:      "http://test",
		},
		config.UpstreamOptions{
			ID:       "test",
			Strategy: "round_robin",
			Targets:  targetOptions,
		},
	)

	assert.NoError(t, err)
	proxiies := upstream.Balancer().Proxies()
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

		assert.Equal(t, plist1[0].ID(), plist2[0].ID())
		assert.Equal(t, plist1[1].ID(), plist2[1].ID())

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
