## ADDED Requirements

### Requirement: Middleware packages SHALL export Init function

Each middleware package SHALL export a function `Init() error` that registers the middleware handler(s) for that package.

#### Scenario: Successful middleware registration
- **WHEN** `Init()` is called on a middleware package
- **THEN** the middleware handler SHALL be registered with `middleware.RegisterTyped()`
- **AND** the function SHALL return `nil`

#### Scenario: Registration failure
- **WHEN** `Init()` is called and registration fails (e.g., duplicate name)
- **THEN** the function SHALL return the error from `RegisterTyped()`

---

### Requirement: Balancer packages SHALL export Init function

Each balancer package SHALL export a function `Init() error` that registers the balancer for that package.

#### Scenario: Successful balancer registration
- **WHEN** `Init()` is called on a balancer package
- **THEN** the balancer SHALL be registered with `balancer.Register()`
- **AND** the function SHALL return `nil`

#### Scenario: Registration failure
- **WHEN** `Init()` is called and registration fails
- **THEN** the function SHALL return the error from `Register()`

---

### Requirement: Bifrost function SHALL call all Init functions

The `initialize.Bifrost()` function SHALL explicitly call `Init()` on all middleware and balancer packages.

#### Scenario: All registrations succeed
- **WHEN** `Bifrost()` is called
- **AND** all `Init()` calls succeed
- **THEN** `Bifrost()` SHALL return `nil`

#### Scenario: Any registration fails
- **WHEN** `Bifrost()` is called
- **AND** any `Init()` call returns an error
- **THEN** `Bifrost()` SHALL return that error immediately (fail-fast)

---

### Requirement: Init functions SHALL not use init()

Packages SHALL NOT use Go's `init()` function for middleware/balancer registration.

#### Scenario: No automatic registration on import
- **WHEN** a middleware or balancer package is imported
- **THEN** no handlers SHALL be registered automatically
- **AND** registration SHALL only occur when `Init()` is explicitly called

---

### Requirement: Init function SHALL be idempotent

Calling `Init()` multiple times on the same package SHALL NOT cause duplicate registrations.

#### Scenario: Duplicate Init calls
- **WHEN** `Init()` is called twice on the same package
- **THEN** the second call SHALL return an error indicating the handler already exists
