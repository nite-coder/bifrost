# Service and Upstream Refactoring Design v2

Revision date: 2026-06-11
Changes from v1 (2026-06-10):
- `pkg/target` extracted from `pkg/proxy` for Endpoint + State types
- Balancer ownership moved from Service → Upstream (Kong model)
- Balancer interface changed: selects `*target.Endpoint`, removed `Proxies()`
- Service manages proxy cache (`proxyByAddress`) instead of owning balancers
- Upstream change detection via endpoint hash
- **Added Target grouping layer** (Kong model): Upstream groups endpoints by their origin target (hostname) for observability and future target-level features.
- Updated package dependency diagram

## 1. Context and Problem Statement

Currently in Bifrost, a `Service` has a 1-to-1 relationship with an `Upstream` instance. This means if multiple services (e.g., HTTP and gRPC services) point to the same Upstream configuration (like a shared Nacos discovery or the same DNS hostname), Bifrost creates duplicate `Upstream` instances.

This leads to two major issues:
1. **Resource Waste**: Duplicate background watchers (e.g., Nacos, DNS, K8s) are spawned for the same targets.
2. **Fragmented Health State**: Health checks (like `failedCount` for circuit breaking) are maintained per `proxy.Proxy` inside the `Upstream`. If Target A goes down, Service A might detect it and stop sending traffic, but Service B is unaware and will still send traffic to the failing Target A until its own proxy fails.

Additionally, Bifrost currently flattens all target resolutions into a flat endpoint list, losing the grouping information (which address belongs to which target/hostname). Kong's three-layer model (Upstream → Target → Address) provides better observability and future feature paths (target-level health, target-level circuit breaker, per-target weight control).

## 2. Goal

Refactor the internal architecture to:
1. Allow `Upstream` instances to be shared globally across multiple `Service` instances.
2. Add a Target grouping layer (Kong model) within Upstream for structural grouping of addresses by their origin target.

- **Strict Constraint**: The user's `config.yaml` configuration must remain 100% backward compatible.
- **Strict Constraint**: External business behaviors (load balancing, per-service timeouts, protocols) must not change.
- **Desired Outcome**: One background watcher per upstream configuration. Shared health check states across all services pointing to the same targets. Target-aware internal data model.

## 3. Architecture Design

Core principle: **"Upstream manages targets, endpoints, balancer, and health states globally; Service manages proxy cache and request execution locally."**

### 3.1. Three-Layer Model

```
Upstream (shared singleton)
  └── Targets[] (grouped by hostname from config TargetOptions.Target)
        └── Endpoints[] (resolved IP:port, inherits weight from target)
              └── State (health tracking, shared across all services)
```

### 3.2. Package Restructuring

New package `pkg/target` extracted from `pkg/proxy`:

```
pkg/target/
  endpoint.go    — Endpoint struct (was proxy.Endpoint, represents IP:port)
  state.go       — State (was proxy.TargetState, health tracking per IP)
  target.go      — Target struct (new, groups endpoints by hostname)
  hash.go        — EndpointHash() for change detection over flat list
```

Package dependency changes:

```
BEFORE:                        AFTER:
proxy                          target
  Endpoint                       Endpoint
  TargetState                    State
  ↑                            ↗  ↑  ↖
balancer                      balancer  proxy  gateway
  (imports proxy               (pure algorithm,
   for Endpoint)                no proxy dep)
```

`balancer` no longer imports `proxy`. `proxy` imports `target` for Endpoint/State.
`gateway` imports `target` for Endpoint, State, and Target types.

### 3.3. Global Upstream Management

- **`UpstreamManager`** (existing): Attached to the global `Bifrost` struct. Initializes all `config.Upstreams` exactly once at startup.
- **`Upstream`** (modified, per config ID):
  - Global singleton per upstream ID.
  - Owns the `ServiceDiscovery` watcher.
  - Owns `targets map[string]*Target` keyed by target name (hostname:port) — each target owns its resolved endpoints.
  - Endpoint structs live across refresh cycles so their `State` is preserved by pointer reuse within each target.
  - **Owns `balancer.Balancer`** — moved from Service.
  - Detects endpoint changes via `EndpointHash()` over the flat address list before rebuilding balancer and broadcasting.
  - Provides `Endpoints()` method for reliable synchronous fetch of current flat endpoint list.
  - Provides `Subscribe()` method for ongoing updates (does NOT send initial state — caller must call `Endpoints()` first).
  - Broadcasts `[]*target.Endpoint` updates to all subscribers after each refresh.
  - Uses **simple drop** pattern (no drain): buffer=64 absorbs concurrent collisions, drops on full with a warn log.

