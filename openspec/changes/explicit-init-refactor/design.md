## Context

The Bifrost gateway uses Go's `init()` functions for middleware and balancer registration. Currently:

- 18 middleware packages use `init()` to call `middleware.RegisterTyped()`
- 4 balancer packages use `init()` to call `balancer.Register()`
- `pkg/initialize/pkg.go` uses blank imports (`_ "..."`) to trigger these `init()` functions
- The `Bifrost()` function is essentially a no-op, just ensuring imports are executed

**Current flow:**
```
import _ "pkg/middleware/addprefix"  →  init() executes  →  RegisterTyped() called (error ignored)
```

## Goals / Non-Goals

**Goals:**
- Replace `init()` with explicit `Init() error` functions in all middleware/balancer packages
- Enable proper error handling during initialization
- Provide explicit, controllable initialization order
- Improve testability by allowing selective initialization
- Maintain backward compatibility for external users

**Non-Goals:**
- Changing the middleware/balancer registration API
- Changing the `middleware.RegisterTyped()` or `balancer.Register()` function signatures
- Supporting dependency injection in `Init()` (future enhancement)

## Decisions

### 1. Function signature: `Init() error`

Each package will export `Init() error` instead of using `init()`.

**Rationale:** Allows callers to handle registration failures explicitly.

**Alternative considered:** `Init(ctx context.Context) error` - Rejected as context is not needed for simple registration.

### 2. Single `Init()` per package

Each middleware/balancer package exports exactly one `Init()` function that registers all handlers for that package.

**Rationale:** Keeps the API simple and matches 1:1 with current `init()` behavior.

### 3. Error aggregation in `pkg/initialize/pkg.go`

`Bifrost()` will call each `Init()` sequentially and return on first error.

**Rationale:** Fail-fast behavior is preferred for initialization. If one middleware fails to register, the gateway should not start.

**Alternative considered:** Collect all errors and return combined - Rejected as partial initialization is not useful.

### 4. Import style: Normal imports (not blank)

```go
// Before
import _ "github.com/nite-coder/bifrost/pkg/middleware/addprefix"

// After  
import "github.com/nite-coder/bifrost/pkg/middleware/addprefix"
```

**Rationale:** Normal imports are required to call `addprefix.Init()`.

## Files to Modify

### Middleware (18 files)
| Package | File |
|---------|------|
| addprefix | `pkg/middleware/addprefix/add_prefix.go` |
| buffering | `pkg/middleware/buffering/buffering.go` |
| compression | `pkg/middleware/compression/gzip.go` |
| coraza | `pkg/middleware/coraza/coraza.go` |
| cors | `pkg/middleware/cors/config.go` |
| iprestriction | `pkg/middleware/iprestriction/ip_restriction.go` |
| mirror | `pkg/middleware/mirror/mirror.go` |
| parallel | `pkg/middleware/parallel/parallel.go` |
| ratelimit | `pkg/middleware/ratelimit/rate_limiting.go` |
| replacepath | `pkg/middleware/replacepath/replace_path.go` |
| replacepathregex | `pkg/middleware/replacepathregex/replace_path_regex.go` |
| requesttermination | `pkg/middleware/requesttermination/req_termination.go` |
| requesttransformer | `pkg/middleware/requesttransformer/req_transformer.go` |
| responsetransformer | `pkg/middleware/responsetransformer/resp_transformer.go` |
| setvars | `pkg/middleware/setvars/setvars.go` |
| stripprefix | `pkg/middleware/stripprefix/strip_prefix.go` |
| trafficsplitter | `pkg/middleware/trafficsplitter/pkg.go` |
| uarestriction | `pkg/middleware/uarestriction/ua_restriction.go` |

### Balancer (4 files)
| Package | File |
|---------|------|
| chash | `pkg/balancer/chash/hashing.go` |
| random | `pkg/balancer/random/random.go` |
| roundrobin | `pkg/balancer/roundrobin/round_robin.go` |
| weighted | `pkg/balancer/weighted/weighted.go` |

### Initialize (1 file)
| Package | File |
|---------|------|
| initialize | `pkg/initialize/pkg.go` |

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| Large number of files to modify (23 files) | Mechanical changes; use consistent pattern |
| Forgetting to call `Init()` for new middleware | Document pattern; consider code generation |
| Breaking external users who import individual middleware | None expected - external API unchanged |

## Migration Pattern

**Before (current):**
```go
// pkg/middleware/addprefix/add_prefix.go
func init() {
    _ = middleware.RegisterTyped([]string{"add_prefix"}, func(cfg Config) (app.HandlerFunc, error) {
        // ...
    })
}
```

**After:**
```go
// pkg/middleware/addprefix/add_prefix.go
func Init() error {
    return middleware.RegisterTyped([]string{"add_prefix"}, func(cfg Config) (app.HandlerFunc, error) {
        // ...
    })
}
```

**pkg/initialize/pkg.go:**
```go
import (
    "github.com/nite-coder/bifrost/pkg/middleware/addprefix"
    // ... other imports
)

func Bifrost() error {
    if err := addprefix.Init(); err != nil {
        return err
    }
    // ... other Init() calls
    return nil
}
```
