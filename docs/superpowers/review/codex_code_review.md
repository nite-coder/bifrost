# Codex Code Review

## Scope

- Reviewed current branch `refactor/upstream` against `main`.
- Base: `ded37abb7891b4661c29318912433206736c1998`
- Head: `db32ab54e937f14821d84d327411249632bca8f5`
- Included current uncommitted working-tree change in `pkg/gateway/service.go`.
- Primary review target: service/upstream refactor that centralizes upstream discovery and target health state through `UpstreamManager`, with per-service proxy and balancer construction.

## Verification

- `make check`: PASS
- Lint: PASS, `0 issues`
- Go tests: PASS, `762 tests`
- E2E upgrade test: PASS

Passing verification does not make this branch merge-ready because the critical issue below is not covered by the current tests.

## Strengths

- The refactor mostly follows the intended separation: `Upstream` now publishes endpoint state, while `Service` builds protocol-specific proxies and balancers locally.
- Endpoint health state is thread-safe and proxies update endpoint references atomically.
- Existing balancer and proxy tests were updated to exercise the new `Endpoint()`-based interface.
- `make check` passes with race-enabled tests and e2e upgrade coverage.

## Issues

### Critical

1. Shared upstreams are closed by individual services

- File: `pkg/gateway/service.go:86`
- What is wrong: `Service.Close()` unconditionally calls `s.upstream.Close()` after the service unsubscribes. In the new design, `s.upstream` can be a shared upstream returned by `s.bifrost.upstreamManager.Get(hostname)` at `pkg/gateway/service.go:421`. Closing it stops the watcher, closes all subscriber channels, and closes discovery resources in `pkg/gateway/upstream.go:38`.
- Why it matters: Closing one service can break other services that share the same upstream. During `Bifrost.Close()`, the first service can also close a manager-owned upstream before `UpstreamManager.Close()` runs at `pkg/gateway/bifrost.go:268`, causing double-close semantics and lifecycle ownership confusion. This violates the refactor goal that upstreams are global and shared across services.
- How to fix: Track whether `s.upstream` is service-owned/direct or manager-owned. Only close service-owned direct upstreams in `Service.Close()`. Manager-owned upstreams should only be unsubscribed by services and closed by `UpstreamManager.Close()`. Add a regression test with two services sharing one configured upstream, close one service, then assert the other service still receives endpoint updates and can serve traffic.

### Important

1. Shared target health state ignores per-upstream passive health configuration

- File: `pkg/gateway/upstream_manager.go:119`
- What is wrong: `GetOrCreateTargetState(address, maxFails, failTimeout)` caches health state only by physical address. If two upstream definitions point at the same address but use different passive health settings, whichever upstream creates the state first determines `maxFails` and `failTimeout` for all later upstreams.
- Why it matters: The design requires shared physical target health, but it also states external business behavior and config compatibility must not change. Per-upstream health configuration currently becomes order-dependent for shared targets.
- How to fix: Decide the intended compatibility rule explicitly. If per-upstream health settings must remain independent, include the health configuration or upstream ID in the state key. If physical address health is intentionally global regardless of upstream config, document the behavioral change and add tests that lock down the chosen behavior.

2. Missing tests for service-owned versus manager-owned upstream lifecycle

- File: `pkg/gateway/service_test.go`
- What is wrong: Existing tests cover service creation, dynamic upstreams, AI models, and shutdown paths, but they do not cover closing one service while another service still uses the same manager-owned upstream.
- Why it matters: This is exactly the ownership boundary introduced by the refactor and is where the critical regression occurs.
- How to fix: Add a focused test for shared upstream lifecycle ownership before merging.

### Minor

1. Some legacy proxy options remain after moving health and weight into endpoints

- File: `pkg/proxy/http/proxy.go:82`
- File: `pkg/proxy/grpc/proxy.go:45`
- What is wrong: `Options` still exposes `MaxFails`, `FailTimeout`, `Weight`, and `Tags` for fallback endpoint construction. This is backward-compatible for direct proxy tests, but it partially blurs the new endpoint ownership model.
- Why it matters: Future changes may accidentally configure proxy-local health when the intended path is shared endpoint state.
- How to fix: Keep if needed for compatibility, but document that these fields are fallback-only when `Endpoint` is nil.

## Recommendations

- Fix the shared upstream ownership bug before merge.
- Add regression tests for multi-service shared upstream shutdown behavior.
- Add a test or documentation decision for target health state keying when the same address appears under upstreams with different passive health settings.

## Assessment

**Ready to merge?** No

**Reasoning:** The branch passes automated verification, but `Service.Close()` currently closes manager-owned shared upstreams. That can stop discovery and close subscribers for unrelated services, which breaks the central architectural goal of globally shared upstreams.
