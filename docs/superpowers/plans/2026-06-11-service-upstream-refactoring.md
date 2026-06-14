# Service and Upstream Refactoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor Bifrost so Upstream owns the Balancer + Target grouping (Kong model) and Service manages a proxy cache, while config is 100% backward compatible.

**Architecture:**
```
Upstream (shared singleton)
  └── Targets[] (grouped by hostname from config TargetOptions.Target)
        └── Endpoints[] (resolved IP:port, weight from instance)
              └── State (health tracking, shared across all services)
```

Upstream holds `targets map[string]*Target` (persistent Target structs, each with `map[string]*Endpoint`) + `balancer.Balancer`. Service fetches initial state via `upstream.Endpoints()` synchronously, then subscribes to ongoing updates via `upstream.Subscribe()` (which sends only future changes). Maintains `proxyByAddress sync.Map` for lock-free reads. In `ServeHTTP()` calls `upstream.Balancer().Select()` then `proxyByAddress.Load(endpoint.Address)`. New `pkg/target` package extracted from `pkg/proxy` for Endpoint, State, and Target types.

**Tech Stack:** Go, Hertz, gRPC, table-driven tests with testify

**TDD Rules:**
- Every production code change MUST be preceded by a failing test
- No commits — user handles commits manually
- Run `make check` after all tasks to confirm lint (0 issues) and tests (all pass)

---

### Task 0: Pre-work — Explore existing test patterns, disable commits

**Files:** None

- [ ] **Step 1: Check current test patterns**

Run: `grep -r "assert\|require" pkg/balancer/*_test.go pkg/balancer/*/*_test.go | head -5`
Expected: testify assertions used throughout

Run: `make check`
Goal: Record current pass/fail state to compare later.

Expected: all tests pass, lint 0 issues.

- [ ] **Step 2: Confirm no dirty state**