```go
type Upstream struct {
    mu            sync.RWMutex
    discovery     provider.ServiceDiscovery
    bifrost       *Bifrost
    options       *config.UpstreamOptions
    subscribers   []chan []*target.Endpoint
    targets       map[string]*target.Target  // keyed by target name (hostname:port)
    endpointsHash string                     // hash of entire flat []*Endpoint, for change detection
    balancer      balancer.Balancer          // moved from Service
    watchOnce     sync.Once
    cancel        context.CancelFunc
    isExclusive   bool
}

// Endpoints returns the current flat endpoint list synchronously (reliable, no timing dependency).
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
```

### 3.4. Target Struct

```go
// pkg/target/target.go
type Target struct {
    Name      string                // hostname:port (from config TargetOptions.Target)
    Weight    uint32                // from config TargetOptions.Weight
    Tags      map[string]string     // from config TargetOptions.Tags
    Endpoints map[string]*Endpoint  // resolved IP:port → Endpoint
}
```

Purpose of the Target layer:
- **Structural grouping**: Know which addresses belong to which hostname.
- **Observability**: Admin API can report "target example.com healthy?" (future).
- **Target-level features**: Weight scaling, circuit breaker (future).
- **Weight propagation**: Each endpoint inherits weight from its target via `instance.Weight()` (unchanged behavior).

### 3.5. Shared Endpoint State

- **`target.State`** (was `proxy.TargetState`): A shared thread-safe struct representing the health state of a physical IP:Port.
  - Holds `failedCount`, `failExpireAt`, methods `RecordFailure()` and `IsAvailable()`.
- **`target.Endpoint`** (was `proxy.Endpoint`): Represents a discovered address.
  - Holds `Address` (IP:Port), `Weight`, `Tags` (from discovery), and an embedded `*target.State`.
  - Endpoint structs survive across refresh cycles within each Target, so their `State` is naturally preserved.

### 3.6. ServiceDiscovery Interface with Target Grouping

**Breaking change**: `ServiceDiscovery` returns a new `DiscoveryResult` struct that preserves the target→instances hierarchy. Providers already iterate config TargetOptions and know which instances belong to which target — no need to flatten and re-group.

```go
// pkg/provider/provider.go

// DiscoveryResult preserves the target→instances grouping.
type DiscoveryResult struct {
    Target string            // hostname:port (from config TargetOptions.Target, or discovery name for DNS/Nacos/K8s)
    Weight uint32            // target-level weight from config
    Tags   map[string]string // target-level tags from config
    Nodes  []Instancer       // resolved instances for this target
}

type ServiceDiscovery interface {
    GetInstances(ctx context.Context, options GetInstanceOptions) ([]DiscoveryResult, error)
    Watch(ctx context.Context, options GetInstanceOptions) (<-chan []DiscoveryResult, error)
    Close() error
}
```

Provider behavior:

| Provider | Returns | Efficiency |
|----------|---------|------------|
| ResolverDiscovery | One `DiscoveryResult` per config `TargetOptions`, each with N resolved IPs | Poll-based |
| StaticDiscovery | One `DiscoveryResult` per config `TargetOptions` (ai: prefix), each with 1 static IP | Static |
| DNS | One `DiscoveryResult` with `Target = options.Name`, all resolved IPs as Nodes | Sends results on change, or nil to trigger re-fetch |
| Nacos/K8s | One `DiscoveryResult` with `Target = service name`, all nodes as Nodes | **Push-based**: Sends actual instances on channel to avoid redundant polling |

### 3.7. Endpoint Refresh with Target Grouping

Discovery returns pre-grouped results. Upstream iterates directly — no `groupByTargetName()` or `matchTargetName()` needed.

