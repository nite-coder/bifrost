# AI Gateway Model Cost Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement automatic AI model cost calculation and telemetry tracking in Bifrost AI Gateway, supporting multi-level fallback (Target > External > Embedded).

**Architecture:** Extend the AI proxy to calculate costs based on token usage and configured rates. Use a pricing registry to manage global and per-target pricing, reporting cumulative costs via Prometheus.

**Tech Stack:** Go, Prometheus, JSON, go:embed.

---

## File Structure Changes

- `pkg/config/options.go`: Add `PricingFile` to `AIOptions` and `Pricing` to `AITargetOptions`.
- `pkg/ai/types.go`: Add cost fields to `Usage` and implement `CalculateCost` method.
- `pkg/ai/pricing/prices.json`: Create embedded default pricing data.
- `pkg/ai/pricing/registry.go`: Create pricing registry for resolution and fallback.
- `pkg/telemetry/metrics/ai.go`: Add `AIRequestCost` Prometheus metric.
- `pkg/proxy/ai/proxy.go`: Update to calculate and record costs.
- `pkg/gateway/service.go`: Inject resolved pricing into `AIProxy`.

---

## Tasks

### Task 1: Update Configuration and Data Structures (TDD)

**Files:**
- Modify: `pkg/config/options.go`
- Modify: `pkg/ai/types.go`
- Create: `pkg/ai/types_test.go`

- [ ] **Step 1: Write failing test for `CalculateCost` in `pkg/ai/types_test.go` (RED)**

```go
package ai

import (
	"testing"
	"github.com/nite-coder/bifrost/pkg/config"
)

func TestUsage_CalculateCost(t *testing.T) {
    p := &config.AIPricingOptions{
        InputPerMtok:       2.0,
        OutputPerMtok:      10.0,
        CachedInputPerMtok: 1.0,
    }
    u := &Usage{
        PromptTokens:     1000000,
        CompletionTokens: 1000000,
        PromptTokensDetails: &PromptTokensDetails{
            CachedTokens: 500000,
        },
    }
    u.CalculateCost(p)
    if u.InputCost != 1.5 { // (0.5 * 2) + (0.5 * 1)
        t.Errorf("expected input cost 1.5, got %f", u.InputCost)
    }
    if u.OutputCost != 10.0 {
        t.Errorf("expected output cost 10.0, got %f", u.OutputCost)
    }
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test -v pkg/ai/types_test.go`
Expected: FAIL (CalculateCost undefined or fields missing)

- [ ] **Step 3: Update `AIOptions` and `AITargetOptions` in `pkg/config/options.go`**

```go
type AIOptions struct {
    Providers   map[string]*AIProvider `json:"providers"    yaml:"providers"`
    PricingFile string                 `json:"pricing_file" yaml:"pricing_file"`
}

type AITargetOptions struct {
    Target  string            `json:"target"  yaml:"target"`
    Weight  int               `json:"weight"  yaml:"weight"`
    Pricing *AIPricingOptions `json:"pricing" yaml:"pricing"`
}

type AIPricingOptions struct {
    InputPerMtok       float64 `json:"input_per_mtok"        yaml:"input_per_mtok"`
    OutputPerMtok      float64 `json:"output_per_mtok"       yaml:"output_per_mtok"`
    CachedInputPerMtok float64 `json:"cached_input_per_mtok" yaml:"cached_input_per_mtok"`
}
```

- [ ] **Step 4: Update `Usage` struct and implement `CalculateCost` in `pkg/ai/types.go` (GREEN)**

```go
type Usage struct {
    PromptTokens            int                      `json:"prompt_tokens"`
    CompletionTokens        int                      `json:"completion_tokens"`
    TotalTokens             int                      `json:"total_tokens"`
    PromptTokensDetails     *PromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
    CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
    InputCost               float64                  `json:"input_cost,omitempty"`
    OutputCost              float64                  `json:"output_cost,omitempty"`
}

func (u *Usage) CalculateCost(p *config.AIPricingOptions) {
    if p == nil {
        return
    }
    inputTokens := float64(u.PromptTokens)
    if u.PromptTokensDetails != nil && p.CachedInputPerMtok > 0 {
        cached := float64(u.PromptTokensDetails.CachedTokens)
        u.InputCost = ((inputTokens - cached) / 1000000.0 * p.InputPerMtok) +
            (cached / 1000000.0 * p.CachedInputPerMtok)
    } else {
        u.InputCost = (inputTokens / 1000000.0) * p.InputPerMtok
    }
    u.OutputCost = (float64(u.CompletionTokens) / 1000000.0) * p.OutputPerMtok
}
```

- [ ] **Step 5: Run tests to verify success**

Run: `go test -v ./pkg/ai/...`
Expected: PASS

---

### Task 2: Implement Pricing Registry (TDD)

**Files:**
- Create: `pkg/ai/pricing/prices.json`
- Create: `pkg/ai/pricing/registry.go`
- Create: `pkg/ai/pricing/registry_test.go`

- [ ] **Step 1: Write failing test for Pricing Registry in `pkg/ai/pricing/registry_test.go` (RED)**