Run: `git status --short`
Expected: clean working tree (or user's existing WIP we need to preserve).

---

### Task 1: Create `pkg/target` package — State, Endpoint, EndpointHash

This is a net-new package with zero dependencies on other bifrost packages. It extracts types from `pkg/proxy/endpoint.go`.

**Files:**
- Create: `pkg/target/state.go`
- Create: `pkg/target/endpoint.go`
- Create: `pkg/target/target.go`
- Create: `pkg/target/hash.go`
- Create: `pkg/target/state_test.go`
- Create: `pkg/target/endpoint_test.go`
- Create: `pkg/target/target_test.go`
- Create: `pkg/target/hash_test.go`

- [ ] **Step 1: Write failing test for State**

File: `pkg/target/state_test.go`

```go
package target_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/nite-coder/bifrost/pkg/target"
)

func TestState_IsAvailable_NoMaxFails(t *testing.T) {
	s := target.NewState(0, time.Second)
	assert.True(t, s.IsAvailable())
	s.RecordFailure()
	assert.True(t, s.IsAvailable(), "no max fails means always available")
}

func TestState_FailThenRecover(t *testing.T) {
	s := target.NewState(2, 50*time.Millisecond)
	assert.True(t, s.IsAvailable())

	s.RecordFailure()
	assert.True(t, s.IsAvailable(), "one failure < maxFails=2")

	s.RecordFailure()
	assert.False(t, s.IsAvailable(), "two failures >= maxFails=2")

	assert.Eventually(t, func() bool {
		return s.IsAvailable()
	}, 200*time.Millisecond, 10*time.Millisecond, "should recover after failTimeout")
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./pkg/target/ -run TestState -v`
Expected: FAIL — package/target doesn't exist yet

- [ ] **Step 3: Write State implementation**

File: `pkg/target/state.go`

```go
package target

import (
	"sync"
	"time"

	"github.com/nite-coder/bifrost/pkg/timecache"
)

type State struct {
	mu           sync.RWMutex
	failedCount  uint
	maxFails     uint
	failTimeout  time.Duration
	failExpireAt time.Time
}

func NewState(maxFails uint, failTimeout time.Duration) *State {
	return &State{
		maxFails:    maxFails,
		failTimeout: failTimeout,
	}
}

func (s *State) IsAvailable() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.maxFails == 0 {
		return true
	}
	now := timecache.Now()
	if now.After(s.failExpireAt) {
		return true
	}
	return s.failedCount < s.maxFails
}

func (s *State) RecordFailure() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := timecache.Now()
	if now.After(s.failExpireAt) {
		s.failExpireAt = now.Add(s.failTimeout)
		s.failedCount = 1
	} else {
		s.failedCount++
	}
}
```

- [ ] **Step 4: Run test, expect pass**

Run: `go test ./pkg/target/ -run TestState -v`
Expected: PASS

- [ ] **Step 5: Write failing test for Endpoint**

File: `pkg/target/endpoint_test.go`

```go
package target_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nite-coder/bifrost/pkg/target"
)

func TestEndpoint_Fields(t *testing.T) {
	s := target.NewState(1, 0)
	ep := &target.Endpoint{
		Address: "10.0.1.5:8080",
		Weight:  10,
		Tags:    map[string]string{"server_name": "example.com"},
		State:   s,
	}
	assert.Equal(t, "10.0.1.5:8080", ep.Address)
	assert.Equal(t, uint32(10), ep.Weight)
	assert.Equal(t, "example.com", ep.Tags["server_name"])
	assert.Same(t, s, ep.State)
}
```

- [ ] **Step 6: Run test, expect failure**

Run: `go test ./pkg/target/ -run TestEndpoint -v`
Expected: FAIL — Endpoint not defined

- [ ] **Step 7: Write Endpoint implementation**

File: `pkg/target/endpoint.go`

```go
package target

type Endpoint struct {
	Address string
	Weight  uint32
	Tags    map[string]string
	State   *State
}
```

- [ ] **Step 8: Run test, expect pass**

Run: `go test ./pkg/target/ -run TestEndpoint -v`
Expected: PASS

- [ ] **Step 9: Write failing test for Target struct**

File: `pkg/target/target_test.go`

```go
package target_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nite-coder/bifrost/pkg/target"
)

func TestTarget_Fields(t *testing.T) {
	tgt := &target.Target{
		Name:   "example.com:80",
		Weight: 100,
		Tags:   map[string]string{"region": "us"},
		Endpoints: map[string]*target.Endpoint{
			"10.0.1.1:80": {Address: "10.0.1.1:80", Weight: 100},
			"10.0.1.2:80": {Address: "10.0.1.2:80", Weight: 100},
		},
	}
	assert.Equal(t, "example.com:80", tgt.Name)
	assert.Equal(t, uint32(100), tgt.Weight)
	assert.Equal(t, "us", tgt.Tags["region"])
	assert.Len(t, tgt.Endpoints, 2)
}
```

- [ ] **Step 10: Run test, expect failure**

Run: `go test ./pkg/target/ -run TestTarget -v`
Expected: FAIL — Target not defined

- [ ] **Step 11: Write Target implementation**

File: `pkg/target/target.go`

```go
package target

type Target struct {
	Name      string
	Weight    uint32
	Tags      map[string]string
	Endpoints map[string]*Endpoint
}
```

- [ ] **Step 12: Run test, expect pass**

Run: `go test ./pkg/target/ -run TestTarget -v`
Expected: PASS

- [ ] **Step 13: Write failing test for EndpointHash**

File: `pkg/target/hash_test.go`

```go
package target_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nite-coder/bifrost/pkg/target"
)

func mustStrSort(t *testing.T, ss []string) []string {
	t.Helper()
	sort.Strings(ss)
	return ss
}

func TestEndpointHash_Deterministic(t *testing.T) {
	eps := []*target.Endpoint{
		{Address: "10.0.1.1:80", Weight: 1, Tags: map[string]string{"region": "us"}},
		{Address: "10.0.1.2:80", Weight: 2, Tags: nil},
	}
	h1 := target.EndpointHash(eps)
	h2 := target.EndpointHash(eps)
	assert.Equal(t, h1, h2, "same input must produce same hash")

	// different order: hash must still match (sorted internally)
	eps2 := []*target.Endpoint{
		{Address: "10.0.1.2:80", Weight: 2, Tags: nil},
		{Address: "10.0.1.1:80", Weight: 1, Tags: map[string]string{"region": "us"}},
	}
	h3 := target.EndpointHash(eps2)
	assert.Equal(t, h1, h3, "order-independent hash")
}

func TestEndpointHash_ChangesOnDiff(t *testing.T) {
	ep := &target.Endpoint{Address: "10.0.1.1:80", Weight: 1}
	eps := []*target.Endpoint{ep}

	h := target.EndpointHash(eps)

	// weight change
	ep.Weight = 2
	h2 := target.EndpointHash([]*target.Endpoint{ep})
	assert.NotEqual(t, h, h2, "weight change must change hash")

	// address change
	ep2 := &target.Endpoint{Address: "10.0.1.2:80", Weight: 1}
	h3 := target.EndpointHash([]*target.Endpoint{ep2})
	assert.NotEqual(t, h2, h3, "address change must change hash")
}

func TestEndpointHash_ExcludesState(t *testing.T) {
	s1 := target.NewState(1, 0)
	s2 := target.NewState(2, 0)
	ep := &target.Endpoint{Address: "10.0.1.1:80", Weight: 1, State: s1}
	eps := []*target.Endpoint{ep}
	h := target.EndpointHash(eps)

	ep.State = s2 // different State, same address/weight/tags
	h2 := target.EndpointHash(eps)
	assert.Equal(t, h, h2, "State change must NOT change hash")
}
```

- [ ] **Step 14: Run test, expect failure**

Run: `go test ./pkg/target/ -run TestEndpointHash -v`
Expected: FAIL — EndpointHash not defined

- [ ] **Step 15: Write EndpointHash implementation**

File: `pkg/target/hash.go`

```go
package target

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

func EndpointHash(endpoints []*Endpoint) string {
	sorted := make([]*Endpoint, len(endpoints))
	copy(sorted, endpoints)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Address < sorted[j].Address
	})

	h := sha256.New()
	for _, ep := range sorted {
		_, _ = fmt.Fprintf(h, "%s|%d|", ep.Address, ep.Weight)
		if len(ep.Tags) > 0 {
			keys := make([]string, 0, len(ep.Tags))
			for k := range ep.Tags {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				_, _ = fmt.Fprintf(h, "%s=%s|", k, ep.Tags[k])
			}
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}
```

- [ ] **Step 16: Run test, expect pass**

Run: `go test ./pkg/target/ -v`
Expected: ALL PASS (State, Endpoint, Target, EndpointHash)

- [ ] **Step 17: Run all target tests once more**

Run: `go test ./pkg/target/ -v -count=1`
Expected: ALL PASS

---

### Task 2: Update ServiceDiscovery interface and providers — add DiscoveryResult

**Files:**
- Modify: `pkg/provider/provider.go` — add `DiscoveryResult` struct, change `GetInstances()`/`Watch()` return types
- Modify: `pkg/provider/dns/discovery.go` — return `[]DiscoveryResult`
- Modify: `pkg/provider/nacos/discovery.go` — return `[]DiscoveryResult`
- Modify: `pkg/provider/k8s/k8s.go` — return `[]DiscoveryResult`
- Modify: `pkg/gateway/discovery.go` — ResolverDiscovery + StaticDiscovery return `[]DiscoveryResult`

**Breaking change**: The `ServiceDiscovery` interface changes signature. All 5 callers (upstream + 4 providers) are updated in this task.

- [ ] **Step 1: Write failing compilation test**

File: `pkg/provider/provider_test.go` (new):

```go
package provider_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nite-coder/bifrost/pkg/provider"
)

func TestDiscoveryResult_Fields(t *testing.T) {
	r := provider.DiscoveryResult{
		Target: "example.com:80",
		Weight: 100,
		Tags:   map[string]string{"region": "us"},
		Nodes:  nil,
	}
	assert.Equal(t, "example.com:80", r.Target)
	assert.Equal(t, uint32(100), r.Weight)
	assert.Equal(t, "us", r.Tags["region"])
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./pkg/provider/ -run TestDiscoveryResult -v`
Expected: FAIL — DiscoveryResult not defined

- [ ] **Step 3: Update `pkg/provider/provider.go` — add DiscoveryResult + change interface**

Add `DiscoveryResult` struct before `ServiceDiscovery`:

```go
// DiscoveryResult preserves the target→instances grouping from discovery.
type DiscoveryResult struct {
	Target string            // hostname:port (from config TargetOptions.Target, or discovery service name)
	Weight uint32            // target-level weight
	Tags   map[string]string // target-level tags
	Nodes  []Instancer       // resolved instances for this target
}
```

Change `ServiceDiscovery` interface signatures:

```go
type ServiceDiscovery interface {
	GetInstances(ctx context.Context, options GetInstanceOptions) ([]DiscoveryResult, error)
	Watch(ctx context.Context, options GetInstanceOptions) (<-chan []DiscoveryResult, error)
	Close() error
}
```

- [ ] **Step 4: Run test, expect pass**

Run: `go test ./pkg/provider/ -run TestDiscoveryResult -v`
Expected: PASS

- [ ] **Step 5: Verify compilation fails everywhere else**

Run: `go build ./pkg/...`
Expected: FAIL — DNS provider, ResolverDiscovery, StaticDiscovery, Upstream all use old signature

- [ ] **Step 6: Update DNS provider**

File: `pkg/provider/dns/discovery.go`

Change `GetInstances()` return type and wrap results:

```go
func (d *Discovery) GetInstances(
	ctx context.Context,
	options provider.GetInstanceOptions,
) ([]provider.DiscoveryResult, error) {
	instances := make([]provider.Instancer, 0)
	targetHost, targetPort, err := net.SplitHostPort(options.Name)
	if err != nil {
		targetHost = options.Name
	}
	ips, err := d.Lookup(ctx, targetHost)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup target '%s', error: %w", targetHost, err)
	}
	for _, ip := range ips {
		if len(targetPort) > 0 {
			ip = net.JoinHostPort(ip, targetPort)
		} else {
			ip = net.JoinHostPort(ip, "0")
		}
		addr, err := net.ResolveTCPAddr("tcp", ip)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve target '%s', error: %w", ip, err)
		}
		instance := provider.NewInstance(addr, 1)
		instance.SetTag("server_name", targetHost)
		instances = append(instances, instance)
	}
	return []provider.DiscoveryResult{
		{
			Target: options.Name,
			Weight: 0,
			Nodes:  instances,
		},
	}, nil
}
```

Change `Watch()` return type:

```go
func (d *Discovery) Watch(
	ctx context.Context,
	_ provider.GetInstanceOptions,
) (<-chan []provider.DiscoveryResult, error) {
	ch := make(chan []provider.DiscoveryResult, 1)
	go safety.Go(ctx, func() {
		defer close(ch)
		if d.ticker != nil {
			for {
				select {
				case <-d.ticker.C:
					ch <- nil
				case <-ctx.Done():
					return
				}
			}
		}
	})
	return ch, nil
}
```

- [ ] **Step 7: Verify DNS provider compiles**

Run: `go build ./pkg/provider/dns/...`
Expected: PASS

- [ ] **Step 8: Update Nacos provider**

File: `pkg/provider/nacos/discovery.go`

Change `GetInstances()` return type — wrap all nodes in a single `DiscoveryResult`:

```go
func (d *Discovery) GetInstances(
	_ context.Context,
	options provider.GetInstanceOptions,
) ([]provider.DiscoveryResult, error) {
	// ... existing Nacos query logic (unchanged) ...
	// At the return point, wrap instead of returning flat:
	return []provider.DiscoveryResult{
		{
			Target: options.Name,
			Nodes:  instances,
		},
	}, nil
}
```

For `Watch()` — **do not send nil**. Instead, wrap the instances pushed by Nacos SDK into `DiscoveryResult` and send on the channel. This avoids an extra `GetInstances()` API call on every notification:

```go
func (d *Discovery) Watch(
	ctx context.Context,
	_ provider.GetInstanceOptions,
) (<-chan []provider.DiscoveryResult, error) {
	ch := make(chan []provider.DiscoveryResult, 1)
	// ... existing Watch logic ...
	// In the Nacos SubscribeCallback, wrap instances:
	//   instances := ToProviderInstance(nacosInstances)
	//   ch <- []provider.DiscoveryResult{{Target: options.Name, Nodes: instances}}
	return ch, nil
}
```

- [ ] **Step 9: Verify Nacos provider compiles**

Run: `go build ./pkg/provider/nacos/...`
Expected: PASS

- [ ] **Step 10: Update K8s provider**

File: `pkg/provider/k8s/k8s.go`

Same pattern — change both `GetInstances()` and `Watch()` return types, wrap in single `DiscoveryResult`:

```go
func (k *Discovery) GetInstances(
	ctx context.Context,
	options provider.GetInstanceOptions,
) ([]provider.DiscoveryResult, error) {
	// ... existing K8s query logic (unchanged) ...
	return []provider.DiscoveryResult{
		{
			Target: options.Name,
			Nodes:  instances,
		},
	}, nil
}
```

For `Watch()` — same as Nacos: wrap pushed instances in `DiscoveryResult` instead of sending nil, to avoid an extra `GetInstances()` call on each notification:

```go
func (k *Discovery) Watch(
	ctx context.Context,
	_ provider.GetInstanceOptions,
) (<-chan []provider.DiscoveryResult, error) {
	// ... existing Watch logic ...
	// In the K8s watch callback, wrap instances:
	//   instances := ToProviderInstance(k8sEndpoints)
	//   ch <- []provider.DiscoveryResult{{Target: options.Name, Nodes: instances}}
}
```

- [ ] **Step 11: Verify K8s provider compiles**

Run: `go build ./pkg/provider/k8s/...`
Expected: PASS

- [ ] **Step 12: Update ResolverDiscovery + StaticDiscovery**

File: `pkg/gateway/discovery.go`

Change `GetInstances()` on both to return `[]provider.DiscoveryResult`:

```go
// ResolverDiscovery.GetInstances()
func (d *ResolverDiscovery) GetInstances(ctx context.Context, _ provider.GetInstanceOptions) ([]provider.DiscoveryResult, error) {
	results := make([]provider.DiscoveryResult, 0, len(d.upstream.options.Targets))
	for _, targetOption := range d.upstream.options.Targets {
		instances := make([]provider.Instancer, 0)
		targetHost, targetPort, err := net.SplitHostPort(targetOption.Target)
		if err != nil {
			targetHost = targetOption.Target
		}
		if targetPort == "" {
			targetPort = "0"
		}

		if strings.EqualFold(targetOption.Tags["type"], "service") {
			ips, lookErr := d.upstream.bifrost.resolver.LookupService(ctx, targetHost, targetPort)
			if lookErr != nil {
				return nil, fmt.Errorf("failed to resolve service '%s', error: %w", targetHost, lookErr)
			}
			for _, ip := range ips {
				addr, addrErr := net.ResolveTCPAddr("tcp", net.JoinHostPort(ip, targetPort))
				if addrErr != nil {
					return nil, fmt.Errorf("failed to resolve target '%s', error: %w", ip, addrErr)
				}
				instance := provider.NewInstance(addr, targetOption.Weight)
				instance.SetTag("server_name", targetHost)
				instances = append(instances, instance)
			}
		} else {
			ips, lookErr := d.upstream.bifrost.resolver.LookupHost(ctx, targetHost)
			if lookErr != nil {
				return nil, fmt.Errorf("failed to lookup host '%s', error: %w", targetHost, lookErr)
			}
			for _, ip := range ips {
				addr, addrErr := net.ResolveTCPAddr("tcp", net.JoinHostPort(ip, targetPort))
				if addrErr != nil {
					return nil, fmt.Errorf("failed to resolve target '%s', error: %w", ip, addrErr)
				}
				instance := provider.NewInstance(addr, targetOption.Weight)
				instance.SetTag("server_name", targetHost)
				instances = append(instances, instance)
			}
		}

		results = append(results, provider.DiscoveryResult{
			Target: targetOption.Target,
			Weight: targetOption.Weight,
			Tags:   targetOption.Tags,
			Nodes:  instances,
		})
	}
	return results, nil
}
```

Same pattern for `StaticDiscovery.GetInstances()` — iterate config targets, return one `DiscoveryResult` per target.

- [ ] **Step 13: Verify gateway compiles**

Run: `go build ./pkg/gateway/...`
Expected: FAIL — Upstream.refreshEndpoints still uses old signature

- [ ] **Step 14: Run all provider tests**

Run: `go test ./pkg/provider/... -v -count=1`
Expected: PASS

---

### Task 3: Update proxy package — use `target` types

**Files:**
- Create: (nothing new)
- Remove: `pkg/proxy/endpoint.go`
- Modify: `pkg/proxy/proxy.go` — change interface signature
- Modify: `pkg/proxy/http/proxy.go` — update import + type
- Modify: `pkg/proxy/grpc/proxy.go` — update import + type
- Modify: `pkg/proxy/ai/proxy.go` — update import + type

- [ ] **Step 1: Write failing compilation test**

The existing code imports `proxy.Endpoint`. When we remove it, compilation fails. We write a test that verifies the proxy interface accepts `*target.Endpoint`:

No new test file needed — the existing `pkg/proxy/http/proxy.go` compilation IS the test.

- [ ] **Step 2: Verify compilation fails after endpoint.go removal**

Run: `go build ./pkg/proxy/...`
Expected: FAIL — types not found (`Endpoint` undefined)

- [ ] **Step 3: Remove `pkg/proxy/endpoint.go`**

```bash
rm pkg/proxy/endpoint.go
```

- [ ] **Step 4: Update Proxy interface**

File: `pkg/proxy/proxy.go`

Change imports to add target:

```go
package proxy

import (
	"context"
	"errors"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/target"
)

var ErrMaxFailedCount = errors.New("proxy: reach max failed count")

type Proxy interface {
	ID() string
	Target() string
	Endpoint() *target.Endpoint
	SetEndpoint(ep *target.Endpoint)
	ServeHTTP(c context.Context, ctx *app.RequestContext)
	Close() error
}
```

- [ ] **Step 5: Verify compilation still fails**

Run: `go build ./pkg/proxy/...`
Expected: FAIL — implementations still use old type

- [ ] **Step 6: Update http proxy**

File: `pkg/proxy/http/proxy.go`

Change import from `"github.com/nite-coder/bifrost/pkg/proxy"` to `"github.com/nite-coder/bifrost/pkg/target"`.

Change field type:
```go
endpoint atomic.Pointer[target.Endpoint]
```

Change methods:
```go
func (p *Proxy) Endpoint() *target.Endpoint {
	return p.endpoint.Load()
}

func (p *Proxy) SetEndpoint(ep *target.Endpoint) {
	p.endpoint.Store(ep)
}
```

Change `Options` struct field:
```go
type Options struct {
	Target           string
	TargetHostHeader string
	ServiceID        string
	Protocol         config.Protocol
	IsTracingEnabled bool
	PassHostHeader   bool
	Endpoint         *target.Endpoint
}
```

Change reference in request execution (where `ep.HealthState.RecordFailure()` is called) — `HealthState` is now `State`, update to `ep.State.RecordFailure()`.

Also update import block to keep `proxy` import only if still used.

- [ ] **Step 7: Verify http proxy compiles**

Run: `go build ./pkg/proxy/http/...`
Expected: PASS (or continue to next if other proxy implementations fail)

- [ ] **Step 8: Update grpc proxy**

File: `pkg/proxy/grpc/proxy.go`

Same pattern:
- Import `"github.com/nite-coder/bifrost/pkg/target"` instead of `"github.com/nite-coder/bifrost/pkg/proxy"` for Endpoint/State types
- Change `atomic.Pointer[proxy.Endpoint]` → `atomic.Pointer[target.Endpoint]`
- Change `Options.Endpoint` type
- Change `Endpoint()` / `SetEndpoint()` return/param type
- Change `ep.HealthState.RecordFailure()` → `ep.State.RecordFailure()`
- Keep `proxy` import if other proxy types are referenced

- [ ] **Step 9: Update AI proxy**

File: `pkg/proxy/ai/proxy.go`

Same pattern:
- Import `"github.com/nite-coder/bifrost/pkg/target"`
- Change `endpoint atomic.Pointer[proxy.Endpoint]` → `endpoint atomic.Pointer[target.Endpoint]`
- Change `Endpoint()` / `SetEndpoint()` return/param type
- Change `ep.HealthState.RecordFailure()` → `ep.State.RecordFailure()` if used

- [ ] **Step 10: Verify all proxy packages compile**

Run: `go build ./pkg/proxy/...`
Expected: PASS

- [ ] **Step 11: Update proxy endpoint_test.go**

File: `pkg/proxy/endpoint_test.go` (alongside the endpoint.go we just removed — it was testing TargetState)

Since `pkg/proxy/endpoint.go` is removed and tests are now in `pkg/target/state_test.go`, delete `pkg/proxy/endpoint_test.go`:

```bash
rm pkg/proxy/endpoint_test.go
```

- [ ] **Step 12: Verify all proxy packages compile and tests pass**

Run: `go test ./pkg/proxy/... -v -count=1`
Expected: PASS

---

### Task 4: Refactor balancer interface and implementations

Balancer now operates on `[]*target.Endpoint` instead of `[]proxy.Proxy`.

**Files:**
- Modify: `pkg/balancer/pkg.go`
- Modify: `pkg/balancer/roundrobin/round_robin.go` + test
- Modify: `pkg/balancer/weighted/weighted.go` + test
- Modify: `pkg/balancer/random/random.go` + test
- Modify: `pkg/balancer/chash/hashing.go` + test

- [ ] **Step 1: Write failing test for new Balancer interface**

File: `pkg/balancer/pkg_test.go` (create new file alongside pkg.go)

```go
package balancer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/target"
)

func TestFactory_CompileCheck(t *testing.T) {
	// This is more of a compile-time check that the interface changed correctly.
	// We'll test round_robin which is the default.
	factory := balancer.Factory("round_robin")
	assert.NotNil(t, factory, "round_robin should be registered")

	endpoints := []*target.Endpoint{
		{Address: "10.0.1.1:80", Weight: 1, State: target.NewState(0, 0)},
		{Address: "10.0.1.2:80", Weight: 1, State: target.NewState(0, 0)},
	}
	b, err := factory(endpoints, nil)
	assert.NoError(t, err)
	assert.NotNil(t, b)

	ep, err := b.Select(context.Background(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, ep)
	assert.Contains(t, []string{"10.0.1.1:80", "10.0.1.2:80"}, ep.Address)
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./pkg/balancer/ -run TestFactory_CompileCheck -v`
Expected: FAIL — interface mismatch

- [ ] **Step 3: Update balancer/pkg.go**

```go
package balancer

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/target"
)

var (
	ErrNotAvailable = errors.New("no available upstream at the moment")
	mu              sync.RWMutex
	balancers       = make(map[string]CreateBalancerHandler)
)

type CreateBalancerHandler func(endpoints []*target.Endpoint, params any) (Balancer, error)

type Balancer interface {
	Select(ctx context.Context, hzCtx *app.RequestContext) (*target.Endpoint, error)
}

func Register(names []string, h CreateBalancerHandler) error {
	if len(names) == 0 {
		return errors.New("balancer names cannot be empty")
	}
	mu.Lock()
	defer mu.Unlock()
	for _, name := range names {
		if _, found := balancers[name]; found {
			return fmt.Errorf("balancer '%s' already exists", name)
		}
		balancers[name] = h
	}
	return nil
}

func Factory(name string) CreateBalancerHandler {
	if name == "" {
		name = "round_robin"
	}
	mu.RLock()
	defer mu.RUnlock()
	return balancers[name]
}
```

- [ ] **Step 4: Run test, expect still fails (round_robin init still uses old signature)**

Run: `go test ./pkg/balancer/ -run TestFactory_CompileCheck -v`
Expected: FAIL — round_robin Init() still expects `[]proxy.Proxy`

- [ ] **Step 5: Rewrite round_robin balancer with TDD**

First, write the failing round_robin test (`pkg/balancer/roundrobin/round_robin_test.go`):

```go
package roundrobin_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/target"
)

func createTestEndpoint(addr string, weight uint32, maxFails uint, failTimeout time.Duration) *target.Endpoint {
	return &target.Endpoint{
		Address: addr,
		Weight:  weight,
		State:   target.NewState(maxFails, failTimeout),
	}
}

func TestRoundRobin(t *testing.T) {
	_ = roundrobin.Init()

	t.Run("success", func(t *testing.T) {
		eps := []*target.Endpoint{
			createTestEndpoint("10.0.1.1:80", 1, 1, time.Second),
			createTestEndpoint("10.0.1.2:80", 1, 1, 10*time.Second),
			createTestEndpoint("10.0.1.3:80", 1, 0, 10*time.Second),
		}
		b := roundrobin.NewBalancer(eps)

		expected := []string{"10.0.1.1:80", "10.0.1.2:80", "10.0.1.3:80"}
		for _, e := range expected {
			ep, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			require.NotNil(t, ep)
			assert.Equal(t, e, ep.Address)
		}
	})

	t.Run("one endpoint failed", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80", 1, 1, time.Second)
		ep2 := createTestEndpoint("10.0.1.2:80", 1, 1, 10*time.Second)
		ep3 := createTestEndpoint("10.0.1.3:80", 1, 0, 10*time.Second)

		ep1.State.RecordFailure() // will recover after 1s
		ep2.State.RecordFailure()
		// ep3 has maxFails=0, always available

		eps := []*target.Endpoint{ep1, ep2, ep3}
		b := roundrobin.NewBalancer(eps)

		// ep1 is down, ep2 is down, ep3 is up → only ep3 returned
		for range 5 {
			ep, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			assert.Equal(t, "10.0.1.3:80", ep.Address, "only ep3 should be selected")
		}
	})

	t.Run("no live endpoint", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80", 1, 1, 10*time.Second)
		ep1.State.RecordFailure()

		b := roundrobin.NewBalancer([]*target.Endpoint{ep1})
		ep, err := b.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		require.Nil(t, ep)
	})

	t.Run("nil endpoints", func(t *testing.T) {
		b := roundrobin.NewBalancer(nil)
		ep, err := b.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})

	t.Run("counter overflow", func(t *testing.T) {
		eps := []*target.Endpoint{
			createTestEndpoint("10.0.1.1:80", 1, 0, 0),
			createTestEndpoint("10.0.1.2:80", 1, 0, 0),
		}
		b := roundrobin.NewBalancer(eps)
		b.counter.Store(math.MaxUint64 - 1)

		ep, err := b.Select(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "10.0.1.1:80", ep.Address)
	})
}
```

- [ ] **Step 6: Run round_robin test, expect failure**

Run: `go test ./pkg/balancer/roundrobin/ -v -count=1`
Expected: FAIL — Init/NewBalancer still use old types

- [ ] **Step 7: Rewrite round_robin implementation**

File: `pkg/balancer/roundrobin/round_robin.go`

```go
package roundrobin

import (
	"context"
	"sync/atomic"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/target"
)

func Init() error {
	return balancer.Register(
		[]string{"round_robin"},
		func(endpoints []*target.Endpoint, _ any) (balancer.Balancer, error) {
			b := NewBalancer(endpoints)
			return b, nil
		},
	)
}

type Balancer struct {
	counter   atomic.Uint64
	endpoints []*target.Endpoint
}

func NewBalancer(endpoints []*target.Endpoint) *Balancer {
	return &Balancer{
		endpoints: endpoints,
	}
}

func (b *Balancer) Select(_ context.Context, _ *app.RequestContext) (*target.Endpoint, error) {
	if len(b.endpoints) == 0 {
		return nil, balancer.ErrNotAvailable
	}

	if len(b.endpoints) == 1 {
		ep := b.endpoints[0]
		if ep.State != nil && ep.State.IsAvailable() {
			return ep, nil
		}
		return nil, balancer.ErrNotAvailable
	}

	// round-robin with skip for unhealthy endpoints
	failedRecords := make(map[string]bool)
	for {
		count := b.counter.Add(1)
		index := (count - 1) % uint64(len(b.endpoints))
		ep := b.endpoints[index]

		if ep.State == nil || ep.State.IsAvailable() {
			return ep, nil
		}
		if len(failedRecords) == len(b.endpoints) {
			return nil, balancer.ErrNotAvailable
		}
		failedRecords[ep.Address] = true
	}
}
```

- [ ] **Step 8: Run round_robin test, expect pass**

Run: `go test ./pkg/balancer/roundrobin/ -v -count=1`
Expected: PASS

- [ ] **Step 9: Rewrite weighted balancer with TDD**

File: `pkg/balancer/weighted/weighted_test.go`:

```go
package weighted_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/target"
)

func createTestEndpoint(addr string, weight uint32, maxFails uint, failTimeout time.Duration) *target.Endpoint {
	return &target.Endpoint{
		Address: addr,
		Weight:  weight,
		State:   target.NewState(maxFails, failTimeout),
	}
}

func TestWeighted(t *testing.T) {
	_ = weighted.Init()

	t.Run("success", func(t *testing.T) {
		eps := []*target.Endpoint{
			createTestEndpoint("10.0.1.1:80", 1, 10, 10*time.Second),
			createTestEndpoint("10.0.1.2:80", 2, 1, 10*time.Second),
			createTestEndpoint("10.0.1.3:80", 3, 100, 10*time.Second),
		}
		b, err := weighted.NewBalancer(eps)
		require.NoError(t, err)

		hits := map[string]int{}
		for range 6000 {
			ep, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			hits[ep.Address]++
		}
		assert.InDelta(t, 1000, hits["10.0.1.1:80"], 100)
		assert.InDelta(t, 2000, hits["10.0.1.2:80"], 100)
		assert.InDelta(t, 3000, hits["10.0.1.3:80"], 100)
	})

	t.Run("one endpoint failed", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80", 1, 10, 10*time.Second)
		ep2 := createTestEndpoint("10.0.1.2:80", 2, 1, 10*time.Second)
		ep3 := createTestEndpoint("10.0.1.3:80", 3, 100, 10*time.Second)
		for range 10 { ep1.State.RecordFailure() }
		eps := []*target.Endpoint{ep1, ep2, ep3}

		b, err := weighted.NewBalancer(eps)
		require.NoError(t, err)

		hits := map[string]int{}
		for range 6000 {
			ep, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			hits[ep.Address]++
		}
		assert.Equal(t, 0, hits["10.0.1.1:80"])
		assert.InDelta(t, 2400, hits["10.0.1.2:80"], 150)
		assert.InDelta(t, 3600, hits["10.0.1.3:80"], 150)
	})

	t.Run("no live endpoint", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80", 1, 1, 0)
		ep1.State.RecordFailure()
		b, err := weighted.NewBalancer([]*target.Endpoint{ep1})
		require.NoError(t, err)

		ep, err := b.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})

	t.Run("nil endpoints", func(t *testing.T) {
		b, err := weighted.NewBalancer(nil)
		require.NoError(t, err)

		ep, err := b.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})

	t.Run("registration", func(t *testing.T) {
		factory := balancer.Factory("weighted")
		require.NotNil(t, factory)
		ep := createTestEndpoint("10.0.1.1:80", 1, 0, 0)
		b, err := factory([]*target.Endpoint{ep}, nil)
		require.NoError(t, err)
		require.NotNil(t, b)

		epOut, err := b.Select(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "10.0.1.1:80", epOut.Address)
	})
}
```

- [ ] **Step 10: Run test, expect failure**

Run: `go test ./pkg/balancer/weighted/ -v -count=1`
Expected: FAIL

- [ ] **Step 11: Rewrite weighted implementation**

File: `pkg/balancer/weighted/weighted.go`

Full rewrite to use `[]*target.Endpoint`:

```go
package weighted

import (
	"context"
	"math"
	"math/rand"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/target"
)

func Init() error {
	return balancer.Register(
		[]string{"weighted"},
		func(endpoints []*target.Endpoint, _ any) (balancer.Balancer, error) {
			return NewBalancer(endpoints)
		},
	)
}

type Balancer struct {
	mu          sync.Mutex
	endpoints   []*target.Endpoint
	totalWeight uint32
}

func NewBalancer(endpoints []*target.Endpoint) (*Balancer, error) {
	b := &Balancer{}
	b.reset(endpoints)
	return b, nil
}

func (b *Balancer) reset(endpoints []*target.Endpoint) {
	var totalWeight uint32
	for _, ep := range endpoints {
		w := ep.Weight
		if w > math.MaxInt32 {
			w = math.MaxInt32
		}
		if w == 0 {
			w = 1
		}
		totalWeight += w
	}
	b.endpoints = endpoints
	b.totalWeight = totalWeight
}

func (b *Balancer) Select(_ context.Context, _ *app.RequestContext) (*target.Endpoint, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.endpoints) == 0 {
		return nil, balancer.ErrNotAvailable
	}

	r := rand.Intn(int(b.totalWeight)) + 1
	for _, ep := range b.endpoints {
		if ep.State != nil && !ep.State.IsAvailable() {
			continue
		}
		w := int(ep.Weight)
		if w == 0 {
			w = 1
		}
		r -= w
		if r <= 0 {
			return ep, nil
		}
	}
	return nil, balancer.ErrNotAvailable
}
```

- [ ] **Step 12: Run test, expect pass**

Run: `go test ./pkg/balancer/weighted/ -v -count=1`
Expected: PASS

- [ ] **Step 13: Rewrite random balancer (test + implementation)**

File: `pkg/balancer/random/random_test.go`:

```go
package random_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/target"
)

func createTestEndpoint(addr string, weight uint32, maxFails uint, failTimeout time.Duration) *target.Endpoint {
	return &target.Endpoint{
		Address: addr,
		Weight:  weight,
		State:   target.NewState(maxFails, failTimeout),
	}
}

func TestRandom(t *testing.T) {
	_ = random.Init()

	t.Run("success", func(t *testing.T) {
		eps := []*target.Endpoint{
			createTestEndpoint("10.0.1.1:80", 1, 1, 10*time.Second),
			createTestEndpoint("10.0.1.2:80", 1, 1, 10*time.Second),
			createTestEndpoint("10.0.1.3:80", 1, 1, 10*time.Second),
		}
		b := random.NewBalancer(eps)

		hits := map[string]int{}
		for range 10000 {
			ep, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			hits[ep.Address]++
		}
		assert.InDelta(t, 3333, hits["10.0.1.1:80"], 500)
		assert.InDelta(t, 3333, hits["10.0.1.2:80"], 500)
		assert.InDelta(t, 3333, hits["10.0.1.3:80"], 500)
	})

	t.Run("two endpoint failed", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80", 1, 1, 10*time.Second)
		ep2 := createTestEndpoint("10.0.1.2:80", 1, 1, 10*time.Second)
		ep3 := createTestEndpoint("10.0.1.3:80", 1, 1, 10*time.Second)
		ep1.State.RecordFailure()
		ep2.State.RecordFailure()

		b := random.NewBalancer([]*target.Endpoint{ep1, ep2, ep3})

		for range 100 {
			ep, err := b.Select(context.Background(), nil)
			require.NoError(t, err)
			assert.Equal(t, "10.0.1.3:80", ep.Address)
		}
	})

	t.Run("no live endpoint", func(t *testing.T) {
		ep1 := createTestEndpoint("10.0.1.1:80", 1, 1, 0)
		ep1.State.RecordFailure()
		b := random.NewBalancer([]*target.Endpoint{ep1})
		ep, err := b.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})

	t.Run("nil endpoints", func(t *testing.T) {
		b := random.NewBalancer(nil)
		ep, err := b.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})

	t.Run("registration", func(t *testing.T) {
		factory := balancer.Factory("random")
		require.NotNil(t, factory)
		ep := createTestEndpoint("10.0.1.1:80", 1, 0, 0)
		b, err := factory([]*target.Endpoint{ep}, nil)
		require.NoError(t, err)
		require.NotNil(t, b)
		epOut, err := b.Select(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "10.0.1.1:80", epOut.Address)
	})
}
```

File: `pkg/balancer/random/random.go`:

```go
package random

import (
	"context"
	"math/rand"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/target"
)

func Init() error {
	return balancer.Register(
		[]string{"random"},
		func(endpoints []*target.Endpoint, _ any) (balancer.Balancer, error) {
			b := NewBalancer(endpoints)
			return b, nil
		},
	)
}

type Balancer struct {
	endpoints []*target.Endpoint
}

func NewBalancer(endpoints []*target.Endpoint) *Balancer {
	return &Balancer{
		endpoints: endpoints,
	}
}

func (b *Balancer) Select(_ context.Context, _ *app.RequestContext) (*target.Endpoint, error) {
	if len(b.endpoints) == 0 {
		return nil, balancer.ErrNotAvailable
	}

	// random selection with skip for unhealthy endpoints
	offset := rand.Intn(len(b.endpoints))
	for i := range b.endpoints {
		idx := (offset + i) % len(b.endpoints)
		ep := b.endpoints[idx]
		if ep.State == nil || ep.State.IsAvailable() {
			return ep, nil
		}
	}
	return nil, balancer.ErrNotAvailable
}
```

- [ ] **Step 14: Run random test, expect pass**

Run: `go test ./pkg/balancer/random/ -v -count=1`
Expected: PASS

- [ ] **Step 15: Rewrite chash balancer (test + implementation)**

File: `pkg/balancer/chash/hashing_test.go`:

```go
package chash_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/target"
)

func createTestEndpoint(addr string, weight uint32, maxFails uint, failTimeout time.Duration) *target.Endpoint {
	return &target.Endpoint{
		Address: addr,
		Weight:  weight,
		State:   target.NewState(maxFails, failTimeout),
	}
}

func TestHashing(t *testing.T) {
	_ = chash.Init()

	ep1 := createTestEndpoint("10.0.1.1:80", 1, 1, 10*time.Minute)
	ep2 := createTestEndpoint("10.0.1.2:80", 1, 1, 10*time.Minute)
	ep3 := createTestEndpoint("10.0.1.3:80", 1, 1, 10*time.Minute)
	eps := []*target.Endpoint{ep1, ep2, ep3}

	t.Run("success", func(t *testing.T) {
		keys := []string{"key1", "key2", "key3"}
		for _, key := range keys {
			params := map[string]any{"hash_on": "$var.uid"}
			b, err := chash.NewBalancer(eps, params)
			require.NoError(t, err)

			hzctx := app.NewContext(0)
			hzctx.Set("uid", key)

			epOut1, err := b.Select(context.Background(), hzctx)
			require.NoError(t, err)
			epOut2, err := b.Select(context.Background(), hzctx)
			require.NoError(t, err)
			assert.Equal(t, epOut1.Address, epOut2.Address, "same key → same endpoint")
		}
	})

	t.Run("two endpoints failed", func(t *testing.T) {
		ep1.State.RecordFailure()
		ep2.State.RecordFailure()

		params := map[string]any{"hash_on": "$var.uid"}
		for _, key := range []string{"key1", "key2", "key3"} {
			b, err := chash.NewBalancer(eps, params)
			require.NoError(t, err)

			hzctx := app.NewContext(0)
			hzctx.Set("uid", key)
			ep, err := b.Select(context.Background(), hzctx)
			require.NoError(t, err)
			assert.Equal(t, "10.0.1.3:80", ep.Address)
		}
	})

	t.Run("no live endpoint", func(t *testing.T) {
		ep1.State.RecordFailure()
		ep2.State.RecordFailure()
		ep3.State.RecordFailure()

		params := map[string]any{"hash_on": "$var.uid"}
		b, err := chash.NewBalancer(eps, params)
		require.NoError(t, err)

		hzctx := app.NewContext(0)
		hzctx.Set("uid", "test")
		ep, err := b.Select(context.Background(), hzctx)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})

	t.Run("registration error paths", func(t *testing.T) {
		factory := balancer.Factory("hashing")
		require.NotNil(t, factory)

		// nil params
		b, err := factory(eps, nil)
		require.Error(t, err)
		assert.Nil(t, b)

		// invalid hash_on type
		b, err = factory(eps, map[string]any{"hash_on": 123})
		require.Error(t, err)
		assert.Nil(t, b)

		// missing hash_on
		b, err = factory(eps, map[string]any{"other": "val"})
		require.Error(t, err)
		assert.Nil(t, b)
	})

	t.Run("nil endpoints", func(t *testing.T) {
		b, err := chash.NewBalancer(nil, map[string]any{"hash_on": "$var.uid"})
		require.NoError(t, err)
		ep, err := b.Select(context.Background(), nil)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})

	t.Run("single endpoint failed", func(t *testing.T) {
		e := createTestEndpoint("10.0.1.1:80", 1, 1, 10*time.Minute)
		e.State.RecordFailure()
		b, err := chash.NewBalancer([]*target.Endpoint{e}, map[string]any{"hash_on": "$var.uid"})
		require.NoError(t, err)

		hzctx := app.NewContext(0)
		hzctx.Set("uid", "key")
		ep, err := b.Select(context.Background(), hzctx)
		require.ErrorIs(t, err, balancer.ErrNotAvailable)
		assert.Nil(t, ep)
	})
}
```

File: `pkg/balancer/chash/hashing.go`:

```go
package chash

import (
	"context"
	"errors"
	"slices"

	"github.com/cloudwego/hertz/pkg/app"

	"github.com/nite-coder/bifrost/internal/pkg/consistent"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/target"
	"github.com/nite-coder/bifrost/pkg/variable"
)

const defaultReplicas = 160

func Init() error {
	return balancer.Register(
		[]string{"hashing", "chash"},
		func(endpoints []*target.Endpoint, params any) (balancer.Balancer, error) {
			if params == nil {
				return nil, errors.New("params cannot be empty")
			}
			parsed, ok := params.(map[string]any)
			if !ok {
				return nil, errors.New("params must be a map")
			}
			hashon, ok := parsed["hash_on"].(string)
			if !ok {
				return nil, errors.New("hash_on is required and must be a string")
			}
			return NewBalancer(endpoints, parsed)
		},
	)
}

type Balancer struct {
	hashon  string
	ring    *consistent.Consistent
	nodeMap map[string]*target.Endpoint
}

func NewBalancer(endpoints []*target.Endpoint, params map[string]any) (*Balancer, error) {
	hashon, ok := params["hash_on"].(string)
	if !ok {
		return nil, errors.New("hash_on is required and must be a string")
	}
	replicas := defaultReplicas
	b := &Balancer{
		hashon:  hashon,
		ring:    consistent.New().SetReplicas(replicas),
		nodeMap: make(map[string]*target.Endpoint),
	}
	sorted := make([]*target.Endpoint, len(endpoints))
	copy(sorted, endpoints)
	slices.SortFunc(sorted, func(a, b *target.Endpoint) int {
		if a.Address < b.Address {
			return -1
		}
		if a.Address > b.Address {
			return 1
		}
		return 0
	})
	for _, ep := range sorted {
		weight := 1
		if ep.Weight > 0 {
			weight = int(ep.Weight)
		}
		_ = b.ring.AddWithReplicas(ep.Address, replicas*weight)
		b.nodeMap[ep.Address] = ep
	}
	return b, nil
}

func (b *Balancer) Select(_ context.Context, c *app.RequestContext) (*target.Endpoint, error) {
	if len(b.nodeMap) == 0 {
		return nil, balancer.ErrNotAvailable
	}
	val := variable.GetString(b.hashon, c)
	candidates, err := b.ring.GetN(val, len(b.nodeMap))
	if err != nil {
		return nil, balancer.ErrNotAvailable
	}
	for _, nodeID := range candidates {
		ep, ok := b.nodeMap[nodeID]
		if ok && (ep.State == nil || ep.State.IsAvailable()) {
			return ep, nil
		}
	}
	return nil, balancer.ErrNotAvailable
}
```

- [ ] **Step 16: Run chash test, expect pass**

Run: `go test ./pkg/balancer/chash/ -v -count=1`
Expected: PASS

- [ ] **Step 17: Run all balancer tests**

Run: `go test ./pkg/balancer/... -v -count=1`
Expected: ALL PASS

---

### Task 5: Update Upstream — hold balancer + targets map with Target grouping

**Files:**
- Modify: `pkg/gateway/upstream.go`
- Modify: `pkg/gateway/upstream_test.go`

- [ ] **Step 1: Write failing test for upstream balancer access**

File: `pkg/gateway/upstream_test.go` — add a new test:

```go
func TestUpstream_HoldsBalancer(t *testing.T) {
	dnsResolver, err := resolver.NewResolver(resolver.Options{})
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
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./pkg/gateway/ -run TestUpstream_HoldsBalancer -v`
Expected: FAIL — Upstream doesn't have Balancer() method

- [ ] **Step 3: Update upstream.go**

Full rewrite of `pkg/gateway/upstream.go`:

```go
package gateway

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net"
	"strings"
	"sync"

	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/balancer"
	_ "github.com/nite-coder/bifrost/pkg/balancer/roundrobin"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/provider"
	"github.com/nite-coder/bifrost/pkg/provider/dns"
	"github.com/nite-coder/bifrost/pkg/provider/k8s"
	"github.com/nite-coder/bifrost/pkg/provider/nacos"
	"github.com/nite-coder/bifrost/pkg/target"
)

const defaultSubscriberBufferSize = 64

type Upstream struct {
	mu            sync.RWMutex
	discovery     provider.ServiceDiscovery
	bifrost       *Bifrost
	options       *config.UpstreamOptions
	subscribers   []chan []*target.Endpoint
	targets       map[string]*target.Target  // keyed by target name (hostname:port)
	endpointsHash string                     // hash of flat endpoint list for change detection
	balancer      balancer.Balancer
	watchOnce     sync.Once
	cancel        context.CancelFunc
	isExclusive   bool
}

func (u *Upstream) Close() error {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.cancel != nil {
		u.cancel()
	}
	for _, sub := range u.subscribers {
		close(sub)
	}
	u.subscribers = nil
	if u.discovery != nil {
		return u.discovery.Close()
	}
	return nil
}

func (u *Upstream) Balancer() balancer.Balancer {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.balancer
}

func newUpstream(bifrost *Bifrost, upstreamOptions config.UpstreamOptions) (*Upstream, error) {
	var err error
	if len(upstreamOptions.ID) == 0 {
		return nil, errors.New("upstream ID cannot be empty")
	}
	if upstreamOptions.Discovery.Type == "" && len(upstreamOptions.Targets) == 0 {
		return nil, fmt.Errorf("targets cannot be empty for upstream ID: %s", upstreamOptions.ID)
	}

	upstream := &Upstream{
		bifrost:      bifrost,
		options:      &upstreamOptions,
		targets:      make(map[string]*target.Target),
	}

	// Pre-populate targets from config
	for _, tgtOpt := range upstreamOptions.Targets {
		upstream.targets[tgtOpt.Target] = &target.Target{
			Name:      tgtOpt.Target,
			Weight:    tgtOpt.Weight,
			Tags:      tgtOpt.Tags,
			Endpoints: make(map[string]*target.Endpoint),
		}
	}

	if strings.HasPrefix(upstreamOptions.ID, "ai:") {
		upstream.discovery = NewStaticDiscovery(upstream)
	} else {
		switch strings.ToLower(upstreamOptions.Discovery.Type) {
		case "dns":
			if !bifrost.options.Providers.DNS.Enabled {
				return nil, fmt.Errorf("dns provider is disabled for upstream ID: %s", upstreamOptions.ID)
			}
			discovery, dErr := dns.NewDNSServiceDiscovery(
				bifrost.options.Providers.DNS.Servers,
				bifrost.options.Providers.DNS.Valid,
			)
			if dErr != nil {
				return nil, dErr
			}
			upstream.discovery = discovery
		case "nacos":
			if !bifrost.options.Providers.Nacos.Discovery.Enabled {
				return nil, fmt.Errorf("nacos discovery provider is disabled for upstream ID: %s", upstreamOptions.ID)
			}
			opts := nacos.Options{
				Username:    bifrost.options.Providers.Nacos.Discovery.Username,
				Password:    bifrost.options.Providers.Nacos.Discovery.Password,
				NamespaceID: bifrost.options.Providers.Nacos.Discovery.NamespaceID,
				Prefix:      bifrost.options.Providers.Nacos.Discovery.Prefix,
				CacheDir:    bifrost.options.Providers.Nacos.Discovery.CacheDir,
				Endpoints:   bifrost.options.Providers.Nacos.Discovery.Endpoints,
				LogDir:      bifrost.options.Providers.Nacos.Discovery.LogDir,
				LogLevel:    bifrost.options.Providers.Nacos.Discovery.LogLevel,
			}
			var discovery provider.ServiceDiscovery
			discovery, err = nacos.NewNacosServiceDiscovery(opts)
			if err != nil {
				return nil, err
			}
			upstream.discovery = discovery
		case "k8s":
			if !bifrost.options.Providers.K8S.Enabled {
				return nil, fmt.Errorf("k8s provider is disabled for upstream ID: %s", upstreamOptions.ID)
			}
			option := k8s.Options{
				APIServer: bifrost.options.Providers.K8S.APIServer,
			}
			var discovery provider.ServiceDiscovery
			discovery, err = k8s.NewK8sDiscovery(option)
			if err != nil {
				return nil, err
			}
			upstream.discovery = discovery
		default:
			upstream.discovery = NewResolverDiscovery(upstream)
		}
	}

	if err = upstream.refreshEndpoints(nil); err != nil {
		return nil, err
	}
	return upstream, nil
}

// Endpoints returns the current flat endpoint list synchronously.
// Reliable — no timing dependency on channel buffer.
func (u *Upstream) Endpoints() []*target.Endpoint {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return u.flattenEndpoints()
}

// Subscribe registers for ongoing updates. Does NOT send initial state.
// Caller must call Endpoints() first for initial data.
func (u *Upstream) Subscribe() <-chan []*target.Endpoint {
	u.watch()
	u.mu.Lock()
	defer u.mu.Unlock()
	ch := make(chan []*target.Endpoint, defaultSubscriberBufferSize)
	u.subscribers = append(u.subscribers, ch)
	return ch
}

func (u *Upstream) Unsubscribe(ch <-chan []*target.Endpoint) {
	u.mu.Lock()
	defer u.mu.Unlock()
	for i, sub := range u.subscribers {
		if sub == ch {
			u.subscribers = append(u.subscribers[:i], u.subscribers[i+1:]...)
			close(sub)
			break
		}
	}
}

// flattenEndpoints iterates all targets and collects their endpoints into a flat slice.
func (u *Upstream) flattenEndpoints() []*target.Endpoint {
	var total int
	for _, t := range u.targets {
		total += len(t.Endpoints)
	}
	slice := make([]*target.Endpoint, 0, total)
	for _, t := range u.targets {
		for _, ep := range t.Endpoints {
			slice = append(slice, ep)
		}
	}
	return slice
}

func (u *Upstream) refreshEndpoints(results []provider.DiscoveryResult) error {
	var err error
	if len(results) == 0 && u.discovery != nil {
		opts := provider.GetInstanceOptions{
			Namespace: u.options.Discovery.Namespace,
			Name:      u.options.Discovery.Name,
		}
		results, err = u.discovery.GetInstances(context.Background(), opts)
		if err != nil {
			return err
		}
	} else if len(results) == 0 {
		return fmt.Errorf("no instances found for upstream ID: %s", u.options.ID)
	}

	maxFails := u.bifrost.options.Default.Upstream.MaxFails
	if u.options.HealthCheck.Passive.MaxFails != nil {
		maxFails = *u.options.HealthCheck.Passive.MaxFails
	}
	failTimeout := u.options.HealthCheck.Passive.FailTimeout
	if failTimeout <= 0 {
		failTimeout = u.bifrost.options.Default.Upstream.FailTimeout
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	// Results are already grouped by target from discovery. Iterate directly.
	// Endpoint map update is inlined here (cannot be a method on pkg/target.Target
	// since this code is in pkg/gateway). Preserves State pointers across refreshes.
	seen := make(map[string]bool, len(results))
	for _, r := range results {
		tgt := u.getOrCreateTarget(r.Target)
		tgt.Weight = r.Weight
		tgt.Tags = r.Tags
		seen[r.Target] = true

		newMap := make(map[string]*target.Endpoint, len(r.Nodes))
		for _, inst := range r.Nodes {
			var address string
			if inst.Address().Network() == "static" {
				address = inst.Address().String()
			} else {
				targetHost, targetPort, splitErr := net.SplitHostPort(inst.Address().String())
				if splitErr != nil {
					targetHost = inst.Address().String()
					targetPort = "0"
				}
				address = net.JoinHostPort(targetHost, targetPort)
			}

			if existing, found := tgt.Endpoints[address]; found {
				existing.Weight = inst.Weight()
				existing.Tags = inst.Tags()
				newMap[address] = existing
			} else {
				state := target.NewState(maxFails, failTimeout)
				tags := make(map[string]string)
				maps.Copy(tags, inst.Tags())
				ep := &target.Endpoint{
					Address: address,
					Weight:  inst.Weight(),
					Tags:    tags,
					State:   state,
				}
				newMap[address] = ep
			}
		}
		tgt.Endpoints = newMap
	}

	// Clear endpoints from targets not present in the new results.
	for name, tgt := range u.targets {
		if !seen[name] {
			tgt.Endpoints = make(map[string]*target.Endpoint)
		}
	}

	// Build flat list for balancer + hash.
	flat := u.flattenEndpoints()
	newHash := target.EndpointHash(flat)
	if newHash == u.endpointsHash {
		return nil
	}
	u.endpointsHash = newHash

	u.rebuildBalancer(flat)

	for _, ch := range u.subscribers {
		select {
		case ch <- flat:
		default:
			slog.Warn("upstream subscriber channel full, dropping update", "upstream_id", u.options.ID)
		}
	}
	return nil
}

func (u *Upstream) rebuildBalancer(endpoints []*target.Endpoint) {
	factory := balancer.Factory(u.options.Balancer.Type)
	if factory == nil {
		slog.Error("unsupported balancer type", "type", u.options.Balancer.Type)
		return
	}
	b, err := factory(endpoints, u.options.Balancer.Params)
	if err != nil {
		slog.Error("failed to create balancer", "upstream_id", u.options.ID, "error", err)
		return
	}
	u.balancer = b
}

// getOrCreateTarget returns an existing target by name or creates a new one.
// This handles dynamic discovery targets not pre-populated from config.
func (u *Upstream) getOrCreateTarget(name string) *target.Target {
	if t, ok := u.targets[name]; ok {
		return t
	}
	t := &target.Target{
		Name:      name,
		Endpoints: make(map[string]*target.Endpoint),
	}
	u.targets[name] = t
	return t
}

func (u *Upstream) watch() {
	u.watchOnce.Do(func() {
		if u.discovery == nil {
			return
		}
		opts := provider.GetInstanceOptions{
			Name: u.options.Discovery.Name,
		}
		ctx, cancel := context.WithCancel(context.Background())
		u.cancel = cancel
		watchCh, err := u.discovery.Watch(ctx, opts)
		if err != nil {
			if errors.Is(err, provider.ErrWatchNotSupported) {
				return
			}
			slog.Error("failed to watch upstream", "error", err.Error(), "upstream_id", u.options.ID)
			return
		}
		if watchCh == nil {
			return
		}
		go safety.Go(ctx, func() {
			for results := range watchCh {
				if rErr := u.refreshEndpoints(results); rErr != nil {
					slog.Warn("upstream refresh failed", "error", rErr.Error(), "upstream_id", u.options.ID)
				}
			}
		})
	})
}
```

- [ ] **Step 4: Run test, expect pass**

Run: `go test ./pkg/gateway/ -run TestUpstream_HoldsBalancer -v`
Expected: PASS

- [ ] **Step 5: Write failing test for target grouping**

File: add to `pkg/gateway/upstream_test.go`:

```go
func TestUpstream_TargetGrouping(t *testing.T) {
	t.Run("targets from config are pre-populated", func(t *testing.T) {
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
		}

		upstream, err := newUpstream(bifrost, config.UpstreamOptions{
			ID: "test",
			Targets: []config.TargetOptions{
				{Target: "example.com:80", Weight: 100, Tags: map[string]string{"region": "us"}},
				{Target: "10.0.1.5:8080", Weight: 50},
			},
			Balancer: config.BalancerOptions{Type: "round_robin"},
		})
		require.NoError(t, err)
		require.Len(t, upstream.targets, 2)
		assert.Equal(t, uint32(100), upstream.targets["example.com:80"].Weight)
		assert.Equal(t, "us", upstream.targets["example.com:80"].Tags["region"])
		assert.Equal(t, uint32(50), upstream.targets["10.0.1.5:8080"].Weight)
	})

	t.Run("endpoints are grouped under correct target", func(t *testing.T) {
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

		// Each static target should have one endpoint with matching weight
		assert.Len(t, upstream.targets["127.0.0.1:1234"].Endpoints, 1)
		assert.Equal(t, uint32(1), upstream.targets["127.0.0.1:1234"].Endpoints["127.0.0.1:1234"].Weight)

		assert.Len(t, upstream.targets["127.0.0.2:1235"].Endpoints, 1)
		assert.Equal(t, uint32(2), upstream.targets["127.0.0.2:1235"].Endpoints["127.0.0.2:1235"].Weight)
	})

	t.Run("flattenEndpoints returns all endpoints from all targets", func(t *testing.T) {
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
```

- [ ] **Step 6: Run test, expect failure**

Run: `go test ./pkg/gateway/ -run TestUpstream_TargetGrouping -v`
Expected: FAIL — target methods don't exist yet

- [ ] **Step 7: Run test, expect pass (after upstream.go rewrite)**

Run: `go test ./pkg/gateway/ -run TestUpstream_TargetGrouping -v`
Expected: PASS

- [ ] **Step 8: Run existing upstream tests to verify no regression**

Run: `go test ./pkg/gateway/ -run TestCreateUpstream -v -count=1`
Expected: PASS

---

### Task 6: Update Service — use proxyByAddress + upstream's balancer

**Files:**
- Modify: `pkg/gateway/service.go`
- Modify: `pkg/gateway/service_test.go`

- [ ] **Step 1: Write failing test for Service using upstream balancer + proxyByAddress**

```go
func TestService_ProxyCache(t *testing.T) {
	h := testServer(t)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = h.Shutdown(ctx)
	}()

	options := config.Options{
		Services: map[string]config.ServiceOptions{
			"testService": {
				URL: "http://127.0.0.1:8088",
			},
		},
		Upstreams: map[string]config.UpstreamOptions{
			"testUpstream": {
				ID: "testUpstream",
				Targets: []config.TargetOptions{
					{Target: "127.0.0.1:8088"},
				},
				Balancer: config.BalancerOptions{Type: "round_robin"},
			},
		},
	}

	bifrost, err := NewBifrost(options, ModeNormal)
	require.NoError(t, err)
	defer bifrost.Close()

	ctx := context.Background()

	// Service with named upstream reference
	serviceOpts := options.Services["testService"]
	serviceOpts.URL = "http://testUpstream"
	service, err := newService(bifrost, serviceOpts)
	require.NoError(t, err)

	hzCtx := app.NewContext(0)
	hzCtx.Request.SetRequestURI("http://127.0.0.1:8088/proxy/backend")
	require.Eventually(t, func() bool {
		hzCtx.Response.Reset()
		service.ServeHTTP(ctx, hzCtx)
		return hzCtx.Response.StatusCode() == 200 && string(hzCtx.Response.Body()) == "I am the backend"
	}, time.Second, 5*time.Millisecond)
}
```

- [ ] **Step 2: Run test, expect failure**

Run: `go test ./pkg/gateway/ -run TestService_ProxyCache -v`
Expected: FAIL — Service still uses old balancer field

- [ ] **Step 3: Rewrite service.go**

Major changes:
1. Remove `balancer atomic.Value`, `balancers map[string]balancer.Balancer`, `activeProxies`, `upstreamProxies`
2. Add `proxyByAddress sync.Map`, `upstreamAddresses map[string]map[string]bool`
3. Rewrite `ServeHTTP()` to use `upstream.Balancer().Select()` → `proxyByAddress.Load(endpoint.Address)`
4. Rewrite `updateEndpoints()` to manage proxy cache instead of balancer
5. **Preserve `resolveUpstreamStrategy()`** — same 4-step logic (parse URL → check `$` variable → upstreamManager lookup → auto-create exclusive upstream). The auto-created case creates a new upstream via `newUpstream()` and sets `upstream.isExclusive = true`. The new `newUpstream()` already fits this path (creates upstream + initial `refreshEndpoints(nil)`).
6. **Preserve AI service handling** — AI type sets `dynamicUpstream = variable.Model`, subscribes to all `ai:` upstreams from upstreamManager.
7. **Preserve `applyProtocolDefaults()` and `initMiddlewares()`** — unchanged.

Update imports — replace `"github.com/nite-coder/bifrost/pkg/balancer"` with `"github.com/nite-coder/bifrost/pkg/target"` (keep balancer if still imported somewhere else).

Full Service struct:

```go
type Service struct {
	bifrost           *Bifrost
	options           *config.ServiceOptions
	upstream          *Upstream
	dynamicUpstream   string
	middlewares       []app.HandlerFunc
	mu                sync.RWMutex
	proxyByAddress    sync.Map                       // address string → proxy.Proxy (lock-free reads)
	upstreamAddresses map[string]map[string]bool       // upstreamID → set of addresses (for cleanup)
	subscriptions     map[string]<-chan []*target.Endpoint
	cancelFuncs       []context.CancelFunc
}
```

Update `newService()` — upstream resolution (preserved from current code), then separate initial fetch from subscription:

```go
	svc := &Service{
		bifrost:           bifrost,
		options:           &serviceOptions,
		upstreamAddresses: make(map[string]map[string]bool),
		subscriptions:     make(map[string]<-chan []*target.Endpoint),
	}

	svc.applyProtocolDefaults()

	if err := svc.initMiddlewares(); err != nil {
		return nil, err
	}

	// Upstream resolution (same logic as current resolveUpstreamStrategy + AI branch)
	if serviceOptions.Type == config.ServiceTypeAI {
		svc.dynamicUpstream = variable.Model
		// svc.upstream stays nil for AI — ServeHTTP looks up upstream by model at runtime
	} else {
		// resolveUpstreamStrategy: same 4-step logic as current code
		// (parse URL → $variable → upstreamManager lookup → auto-create exclusive)
		if err := svc.resolveUpstreamStrategy(); err != nil {
			return nil, err
		}
	}

	// Step 1: Synchronous initial fetch (reliable, no timing dependency)
	if serviceOptions.Type == config.ServiceTypeAI {
		if svc.bifrost != nil && svc.bifrost.upstreamManager != nil {
			for _, u := range svc.bifrost.upstreamManager.List() {
				if strings.HasPrefix(u.options.ID, "ai:") {
					svc.updateEndpoints(u.options.ID, u.Endpoints())
				}
			}
		}
	} else if svc.upstream != nil {
		// Static service: only its own upstream
		svc.updateEndpoints(svc.upstream.options.ID, svc.upstream.Endpoints())
	} else if svc.bifrost != nil && svc.bifrost.upstreamManager != nil {
		// Dynamic service ($variable): all upstreams
		for _, u := range svc.bifrost.upstreamManager.List() {
			svc.updateEndpoints(u.options.ID, u.Endpoints())
		}
	}

	// Step 2: Subscribe for ongoing updates
	if serviceOptions.Type == config.ServiceTypeAI {
		if err := svc.subscribeToAIModels(); err != nil {
			return nil, err
		}
	} else if err := svc.subscribeToUpstream(); err != nil {
		return nil, err
	}
```

Add methods — `subscribeToUpstream()` for static/dynamic services, `subscribeToAIModels()` for AI services:

```go
// subscribeToUpstream subscribes to the service's own static upstream,
// or all upstreams for dynamic ($variable) services.
func (s *Service) subscribeToUpstream() error {
	if s.upstream != nil {
		if err := s.subscribeOne(s.upstream.options.ID, s.upstream); err != nil {
			return err
		}
	} else if s.bifrost != nil && s.bifrost.upstreamManager != nil {
		for _, u := range s.bifrost.upstreamManager.List() {
			if err := s.subscribeOne(u.options.ID, u); err != nil {
				return err
			}
		}
	}
	return nil
}

// subscribeToAIModels subscribes only to ai:* upstreams from the upstream manager.
func (s *Service) subscribeToAIModels() error {
	if s.bifrost == nil || s.bifrost.upstreamManager == nil {
		return nil
	}
	for _, u := range s.bifrost.upstreamManager.List() {
		if strings.HasPrefix(u.options.ID, "ai:") {
			if err := s.subscribeOne(u.options.ID, u); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) subscribeOne(id string, u *Upstream) error {
	if _, ok := s.subscriptions[id]; ok {
		return nil
	}
	ch := u.Subscribe()
	s.subscriptions[id] = ch
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelFuncs = append(s.cancelFuncs, cancel)
	go func() {
		defer cancel()
		for {
			select {
			case eps, ok := <-ch:
				if !ok {
					return
				}
				s.updateEndpoints(id, eps)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}
```

Update `Close()`:
- Remove `s.activeProxies` cleanup, use `s.proxyByAddress` instead
- Remove `s.upstreamProxies`, use `s.upstreamAddresses` instead
- **Keep `s.upstream.isExclusive` check** — if the upstream was auto-created (not from config), close it on service teardown. Shared upstreams (from UpstreamManager) are NOT closed here. Update references from `s.activeProxies`/`s.balancers` to `s.proxyByAddress` cleanup.

Update `ServeHTTP()`:

```go
func (s *Service) ServeHTTP(ctx context.Context, c *app.RequestContext) {
	logger := log.FromContext(ctx)

	defer func() {
		if r := recover(); r != nil {
			stackTrace := cast.B2S(debug.Stack())
			logger.ErrorContext(ctx, "service panic recovered", slog.Any("panic", r), slog.String("stack", stackTrace))
			c.SetStatusCode(http.StatusInternalServerError)
			c.Abort()
		}
	}()

	if err := ctx.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			// ...client cancel handling (unchanged)...
			return
		}
	}

	var myProxy proxy.Proxy
	var err error

	if len(s.dynamicUpstream) > 0 {
		upstreamID := variable.GetString(s.dynamicUpstream, c)
		if len(upstreamID) == 0 {
			// ...error handling (unchanged)...
			return
		}
		if s.options.Type == config.ServiceTypeAI {
			upstreamID = "ai:" + upstreamID
		}

		var u *Upstream
		var found bool
		if s.bifrost != nil && s.bifrost.upstreamManager != nil {
			u, found = s.bifrost.upstreamManager.Get(upstreamID)
		}
		if !found {
			// ...error handling (unchanged)...
			return
		}

		c.Set(variable.UpstreamID, upstreamID)
		bal := u.Balancer()
		if bal == nil {
			logger.Warn("balancer is nil for upstream", "upstream_id", upstreamID)
			c.SetStatusCode(http.StatusServiceUnavailable)
			return
		}
		var ep *target.Endpoint
		ep, err = bal.Select(ctx, c)
		if err == nil && ep != nil {
			if p, ok := s.proxyByAddress.Load(ep.Address); ok {
				myProxy = p.(proxy.Proxy)
			}
		}
	} else if s.upstream != nil {
		c.Set(variable.UpstreamID, s.upstream.options.ID)
		bal := s.upstream.Balancer()
		if bal == nil {
			logger.Warn("balancer is nil", "upstream_id", s.upstream.options.ID, "service_id", s.options.ID)
			c.SetStatusCode(http.StatusServiceUnavailable)
			return
		}
		var ep *target.Endpoint
		ep, err = bal.Select(ctx, c)
		if err == nil && ep != nil {
			if p, ok := s.proxyByAddress.Load(ep.Address); ok {
				myProxy = p.(proxy.Proxy)
			}
		}
	}

	if myProxy == nil || err != nil {
		c.SetStatusCode(http.StatusServiceUnavailable)
		if !errors.Is(err, balancer.ErrNotAvailable) {
			_ = c.Error(err)
		}
		return
	}

	startTime := timecache.Now()
	myProxy.ServeHTTP(ctx, c)
	endTime := timecache.Now()
	dur := endTime.Sub(startTime)
	c.Set(variable.UpstreamDuration, dur)

	if c.GetBool(variable.TargetTimeout) {
		c.Response.SetStatusCode(http.StatusGatewayTimeout)
	} else {
		c.Set(variable.UpstreamResponoseStatusCode, c.Response.StatusCode())
	}
}
```

Update `updateEndpoints()` — remove balancer creation, manage proxy cache:

```go
func (s *Service) updateEndpoints(upstreamID string, endpoints []*target.Endpoint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Track old addresses for this upstream to detect removals
	oldAddresses := s.upstreamAddresses[upstreamID]
	s.upstreamAddresses[upstreamID] = make(map[string]bool)

	for _, ep := range endpoints {
		if p, found := s.proxyByAddress.Load(ep.Address); found {
			p.(proxy.Proxy).SetEndpoint(ep)
			s.upstreamAddresses[upstreamID][ep.Address] = true
			continue
		}

		// Build proxy from ServiceOptions + endpoint (unchanged logic)
		// ... (copy the proxy creation logic from existing updateEndpoints)
		p := s.buildProxy(ep, upstreamID)
		if p != nil {
			s.proxyByAddress.Store(ep.Address, p)
			s.upstreamAddresses[upstreamID][ep.Address] = true
		}
	}

	// Close proxies that disappeared from this upstream
	if oldAddresses != nil {
		for addr := range oldAddresses {
			if !s.upstreamAddresses[upstreamID][addr] {
				if !s.isAddressUsedByAnyUpstream(addr) {
					if p, found := s.proxyByAddress.LoadAndDelete(addr); found {
						_ = p.(proxy.Proxy).Close()
					}
				}
			}
		}
	}
}

func (s *Service) isAddressUsedByAnyUpstream(addr string) bool {
	for _, addrs := range s.upstreamAddresses {
		if addrs[addr] {
			return true
		}
	}
	return false
}
```

Extract proxy creation into `buildProxy()` helper method — this is the same logic as the existing `updateEndpoints()` proxy creation but factored out:

```go
func (s *Service) buildProxy(ep *target.Endpoint, upstreamID string) proxy.Proxy {
	// Same logic as current updateEndpoints() proxy creation
	// Read s.options.Protocol, s.options.Timeout, s.options.TLSVerify, s.options.URL
	// Create appropriate proxy type (http/grpc/ai)
	// Returns the proxy or nil on error
	// (Copy the existing switch statement from service.go lines 525-676)
}
```

The proxy creation logic is unchanged (lines 525-676 of current service.go).

- [ ] **Step 4: Run service test, expect pass**

Run: `go test ./pkg/gateway/ -run TestService_ProxyCache -v`
Expected: PASS

- [ ] **Step 5: Run existing service tests**

Run: `go test ./pkg/gateway/ -run "TestServices|TestClientCancelRequest|TestDynamicService" -v -count=1`
Expected: PASS

---

### Task 7: Final verification — make check

- [ ] **Step 1: Run all package tests**

```bash
go test ./pkg/... -count=1
```

Expected: ALL PASS

- [ ] **Step 2: Run make check**

```bash
make check
```

Expected: lint 0 issues, tests all pass

- [ ] **Step 3: If lint issues found, run make fix**

```bash
make fix
make check
```

Expected: lint 0 issues, tests all pass

---

### Task 8: Code review by subagent

- [ ] **Step 1: Invoke requesting-code-review skill via subagent**

Dispatch a subagent with the requesting-code-review skill to review all changed files.

- [ ] **Step 2: Invoke security-review skill via subagent**

Dispatch a subagent with the security-review skill to review the refactored code.

- [ ] **Step 3: Address review feedback**

If review finds issues, fix them and re-run:
```bash
make check
```
Loop until both reviews pass.