```
Config:
  upstream.targets:
    - target: "example.com:80"    weight: 100
    - target: "10.0.1.5:8080"    weight: 50

Discovery returns 2 DiscoveryResults:
  {Target: "example.com:80", Weight: 100, Nodes: [3 Instancers]}
  {Target: "10.0.1.5:8080", Weight: 50, Nodes: [1 Instancer]}

Upstream creates/updates Targets:
  targets["example.com:80"]:
    Name: "example.com:80", Weight: 100
    Endpoints: {
      1.1.1.1:80 → Endpoint{Weight: 100, State: ...}
      1.1.1.2:80 → Endpoint{Weight: 100, State: ...}
      1.1.1.3:80 → Endpoint{Weight: 100, State: ...}
    }
  targets["10.0.1.5:8080"]:
    Name: "10.0.1.5:8080", Weight: 50
    Endpoints: {10.0.1.5:8080 → Endpoint{Weight: 50, State: ...}}

Flat list for balancer:
  [1.1.1.1:80 (W=100), 1.1.1.2:80 (W=100), 1.1.1.3:80 (W=100), 10.0.1.5:8080 (W=50)]
```

```go
func (u *Upstream) refreshEndpoints(results []provider.DiscoveryResult) error {
    // Fetch from discovery if not provided (initial load)
    if len(results) == 0 && u.discovery != nil {
        opts := provider.GetInstanceOptions{...}
        var err error
        results, err = u.discovery.GetInstances(context.Background(), opts)
        if err != nil {
            return err
        }
    } else if len(results) == 0 {
        // If results is empty after GetInstances or passed as empty from Watch,
        // it means we should probably clear endpoints or it's a no-op Watch signal.
        // In the implementation, we'll ensure Watch sends actual results for efficiency.
        return nil 
    }

    // Track which targets are seen in this refresh cycle
    seen := make(map[string]bool, len(results))
    // ... rest of implementation ...
}
```

### 3.8. Balancer Interface

```go
// pkg/balancer/pkg.go

type Balancer interface {
    Select(ctx context.Context, hzCtx *app.RequestContext) (*target.Endpoint, error)
}
```

Removed `Proxies() []proxy.Proxy`. The balancer is a pure algorithm over endpoints.

`CreateBalancerHandler` signature:

```go
type CreateBalancerHandler func(endpoints []*target.Endpoint, params any) (Balancer, error)
```

Each balancer implementation (round_robin, weighted, random, chash) changes:
- Input: `[]proxy.Proxy` → `[]*target.Endpoint`
- Weight: read directly from `ep.Weight` (was `p.Endpoint().Weight`)
- Health: read directly from `ep.State.IsAvailable()` (was `p.Endpoint().HealthState`)
- Hash key: `ep.Address` (was `p.Target()`)

### 3.9. Service and Proxy Cache

- **`Service`** (modified):
  - **Fetches initial state synchronously** via `upstream.Endpoints()` — no timing dependency on channel buffer.
  - **Subscribes for ongoing updates** via `upstream.Subscribe()` — receives flat `[]*target.Endpoint`.
  - Subscription strategy differs by type: **`subscribeToUpstream()`** for static/dynamic (`$variable`) services, **`subscribeToAIModels()`** for AI services (only subscribes to `ai:*` upstreams, not all).
  - On endpoint update: creates/drops proxies using `ServiceOptions` (protocol/timeout/TLS/URL — **unchanged logic**).
  - Stores proxies in **`proxyByAddress sync.Map`** (address string → proxy.Proxy). This allows lock-free reads in the high-frequency `ServeHTTP` hot path, avoiding data races with `updateEndpoints` writes.
  - Removes `balancer atomic.Value` and `balancers map[string]balancer.Balancer`.
  - On request: calls `upstream.Balancer().Select()` → `proxyByAddress.Load(endpoint.Address)`.
  - **Exclusive upstreams preserved**: When a service references a hostname not in config (e.g., `url: http://example.com/api` where `example.com` is not a configured upstream), `resolveUpstreamStrategy()` auto-creates a dedicated upstream with `isExclusive = true`. On service close, the exclusive upstream is cleaned up. Shared upstreams (from UpstreamManager) are never closed by individual services.

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

#### Proxy Cache Lookup

During `updateEndpoints()` — proxy dedup is now by address (no hash needed):

