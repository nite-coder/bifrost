package gateway

import (
	"sync"

	"github.com/nite-coder/bifrost/pkg/config"
)

// UpstreamManager manages the lifecycle of all global upstreams and caches target health states.
type UpstreamManager struct {
	mu        sync.RWMutex
	bifrost   *Bifrost
	upstreams map[string]*Upstream
}

// newUpstreamManager creates a new UpstreamManager instance.
func newUpstreamManager(bifrost *Bifrost) *UpstreamManager {
	return &UpstreamManager{
		bifrost:   bifrost,
		upstreams: make(map[string]*Upstream),
	}
}

// Start loads and initializes all upstreams.
func (m *UpstreamManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	cleanup := func() {
		for _, upstream := range m.upstreams {
			_ = upstream.Close()
		}
		m.upstreams = make(map[string]*Upstream)
	}

	for id, upstreamOptions := range m.bifrost.options.Upstreams {
		upstreamOptions.ID = id
		u, err := newUpstream(m.bifrost, upstreamOptions)
		if err != nil {
			cleanup()
			return err
		}
		m.upstreams[id] = u
	}

	if m.bifrost.options.Models != nil {
		for modelID, modelOpts := range m.bifrost.options.Models {
			if len(modelID) == 0 {
				continue
			}

			var targets []config.TargetOptions
			for _, t := range modelOpts.Targets {
				weight := uint32(t.Weight) //nolint:gosec
				if weight == 0 {
					weight = 1
				}
				targets = append(targets, config.TargetOptions{
					Target: t.Target,
					Weight: weight,
				})
			}

			balancerType := "weighted"
			if modelOpts.Balancer != nil && modelOpts.Balancer.Type != "" {
				balancerType = modelOpts.Balancer.Type
			}

			upstreamOpts := config.UpstreamOptions{
				ID: "ai:" + modelID,
				Balancer: config.BalancerOptions{
					Type: balancerType,
				},
				Targets: targets,
			}

			u, err := newUpstream(m.bifrost, upstreamOpts)
			if err != nil {
				cleanup()
				return err
			}
			m.upstreams["ai:"+modelID] = u
		}
	}
	return nil
}

// Close closes all upstreams and clears states.
func (m *UpstreamManager) Close() error {
	m.mu.Lock()
	for _, u := range m.upstreams {
		_ = u.Close()
	}
	m.upstreams = make(map[string]*Upstream)
	m.mu.Unlock()

	return nil
}

// Get retrieves an upstream by its ID.
func (m *UpstreamManager) Get(id string) (*Upstream, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, found := m.upstreams[id]
	return u, found
}
