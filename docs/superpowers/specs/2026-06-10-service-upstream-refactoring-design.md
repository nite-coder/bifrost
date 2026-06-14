# Service and Upstream Refactoring Design

## 1. Context and Problem Statement
Currently in Bifrost, a `Service` has a 1-to-1 relationship with an `Upstream` instance. This means if multiple services (e.g., HTTP and gRPC services) point to the same Upstream configuration (like a shared Nacos discovery or the same DNS hostname), Bifrost creates duplicate `Upstream` instances.

This leads to two major issues:
1. **Resource Waste**: Duplicate background watchers (e.g., Nacos, DNS, K8s) are spawned for the same targets.
2. **Fragmented Health State**: Health checks (like `failedCount` for circuit breaking) are maintained per `proxy.Proxy` inside the `Upstream`. If Target A goes down, Service A might detect it and stop sending traffic, but Service B is unaware and will still send traffic to the failing Target A until its own proxy fails.

## 2. Goal
Refactor the internal architecture to allow `Upstream` instances to be shared globally across multiple `Service` instances.
- **Strict Constraint**: The user's `config.yaml` configuration must remain 100% backward compatible.
- **Strict Constraint**: External business behaviors (load balancing, per-service timeouts, protocols) must not change.
- **Desired Outcome**: One background watcher per upstream configuration. Shared health check states across all services pointing to the same targets.

## 3. Architecture Design (Direction 1)

The core principle is: **"Upstreams manage endpoints and health states globally; Services manage proxy execution and balancers locally."**

### 3.1. Global Upstream Management
- **`UpstreamManager` (New)**: Attached to the global `Bifrost` struct. It initializes all `config.Upstreams` exactly once at startup.
- **`Upstream` (Modified)**: 
  - Becomes a global singleton per configured upstream ID.
  - Owns the `ServiceDiscovery` watcher.
  - Maintains a list of `Endpoint`s.
  - **No longer holds** `proxy.Proxy` or `balancer.Balancer`.
  - Provides a subscription mechanism (Pub/Sub) to broadcast `Endpoint` updates to interested `Service`s.

### 3.2. Shared Target State
- **`TargetState` (New)**: A shared thread-safe struct representing the health state of a physical IP:Port.
  - Holds `failedCount`, `failExpireAt`, and methods like `RecordFailure()` and `IsAvailable()`.
- **`Endpoint` (New)**: Represents the discovered target.
  - Holds `Address` (IP:Port), `Weight`, `Tags` (from discovery), and a pointer to the shared `*TargetState`.
  - `Upstream` broadcasts `[]Endpoint` to its subscribers.

### 3.3. Service and Proxy Adjustments
- **`Service` (Modified)**:
  - Subscribes to the global `UpstreamManager` to receive `[]Endpoint` updates.
  - When an update is received, it combines the `[]Endpoint` with its own `config.ServiceOptions` (e.g., `Timeout`, `Protocol`, `TLSVerify`) to instantiate new `proxy.Proxy` arrays.
  - Holds a primary `balancer balancer.Balancer` (for standard fast-path API requests) and a `balancers map[string]balancer.Balancer` (for dynamic routing, such as AI model selection), mirroring the current optimization of `upstream` and `upstreams`.
- **`proxy.Proxy` Implementations (Modified)**:
  - No longer track `failedCount` internally.
  - Hold a reference to the `Endpoint` (which contains the `*TargetState`).
  - Upon request failure (e.g., HTTP 5xx or Timeout), the proxy calls `p.endpoint.HealthState.RecordFailure()`, which updates the global state. All other proxies holding this state will immediately recognize the target as unavailable.

## 4. Data Flow
1. **Startup**: `Bifrost` loads `config.yaml`. `UpstreamManager` starts watchers for all declared upstreams.
2. **Subscription**: `Service` initializes and subscribes to its designated upstream.
3. **Discovery Event**: Nacos updates targets. `Upstream` updates the shared `TargetState` list and broadcasts `[]Endpoint` to subscribers.
4. **Proxy Creation**: `Service` receives `[]Endpoint`. It drops old proxies and creates new ones, injecting the shared `*TargetState` into each. It updates its `Balancer`.
5. **Request Execution**: 
   - A request hits `Service`.
   - `Service`'s `Balancer` picks a `Proxy`.
   - `Proxy` checks `p.endpoint.HealthState.IsAvailable()`. If false, fails fast.
   - If true, forwards request. If it fails, calls `p.endpoint.HealthState.RecordFailure()`.

## 5. Backward Compatibility
- The `config.Options` struct remains identical.
- Proxies are still customized at creation time with `ServiceOptions`, ensuring differing protocols (HTTP vs gRPC) and TLS settings have distinct connection pools, even though they share the logical `TargetState`.