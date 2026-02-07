## Why

The current middleware and balancer initialization uses Go's `init()` functions, which has several issues:
1. **No error handling**: `init()` cannot return errors; existing code silently ignores errors with `_ = middleware.RegisterTyped(...)`
2. **Uncontrollable execution order**: Depends on Go compiler's import order
3. **Poor testability**: Executes automatically, cannot be controlled in tests
4. **No conditional initialization**: Cannot selectively load based on configuration

## What Changes

- Convert each middleware and balancer `init()` function to an exported `Init() error` function
- Modify `pkg/initialize/pkg.go`'s `Bifrost()` function to explicitly call each `Init()` and handle errors
- Remove blank imports (`_ "..."`), use normal imports instead

## Capabilities

### New Capabilities

- `explicit-initialization`: All middleware and balancer use explicit `Init() error` functions for registration, supporting error handling and controllable initialization order

### Modified Capabilities

_None_ - This is an internal refactoring that does not change any external behavior or API

## Impact

- **Files affected**: ~20+ middleware/balancer files need `init()` → `Init()` modification
- **`pkg/initialize/pkg.go`**: Complete rewrite from blank imports to explicit initialization calls
- **Backward compatibility**: Transparent to external users as long as they continue using `initialize.Bifrost()`
- **Risk**: Low risk, mainly mechanical code changes
