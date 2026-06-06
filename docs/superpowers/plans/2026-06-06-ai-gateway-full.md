# AI Gateway Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.
> REQUIRED SUB-SKILL: Use superpowers:test-driven-development for all implementation steps. You MUST write failing tests before writing any production code. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the complete AI Gateway features based on the Phase 1 specification, enabling dynamic routing, multi-protocol translation, and robust streaming observability.

**Architecture:** Hub-and-Spoke architecture utilizing a Canonical `ChatRequest` format. It dynamically converts user-configured `models` into standard Bifrost `Upstream` objects. The request flows through the `ai_transformer` middleware (Ingress) to the `AIProxy` (Egress), which handles blocking I/O and usage telemetry via decorators.

**Tech Stack:** Go, Hertz, Prometheus (client_golang).

---

### Task 1: Core Constants and Canonical Types

**Files:**
- Create/Modify: `pkg/variable/keys.go`
- Modify: `pkg/ai/types.go`
- Create: `pkg/ai/types_test.go`

- [ ] **Step 1 (RED): Write failing tests for `UnknownFields` JSON Logic**
  Create `pkg/ai/types_test.go`. Write tests verifying that `ChatRequest` correctly unmarshals unknown fields into `UnknownFields` and marshals them back into the root level of the JSON output.
- [ ] **Step 2 (Verify RED): Watch tests fail**
  Run: `go test ./pkg/ai/...` and confirm the tests fail because the custom JSON logic is missing.
- [ ] **Step 3 (GREEN): Implement Core Constants and Canonical Types**
  1. Add `AIModelName = "$ai_model_name"` to `pkg/variable/keys.go`.
  2. Implement `UnmarshalJSON` and `MarshalJSON` on `ChatRequest` in `pkg/ai/types.go` using `sonic` to capture and flatten `UnknownFields`.
- [ ] **Step 4 (Verify GREEN): Watch tests pass**
  Run: `go test ./pkg/ai/...` and confirm all tests pass.
- [ ] **Step 5 (REFACTOR): Clean up**
  Refactor test or implementation code if needed.

---

### Task 2: Config Options and Validation

**Files:**
- Modify: `pkg/config/options.go`
- Modify: `pkg/config/validation.go`
- Create/Modify: `pkg/config/validation_test.go`

- [ ] **Step 1 (RED): Write failing tests for AI Config Validation**
  Add test cases in `pkg/config/validation_test.go` to verify validation rules: provider missing handler/base_url, model lacking targets, target with invalid format (`provider/model`), and target referencing unknown provider.
- [ ] **Step 2 (Verify RED): Watch tests fail**
  Run: `go test ./pkg/config/...` and confirm validation tests fail.
- [ ] **Step 3 (GREEN): Implement Config Options and Validation**
  1. Add `AIOptions`, `AIProvider`, `AIModelOptions`, `AIBalancerOptions`, and `AITargetOptions` to `pkg/config/options.go`.
  2. Implement `validateAIConfig` in `pkg/config/validation.go` and call it within `ValidateConfig`.
- [ ] **Step 4 (Verify GREEN): Watch tests pass**
  Run: `go test ./pkg/config/...` and confirm validation tests pass.
- [ ] **Step 5 (REFACTOR): Clean up**
  Ensure code is clean and idiomatic.

---

### Task 3: Automated Model Injection (`loadModels`)

**Files:**
- Modify: `pkg/gateway/service.go`
- Modify: `pkg/gateway/service_test.go`

- [ ] **Step 1 (RED): Write failing tests for `loadModels`**
  Add tests in `pkg/gateway/service_test.go` to verify that when `options.Models` is configured, `loadModels` correctly creates virtual upstreams with the `ai:` prefix and assigns targets/weights properly. Also test `type: ai` handles dynamic upstream correctly.
- [ ] **Step 2 (Verify RED): Watch tests fail**
  Run: `go test ./pkg/gateway/...` and confirm failure.
- [ ] **Step 3 (GREEN): Implement `loadModels` and update `newService`**
  1. Add `loadModels()` method to `*Service` to parse models into upstreams.
  2. Update `newService` to call `loadModels()` and handle `serviceOptions.Type == "ai"` logic (`svc.dynamicUpstream = variable.AIModelName`).
- [ ] **Step 4 (Verify GREEN): Watch tests pass**
  Run: `go test ./pkg/gateway/...` and confirm success.
- [ ] **Step 5 (REFACTOR): Clean up**
  Review target resolving logic for neatness.

---

### Task 4: AIProxy Core Implementation

**Files:**
- Create: `pkg/proxy/ai/proxy.go`
- Create: `pkg/proxy/ai/proxy_test.go`

- [ ] **Step 1 (RED): Write failing tests for AIProxy**
  Create `pkg/proxy/ai/proxy_test.go`. Write tests mocking `LLMAdapter` and `ClientAdapter`. Test unary success, unary error (returns AIError), stream success (verifies chunked write), and stream mid-transmission error.
- [ ] **Step 2 (Verify RED): Watch tests fail**
  Run: `go test ./pkg/proxy/ai/...`
- [ ] **Step 3 (GREEN): Implement AIProxy (`ServeHTTP`, `handleChatUnary`, `handleChatStream`)**
  1. Implement `ServeHTTP` to handle `ContextKeyAIFamily`, virtual model parsing, and routing to unary/stream handlers.
  2. Implement `handleChatUnary` to call adapter, record usage, and write JSON.
  3. Implement `handleChatStream` with a flush loop (`Read chunk -> c.Write -> c.Flush()`), bypassing `io.Copy`.
- [ ] **Step 4 (Verify GREEN): Watch tests pass**
  Run: `go test ./pkg/proxy/ai/...`
- [ ] **Step 5 (REFACTOR): Clean up**
  Ensure error handling aligns with Hertz context best practices.

---

### Task 5: Middleware (`ai_transformer`)

**Files:**
- Create: `pkg/middleware/aitransformer/ai_transformer.go`
- Create: `pkg/middleware/aitransformer/ai_transformer_test.go`

- [ ] **Step 1 (RED): Write failing tests for Ingress/Egress middleware**
  Create `pkg/middleware/aitransformer/ai_transformer_test.go`. Write tests ensuring body is parsed via `ClientAdapter`, context keys (`ContextKeyChatRequest`, `AIModelName`) are set, and errors (`c.Errors`) are intercepted and reformatted into JSON responses.
- [ ] **Step 2 (Verify RED): Watch tests fail**
  Run: `go test ./pkg/middleware/aitransformer/...`
- [ ] **Step 3 (GREEN): Implement Ingress parsing and Context injection**
  Implement the `ServeHTTP` method handling Phase 1 (Ingress parsing to context) and Phase 2 (Egress error interception).
- [ ] **Step 4 (Verify GREEN): Watch tests pass**
  Run: `go test ./pkg/middleware/aitransformer/...`
- [ ] **Step 5 (REFACTOR): Clean up**

---

Plan complete and saved to `docs/superpowers/plans/2026-06-06-ai-gateway-full.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**