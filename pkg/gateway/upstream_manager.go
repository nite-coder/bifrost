package gateway

import (
	"sync"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/proxy"
)

// UpstreamManager manages the lifecycle of all global upstreams and caches target health states.
type UpstreamManager struct {
	mu        sync.RWMutex
	bifrost   *Bifrost
	upstreams map[string]*Upstream

	targetsMu sync.Mutex
	targets   map[string]*proxy.TargetState
}

// newUpstreamManager creates a new UpstreamManager instance.
func newUpstreamManager(bifrost *Bifrost) *UpstreamManager {
	return &UpstreamManager{
		bifrost:   bifrost,
		upstreams: make(map[string]*Upstream),
		targets:   make(map[string]*proxy.TargetState),
	}
}

// Start loads and initializes all upstreams.
func (m *UpstreamManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, upstreamOptions := range m.bifrost.options.Upstreams {
		upstreamOptions.ID = id
		u, err := newUpstream(m.bifrost, upstreamOptions)
		if err != nil {
			// clean up already loaded upstreams
			for _, upstream := range m.upstreams {
				_ = upstream.Close()
			}
			m.upstreams = make(map[string]*Upstream)
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
				if weight <= 0 {
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
				// clean up already loaded upstreams
				for _, upstream := range m.upstreams {
					_ = upstream.Close()
				}
				m.upstreams = make(map[string]*Upstream)
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

	m.targetsMu.Lock()
	m.targets = make(map[string]*proxy.TargetState)
	m.targetsMu.Unlock()

	return nil
}

// Get retrieves an upstream by its ID.
func (m *UpstreamManager) Get(id string) (*Upstream, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, found := m.upstreams[id]
	return u, found
}

// GetOrCreateTargetState retrieves or creates a TargetState for the physical target address.
func (m *UpstreamManager) GetOrCreateTargetState(
	address string,
	maxFails uint,
	failTimeout time.Duration,
) *proxy.TargetState {
	m.targetsMu.Lock()
	defer m.targetsMu.Unlock()

	state, found := m.targets[address]
	if !found {
		state = proxy.NewTargetState(maxFails, failTimeout)
		m.targets[address] = state
	}
	return state
}
