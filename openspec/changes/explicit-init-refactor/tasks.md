## 1. Middleware Init Functions

- [ ] 1.1 Convert `pkg/middleware/addprefix/add_prefix.go`: `init()` → `Init() error`
- [ ] 1.2 Convert `pkg/middleware/buffering/buffering.go`: `init()` → `Init() error`
- [ ] 1.3 Convert `pkg/middleware/compression/gzip.go`: `init()` → `Init() error`
- [ ] 1.4 Convert `pkg/middleware/coraza/coraza.go`: `init()` → `Init() error`
- [ ] 1.5 Convert `pkg/middleware/cors/config.go`: `init()` → `Init() error`
- [ ] 1.6 Convert `pkg/middleware/iprestriction/ip_restriction.go`: `init()` → `Init() error`
- [ ] 1.7 Convert `pkg/middleware/mirror/mirror.go`: `init()` → `Init() error`
- [ ] 1.8 Convert `pkg/middleware/parallel/parallel.go`: `init()` → `Init() error`
- [ ] 1.9 Convert `pkg/middleware/ratelimit/rate_limiting.go`: `init()` → `Init() error`
- [ ] 1.10 Convert `pkg/middleware/replacepath/replace_path.go`: `init()` → `Init() error`
- [ ] 1.11 Convert `pkg/middleware/replacepathregex/replace_path_regex.go`: `init()` → `Init() error`
- [ ] 1.12 Convert `pkg/middleware/requesttermination/req_termination.go`: `init()` → `Init() error`
- [ ] 1.13 Convert `pkg/middleware/requesttransformer/req_transformer.go`: `init()` → `Init() error`
- [ ] 1.14 Convert `pkg/middleware/responsetransformer/resp_transformer.go`: `init()` → `Init() error`
- [ ] 1.15 Convert `pkg/middleware/setvars/setvars.go`: `init()` → `Init() error`
- [ ] 1.16 Convert `pkg/middleware/stripprefix/strip_prefix.go`: `init()` → `Init() error`
- [ ] 1.17 Convert `pkg/middleware/trafficsplitter/pkg.go`: `init()` → `Init() error`
- [ ] 1.18 Convert `pkg/middleware/uarestriction/ua_restriction.go`: `init()` → `Init() error`

## 2. Balancer Init Functions

- [ ] 2.1 Convert `pkg/balancer/chash/hashing.go`: `init()` → `Init() error`
- [ ] 2.2 Convert `pkg/balancer/random/random.go`: `init()` → `Init() error`
- [ ] 2.3 Convert `pkg/balancer/roundrobin/round_robin.go`: `init()` → `Init() error`
- [ ] 2.4 Convert `pkg/balancer/weighted/weighted.go`: `init()` → `Init() error`

## 3. Initialize Package

- [ ] 3.1 Rewrite `pkg/initialize/pkg.go`: Replace blank imports with explicit `Init()` calls
- [ ] 3.2 Update `Bifrost()` function to call all `Init()` functions and handle errors

## 4. Verification

- [ ] 4.1 Run `go build ./...` to verify compilation
- [ ] 4.2 Run `go test ./...` to verify all tests pass
- [ ] 4.3 Run `make release` to verify no lint and test errors
