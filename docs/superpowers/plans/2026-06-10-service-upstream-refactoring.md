# Service and Upstream Refactoring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans to implement this plan task-by-task.
> **CRITICAL USER DIRECTIVE:** DO NOT COMMIT any code during the execution of these tasks. The user will handle the final commit. 
> **CRITICAL USER DIRECTIVE:** You MUST strictly follow Test-Driven Development (TDD). Write tests first.

**Goal:** Refactor Bifrost's architecture to share Upstreams globally across Services, centralizing Target health state and Service Discovery, while moving Balancer and Proxy instantiation down to the Service level.

**Architecture:** 
1. Decouple connection logic (`proxy.Proxy`) from physical target state by introducing `proxy.Endpoint` and `proxy.TargetState`.
2. Simplify the `proxy.Proxy` interface by replacing weight/tags/health methods with an `Endpoint()` method.
3. Update balancers to read weights and tags from `proxy.Endpoint()`.
4. Transform `gateway.Upstream` into a global manager that watches Service Discovery and broadcasts `[]*proxy.Endpoint`.
5. Update `gateway.Service` to subscribe to these endpoints, instantiate its own proxies based on `ServiceOptions`, and manage its own `balancer.Balancer` (fast-path) and `map[string]balancer.Balancer` (dynamic routing).

**Tech Stack:** Go, Hertz, gRPC, Testify

---

### Task 1: Introduce Shared State (`TargetState` and `Endpoint`)

**Files:**
- Create: `pkg/proxy/endpoint.go`
- Create: `pkg/proxy/endpoint_test.go`

- [ ] **Step 1: Write the failing test for TargetState**

```go
// pkg/proxy/endpoint_test.go
package proxy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/nite-coder/bifrost/pkg/timecache"
)

func TestTargetState_Health(t *testing.T) {
	ts := NewTargetState(2, time.Second)
	assert.True(t, ts.IsAvailable())

	ts.RecordFailure()
	assert.True(t, ts.IsAvailable(), "Should still be available after 1 failure")

	ts.RecordFailure()
	assert.False(t, ts.IsAvailable(), "Should be unavailable after 2 failures")

	// Mock time advancement or wait
	time.Sleep(1100 * time.Millisecond)
	assert.True(t, ts.IsAvailable(), "Should recover after FailTimeout")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/proxy -run TestTargetState_Health -v`
Expected: FAIL (undefined: NewTargetState)

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/proxy/endpoint.go
package proxy

import (
	"sync"
	"time"

	"github.com/nite-coder/bifrost/pkg/timecache"
)

// TargetState holds the shared health state for a physical IP:Port.
type TargetState struct {
	mu           sync.RWMutex
	failedCount  uint
	maxFails     uint
	failTimeout  time.Duration
	failExpireAt time.Time
}

func NewTargetState(maxFails uint, failTimeout time.Duration) *TargetState {
	return &TargetState{
		maxFails:    maxFails,
		failTimeout: failTimeout,
	}
}

func (ts *TargetState) IsAvailable() bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	if ts.maxFails == 0 {
		return true
	}
	if timecache.Now().After(ts.failExpireAt) {
		return true
	}
	return ts.failedCount < ts.maxFails
}

func (ts *TargetState) RecordFailure() {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	now := timecache.Now()
	if now.After(ts.failExpireAt) {
		ts.failedCount = 0
	}
	ts.failedCount++
	if ts.failedCount >= ts.maxFails {
		ts.failExpireAt = now.Add(ts.failTimeout)
	}
}