```go
package pricing

import (
	"testing"
	"github.com/nite-coder/bifrost/pkg/config"
)

func TestResolve(t *testing.T) {
    Init("")
    p := Resolve("openai-chat", "gpt-4o", nil)
    if p == nil || p.InputPerMtok != 2.50 {
        t.Errorf("failed to resolve embedded gpt-4o price, got %+v", p)
    }
    
    override := &config.AIPricingOptions{InputPerMtok: 1.0}
    p2 := Resolve("openai-chat", "gpt-4o", override)
    if p2.InputPerMtok != 1.0 {
        t.Error("failed to respect override")
    }
}
```

- [ ] **Step 2: Run test to verify failure**

Run: `go test -v ./pkg/ai/pricing/...`
Expected: FAIL (Init/Resolve undefined)

- [ ] **Step 3: Create `pkg/ai/pricing/prices.json`**

```json
{
  "openai-chat/gpt-4o": {
    "input_per_mtok": 2.50,
    "output_per_mtok": 10.00,
    "cached_input_per_mtok": 1.25
  },
  "deepseek-chat": {
    "input_per_mtok": 0.14,
    "output_per_mtok": 0.28,
    "cached_input_per_mtok": 0.014
  }
}
```

- [ ] **Step 4: Implement `pkg/ai/pricing/registry.go` (GREEN)**

```go
package pricing

import (
    _ "embed"
    "encoding/json"
    "os"
    "sync"
    "github.com/nite-coder/bifrost/pkg/config"
)

//go:embed prices.json
var embeddedPrices []byte

var (
    registry map[string]*config.AIPricingOptions
    mu       sync.RWMutex
)

func Init(customPath string) error {
    mu.Lock()
    defer mu.Unlock()
    registry = make(map[string]*config.AIPricingOptions)
    if err := json.Unmarshal(embeddedPrices, &registry); err != nil {
        return err
	}
    if customPath != "" {
        data, err := os.ReadFile(customPath)
        if err == nil {
            var custom map[string]*config.AIPricingOptions
            if err := json.Unmarshal(data, &custom); err == nil {
                for k, v := range custom {
                    registry[k] = v
                }
            }
        }
    }
    return nil
}

func Resolve(handler, model string, override *config.AIPricingOptions) *config.AIPricingOptions {
    if override != nil {
        return override
    }
    mu.RLock()
    defer mu.RUnlock()
    if p, ok := registry[handler+"/"+model]; ok {
        return p
    }
    return registry[model]
}
```

- [ ] **Step 5: Run tests to verify success**

Run: `go test -v ./pkg/ai/pricing/...`
Expected: PASS

---

### Task 3: Update Telemetry Metrics

**Files:**
- Modify: `pkg/telemetry/metrics/ai.go`

- [ ] **Step 1: Add `AIRequestCost` to `pkg/telemetry/metrics/ai.go`**

```go
var (
    // ... existing
    AIRequestCost *prom.CounterVec
)

func InitAI(latencyBuckets, tpsBuckets []float64) {
    // ... existing
    AIRequestCost = prom.NewCounterVec(
        prom.CounterOpts{
            Name: "bifrost_ai_request_cost_usd_total",
            Help: "Total AI request cost in USD",
        },
        []string{"model", "model_id"},
    )
    prom.MustRegister(AIRequestCost)
}
```

---

### Task 4: Integrate Cost Calculation in AI Proxy (TDD)

**Files:**
- Modify: `pkg/proxy/ai/proxy.go`

- [ ] **Step 1: Update `Proxy` struct and integration points**

- Update `Proxy` struct to include `pricing *config.AIPricingOptions`.
- Update `NewProxy` signature.
- Call `resp.Usage.CalculateCost(p.pricing)` in `handleChatUnary`, `handleChatStream` (observer), and `handleResponses`.
- Record `metrics.AIRequestCost.WithLabelValues(virtualModel, modelID).Add(u.InputCost + u.OutputCost)`.

- [ ] **Step 2: Verify with existing AI Gateway tests**

Run: `go test -v ./pkg/proxy/ai/...`
Expected: PASS (Check if tokens/cost are being tracked if possible via test assertions)

---

### Task 5: Inject Pricing in Service Initialization

**Files:**
- Modify: `pkg/gateway/service.go`
- Modify: `pkg/initialize/pkg.go`

- [ ] **Step 1: Initialize Pricing Registry in `pkg/initialize/pkg.go`**
- [ ] **Step 2: Resolve and inject pricing in `loadModels` in `pkg/gateway/service.go`**

---

### Task 6: Review and Final Verification

- [ ] **Step 1: Request Security Review**
  - Invoke a subagent with the `security-review` skill.
  - Provide all modified files for review.
  - Fix any identified security vulnerabilities.
  - Iterate until the subagent provides a "PASS".

- [ ] **Step 2: Request Code Review**
  - Invoke a subagent with the `requesting-code-review` skill.
  - Context: "Implementing AI model cost calculation with multi-level fallback".
  - Fix any identified code quality or idiomatic issues.
  - Iterate until the subagent provides a "PASS".

- [ ] **Step 3: Run all project tests and lint**
  - Run: `make check`
  - Expected: ALL PASS (tests pass and no lint issues)

- [ ] **Step 4: Manual Verification of Metrics**
  - Start Bifrost with AI configuration.
  - Send a few requests.
  - Check `/metrics` endpoint for `bifrost_ai_request_cost_usd_total`.
  - Confirm cost matches tokens * rate.