```go
for _, ep := range endpoints {
    if p, found := s.proxyByAddress.Load(ep.Address); found {
        existing := p.(proxy.Proxy)
        existing.SetEndpoint(ep)
        s.upstreamAddresses[upstreamID][ep.Address] = true
        continue
    }
    // create proxy from ServiceOptions + endpoint (unchanged logic)
    p := buildProxy(s.options, ep)
    s.proxyByAddress.Store(ep.Address, p)
    s.upstreamAddresses[upstreamID][ep.Address] = true
}

// Close proxies for addresses that disappeared from this upstream
for addr := range oldAddresses {
    if !stillUsedByAnyUpstream(addr) {
        if p, found := s.proxyByAddress.LoadAndDelete(addr); found {
            p.(proxy.Proxy).Close()
        }
    }
}
```

During `ServeHTTP()`:
```go
// static upstream
endpoint, err := s.upstream.Balancer().Select(ctx, c)
if p, ok := s.proxyByAddress.Load(endpoint.Address); ok {
    myProxy = p.(proxy.Proxy)
}

// dynamic upstream (AI / $variable)
u, found := s.bifrost.upstreamManager.Get(upstreamID)
endpoint, err := u.Balancer().Select(ctx, c)
if p, ok := s.proxyByAddress.Load(endpoint.Address); ok {
    myProxy = p.(proxy.Proxy)
}
```


### 3.10. Upstream Change Detection (Endpoint Hash)

Avoid redundant balancer rebuild and broadcast:

```go
// pkg/target/hash.go
func EndpointHash(endpoints []*Endpoint) string
```

The hash computes a single hash over the entire flat endpoint list, covering: address + weight + sorted tags for each endpoint. **Not** State (shared pointer, preserved across rebuilds for same address).

Hash is order-independent (sorts by address internally).

### 3.11. Proxy Ownership and Request Execution

- **`proxy.Proxy` Implementations** (unchanged interface):
  - HTTP proxy holds `*client.Client` (Hertz, per-proxy — **not shared** in this refactoring).
  - gRPC proxy holds `grpc.ClientConnInterface` (per-target-address, cannot share).
  - Both hold a reference to `*target.Endpoint` (type changed from `*proxy.Endpoint`).
  - On failure: call `p.Endpoint().State.RecordFailure()`. All proxies (across all Services) sharing this endpoint see the updated health state.

## 4. Data Flow

1. **Startup**: `Bifrost` loads `config.yaml`. `UpstreamManager.Start()` creates all upstreams with watchers and balancers.
2. **Target Resolution**: Each `Upstream` resolves its `TargetOptions` via discovery, which returns `[]DiscoveryResult` pre-grouped by target name.
3. **Initial Fetch + Subscription**: `Service` initializes, first calls `upstream.Endpoints()` synchronously for initial state, then calls `upstream.Subscribe()` for ongoing updates.
4. **Discovery Event**: Nacos/DNS/K8s updates targets.
   - `Upstream.refreshEndpoints()` receives `[]DiscoveryResult` — targets are already grouped, no re-grouping needed.
   - Iterates results, updates each `Target.Endpoints` map (preserving State pointers).
   - Computes `EndpointHash()` over flat list. If unchanged, skips rebuild.
   - If changed: rebuilds balancer, broadcasts flat `[]*target.Endpoint` to all subscribers.
5. **Proxy Cache Update**: `Service` receives new endpoints, drops stale proxies, creates new ones (with `ServiceOptions`), updates `proxyByAddress`.
6. **Request Execution**:
   - Request hits `Service.ServeHTTP()`.
   - `upstream.Balancer().Select(ctx, c)` → `*target.Endpoint`.
   - `proxyByAddress[endpoint.Address]` → `proxy.Proxy`.
   - Proxy checks `endpoint.State.IsAvailable()`. If false, fails fast.
   - If true, forwards request. On failure: `endpoint.State.RecordFailure()`.

## 5. Backward Compatibility