// Endpoint represents a discovered backend target.
type Endpoint struct {
	Address     string
	Weight      uint32
	Tags        map[string]string
	HealthState *TargetState
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/proxy -run TestTargetState_Health -v`
Expected: PASS

---

### Task 2: Refactor `proxy.Proxy` Interface and Balancers

**Files:**
- Modify: `pkg/proxy/proxy.go`
- Modify: `pkg/balancer/chash/balancer.go`
- Modify: `pkg/balancer/random/balancer.go`
- Modify: `pkg/balancer/roundrobin/balancer.go`
- Modify: `pkg/balancer/weighted/balancer.go`

- [ ] **Step 1: Write the failing build by changing the interface**

```go
// pkg/proxy/proxy.go
// Replace existing Proxy interface with:
type Proxy interface {
	ID() string
	Target() string
	Endpoint() *Endpoint // Replaces Weight(), Tags(), IsAvailable(), AddFailedCount()
	ServeHTTP(ctx context.Context, hzCtx *app.RequestContext)
	Close() error
}
```

- [ ] **Step 2: Run test to verify compilation fails**

Run: `go build ./pkg/balancer/... ./pkg/proxy/...`
Expected: FAIL (missing methods in balancers and proxy implementations)

- [ ] **Step 3: Write minimal implementation to fix balancers**

*Update all balancers (e.g., in `pkg/balancer/weighted/balancer.go`, `pkg/balancer/chash/balancer.go`)* to use `p.Endpoint().Weight`, `p.Endpoint().Tags`, and `p.Endpoint().HealthState.IsAvailable()` instead of the old direct methods.

```go
// Example snippet for a balancer checking availability:
// if p.Endpoint().HealthState.IsAvailable() { ... }

// Example snippet for a balancer checking weight:
// weight := p.Endpoint().Weight
```

- [ ] **Step 4: Run build to verify balancers pass**

Run: `go build ./pkg/balancer/...`
Expected: PASS (although proxy implementations will still fail, which is handled in Task 3)

---

### Task 3: Update `HTTP`, `gRPC`, and `AI` Proxy Implementations

**Files:**
- Modify: `pkg/proxy/http/proxy.go` and `proxy_test.go`
- Modify: `pkg/proxy/grpc/proxy.go` and `proxy_test.go`
- Modify: `pkg/proxy/ai/proxy.go` and `proxy_test.go`

- [ ] **Step 1: Write minimal implementation to fix proxy structs**

Update `HTTP` and `gRPC` proxy structs to remove `failedCount`, `failExpireAt`, and `weight`/`tags` fields. Add `endpoint *Endpoint`. Update `Options` structs to no longer take `Weight`, `Tags`, `MaxFails`, `FailTimeout`.

```go
// In pkg/proxy/http/proxy.go
type Proxy struct {
	id          string
	targetHost  string
	options     *Options
	endpoint    *Endpoint
	client      *client.Client
	// ... director, etc.
}

func (p *Proxy) Endpoint() *proxy.Endpoint {
	return p.endpoint
}

// Inside ServeHTTP defer block:
// if c.Response.StatusCode() >= http.StatusInternalServerError {
//     p.endpoint.HealthState.RecordFailure()
// }
```

Do the same for `pkg/proxy/grpc/proxy.go`.

For `pkg/proxy/ai/proxy.go`, create a dummy Endpoint during initialization since AI targets don't use dynamic discovery yet:
```go
// In pkg/proxy/ai/proxy.go
func NewProxy(opts ProxyOptions) *Proxy {
    // ... existing setup
	p.endpoint = &proxy.Endpoint{
		Address: opts.Target,
		Weight:  opts.Weight,
		HealthState: proxy.NewTargetState(0, 0), // Never fails
	}
	return p
}

func (p *Proxy) Endpoint() *proxy.Endpoint {
	return p.endpoint
}
```

- [ ] **Step 2: Fix tests in proxy packages**

Update `New(...)` calls in `proxy_test.go` files to inject a valid `*proxy.Endpoint`.

- [ ] **Step 3: Run test to verify proxy packages pass**

Run: `go test ./pkg/proxy/... -v`
Expected: PASS

---

### Task 4: Global `Upstream` and Event Broadcasting

**Files:**
- Modify: `pkg/gateway/upstream.go`
- Create: `pkg/gateway/upstream_manager.go`

- [ ] **Step 1: Write tests for Upstream broadcasting**
Write a test in `upstream_test.go` verifying that `Upstream` takes Service Discovery instances, creates `[]*proxy.Endpoint`, and broadcasts them to a subscriber channel.

- [ ] **Step 2: Run test to verify failure**
Run: `go test ./pkg/gateway -run TestUpstream_Broadcast -v`
Expected: FAIL

- [ ] **Step 3: Write implementation**

Create `pkg/gateway/upstream_manager.go`:
```go
// Minimal manager that holds global upstreams.
```

Modify `pkg/gateway/upstream.go`:
Remove `Balancer` and `proxy.Proxy` slice. Add pub/sub for endpoints.
```go
type Upstream struct {
	// ... discovery
	subscribers []chan []*proxy.Endpoint
	endpoints   []*proxy.Endpoint
	// ...
	targetsMu   sync.Mutex
	targets     map[string]*proxy.TargetState // Keyed by physical address (IP:Port)
}

func (u *Upstream) Subscribe() <-chan []*proxy.Endpoint {
    // return channel and send current endpoints
}

// In refreshEndpoints:
// Map instances to Endpoints. Maintain the map of TargetState directly inside each Upstream instance.
// Scoping TargetState to the Upstream struct has several benefits:
// 1. Clean Lifecycle & GC: When a dynamic or direct upstream is closed/deleted, its local targets map is garbage-collected. This prevents memory leaks of orphan target states in the global UpstreamManager.
// 2. Config Compliance: Different upstreams pointing to the same backend IP address will maintain isolated health states, correctly honoring their respective passive health check configurations (MaxFails, FailTimeout) without key collision or ordering dependencies.
// 3. Simpler Concurrency: Scoping map locks locally to the Upstream reduces lock contention and removes the need for compound cache keys.
//
// Look up or create TargetState within `u.targets` using `address` as key. Send updated endpoints to subscribers.
```

- [ ] **Step 4: Run test to verify it passes**
Run: `go test ./pkg/gateway -run TestUpstream_Broadcast -v`
Expected: PASS

---

### Task 5: Refactor `Service` to Manage Balancers

**Files:**
- Modify: `pkg/gateway/service.go`

- [ ] **Step 1: Update `Service` Struct and tests**
Update `Service` struct:
```go
type Service struct {
    // ...
	balancer  balancer.Balancer             // Fast-path
	balancers map[string]balancer.Balancer  // Dynamic routing
	// ...
}
```

- [ ] **Step 2: Run test to verify compilation fails in gateway package**
Run: `go test ./pkg/gateway/...`
Expected: FAIL (service.go still references old upstreams map)

- [ ] **Step 3: Write implementation for `Service`**
Update `newService`: Subscribe to the global `Upstream` via `UpstreamManager`.
Create a background goroutine for the Service that listens to the endpoint channel.
When endpoints arrive:
1. Iterate `endpoints`, generate `[]proxy.Proxy` using `ServiceOptions` (HTTP vs gRPC config).
2. Instantiate `balancer.Factory`.
3. Update `s.balancer` (or `s.balancers[dynamicID]`).

Update `ServeHTTP`:
```go
// For fast-path:
myProxy, err = s.balancer.Select(ctx, c)

// For dynamic:
targetBalancer, found := s.balancers[upstreamName]
myProxy, err = targetBalancer.Select(ctx, c)
```

- [ ] **Step 4: Run test to verify gateway tests pass**
Run: `go test ./pkg/gateway/... -v`
Expected: PASS

---

### Task 6: Comprehensive Code and Security Review

**Files:**
- All modified files

- [ ] **Step 1: Request Code Review**
Invoke the `requesting-code-review` skill. Address all feedback and fix any issues found. Repeat this step until the review passes completely.

- [ ] **Step 2: Request Security Review**
Invoke the `security-review` skill. Ensure no security regressions or data leaks are introduced. Address all feedback and fix any issues found. Repeat this step until the review passes completely.

---

### Task 7: Final Integration and Verification

**Files:**
- End-to-end testing

- [ ] **Step 1: Run complete test suite**
Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 2: Manual review**
Verify that NO CODE HAS BEEN COMMITTED. Wait for the user to review the changes and commit manually.