- **Config**: `UpstreamOptions.Balancer` already exists in config. `ServiceOptions` has no balancer field. Target grouping is an internal struct change. **Zero config changes.**
- **ServiceOptions fields**: All fields (Protocol, Timeout, TLSVerify, URL, PassHostHeader, MaxConnsPerHost) are consumed during proxy creation in `updateEndpoints()`, which remains unchanged. Behavior is preserved.
- **Weight behavior**: Each resolved address gets `TargetOptions.Weight` via `instance.Weight()` — **identical to current behavior**. Target grouping is structural only, weight distribution unchanged.
- **Balancer type per upstream**: Config `upstreams.<id>.balancer.type` is now read by `Upstream` (not `Service`) but same field, same semantics.
- **AI model balancer**: `models.<id>.balancer.type` is already mapped to `UpstreamOptions.Balancer` in `UpstreamManager.Start()`. No change.
- **Proxy lifecycle**: One proxy per unique address per Service (unchanged). HTTP client per proxy (unchanged). gRPC connection per proxy (unchanged).
- **Health state**: `proxyByAddress` is rebuilt on endpoint change; old proxies are closed (same as today).
- **Dynamic upstreams** (URL starting with `$`): Service looks up upstream from `UpstreamManager` instead of `s.balancers` map — functionally equivalent.
- **Subscriber interface**: Still delivers flat `[]*target.Endpoint` — no change to Service code for receiving updates.

## 6. Change Scope

| File | Change |
|------|--------|
| `pkg/target/state.go` | New. State (was proxy.TargetState), moved from pkg/proxy/endpoint.go. |
| `pkg/target/endpoint.go` | New. Endpoint (was proxy.Endpoint), moved from pkg/proxy/endpoint.go. |
| `pkg/target/target.go` | New. Target struct — groups endpoints by hostname for Kong-model grouping. |
| `pkg/target/hash.go` | New. EndpointHash() — order-independent SHA-256 over endpoint list for change detection. |
| `pkg/proxy/endpoint.go` | Remove. Content moved to `pkg/target/`. |
| `pkg/proxy/proxy.go` | Keep interface. Change `Endpoint() *proxy.Endpoint` → `Endpoint() *target.Endpoint` and `SetEndpoint()` signature. |
| `pkg/proxy/http/proxy.go` | Update Endpoint type import. |
| `pkg/proxy/grpc/proxy.go` | Update Endpoint type import. |
| `pkg/proxy/ai/proxy.go` | Update Endpoint type import. |
| `pkg/balancer/pkg.go` | Change interface (`Select` returns `*target.Endpoint`, remove `Proxies()`). Change `CreateBalancerHandler` signature. |
| All 4 balancers | Rewrite to operate on `[]*target.Endpoint`. |
| `pkg/provider/provider.go` | Add `DiscoveryResult` struct. Change `ServiceDiscovery.GetInstances()` and `Watch()` return type from `[]Instancer` to `[]DiscoveryResult`. |
| `pkg/provider/dns/discovery.go` | Update return type to `[]DiscoveryResult` — all resolved IPs grouped under one result with `Target = options.Name`. |
| `pkg/provider/nacos/discovery.go` | Update return type to `[]DiscoveryResult` — all nodes from Nacos grouped under one result with `Target = options.Name`. |
| `pkg/provider/k8s/k8s.go` | Update return type to `[]DiscoveryResult` — all endpoints from K8s grouped under one result with `Target = options.Name`. |
| `pkg/gateway/discovery.go` | ResolverDiscovery + StaticDiscovery: update return type to `[]DiscoveryResult` — one result per config TargetOptions. |
| `pkg/gateway/upstream.go` | Add `balancer.Balancer` field, **`targets map[string]*Target`** (replaces flat `endpointsMap`), `endpointsHash` field, `rebuildBalancer()` method. `refreshEndpoints()` signature changes to `[]DiscoveryResult`, **no group-by-target logic needed** — results are pre-grouped. Remove `groupByTargetName()`, `matchTargetName()`. |
| `pkg/gateway/service.go` | Remove `balancer atomic.Value`, `balancers map[string]balancer.Balancer`, `activeProxies`, `upstreamProxies`. Add **`proxyByAddress sync.Map`** (lock-free reads) + `upstreamAddresses`. Rewrite `ServeHTTP()` to use upstream balancer + `proxyByAddress.Load()`. Remove balancer factory call from `updateEndpoints()`. |
| `pkg/gateway/upstream_manager.go` | No change (balancer config already passed through). |

### Out of scope for this refactoring

- Sharing Hertz client across HTTP proxies within a Service (has TLS ServerName complication — separate optimization).
- Active health checks (future feature).
- Admin API for target-level health reporting (future feature).
- Target-level circuit breaker (future feature).
- Target-level weight scaling (future feature).
