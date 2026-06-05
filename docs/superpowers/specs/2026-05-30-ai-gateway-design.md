# AI Gateway on Bifrost — Design Spec

## 1. Overview

在 bifrost gateway 上建構一個 AI Gateway，讓使用者可以透過多種 SDK 協議的接口 (OpenAI Chat / Responses / Anthropic / Gemini) 呼叫多個上遊 LLM provider。

本設計的核心目標是將 **「基礎設施 (Infrastructure)」** 與 **「模型協議 (LLM Protocol)」** 徹底解耦，並對齊 Bifrost 既有的 Proxy/Upstream 設計。

## 2. Architecture

### 2.1 核心架構圖

```
Client Request (SDK)
  │
  ▼
bifrost Server
  │
  ├─ Route Matching
  │
  ├─ Middleware Chain
  │   ├─ ai_transformer (Ingress)  ← 1. 轉譯 Client 格式 → Canonical (ChatRequest / ResponsesRequest)
  │   ├─ ai_auth                   ← 2. 驗證與權限控制
  │   └─ ... (rate_limit, etc.)
  │
  ├─ Service Dispatch (Bifrost Service)
  │   └─ 根據 Model Name 查找對應的 Upstream (帶有 ai: namespace)
  │
  ├─ Upstream (Load Balancer)
  │   └─ 透過 Balancer (預設 weighted) 選擇具體的 AIProxy
  │
  ▼
AIProxy (implements proxy.Proxy)
  ├─ 持有 http.Client (連線池) 與 LLMAdapter
  ├─ 呼叫 LLMAdapter 執行協議對接與執行
  └─ 處理監控、失敗統計、Retry 邏輯
  │
  ▼
LLMAdapter (Functional)
  ├─ 持有 API Key 與 BaseURL
  ├─ 負責 Canonical ↔ Provider Native 雙向轉譯
  └─ 執行 HTTP 請求並回傳結果或 io.ReadCloser (串流)
```

### 2.2 職責分工 (Separation of Concerns)

| 組件 | 職責 | 關鍵方法 |
| :--- | :--- | :--- |
| **ai_transformer** | **Ingress**: 負責 Client ↔ Canonical 轉譯。解析模型名稱並註冊 `variable.AIModelName` (帶有 `ai:` 前綴)。 | `ToInternal()`, `ToClient()` |
| **Bifrost Service** | **Router**: 直接重用 `pkg/gateway/service.go`。透過動態路由尋找對應的 Upstream。 | `ServeHTTP()` |
| **Model (as Upstream)** | **Load Balancer**: 系統啟動時，將 `models` 配置轉換為 Bifrost `Upstream`，並加上 `ai:` Namespace 避免衝突。 | `Balancer().Select()` |
| **AIProxy** | **Infrastructure**: 實作 `proxy.Proxy`，對齊 Bifrost 核心。負責連線池、監控與失敗統計。 | `ServeHTTP()` |
| **LLMAdapter** | **Protocol**: 負責 Canonical ↔ Provider 轉譯。執行對外請求，處理認證與錯誤解析。參考 GoModel 設計。 | `Chat()`, `StreamChat()` |

## 3. Config

### 3.1 AI Config Struct

```go
// pkg/config/options.go
type Options struct {
    AI     *AIOptions                 `json:"ai" yaml:"ai"`
    Models map[string]*AIModelOptions `json:"models" yaml:"models"`
}

type AIOptions struct {
    Providers map[string]*AIProvider `yaml:"providers"`
}

type AIProvider struct {
    Handler string `json:"handler" yaml:"handler"` // "openai-chat" | "openai-responses" | "anthropic" | "gemini"
    BaseURL string `json:"base_url" yaml:"base_url"`
    APIKey  string `json:"api_key" yaml:"api_key"`
}

type AIModelOptions struct {
    Balancer *AIBalancerOptions `yaml:"balancer"`
    Targets  []AITargetOptions  `yaml:"targets"`
}

type AIBalancerOptions struct {
    Type string `yaml:"type"` // 預設為 "weighted" (加權隨機)
}

type AITargetOptions struct {
    Target string `json:"target" yaml:"target"` // "provider_id/actual-model-name" 格式
    Weight int    `json:"weight" yaml:"weight"` // 負載平衡權重，預設 1
}
```

### 3.2 完整 Config 範例

```yaml
services:
  ai_gateway:
    type: ai

ai:
  providers:
    anthropic-us:
      handler: "anthropic"
      base_url: "https://api.anthropic.com"
      api_key: "ANTHROPIC_API_KEY"
    openai-official:
      handler: "openai-chat"
      base_url: "https://api.openai.com/v1"
      api_key: "OPENAI_API_KEY"

models:
  gpt-4o:
    balancer:
      type: "weighted" # 若省略則預設為 weighted
    targets:
      - target: "openai-official/gpt-4o"
        weight: 1
      - target: "anthropic-us/claude-3-5-sonnet-20241022"
        weight: 1
```

## 4. Data Structures (Canonical Types)

所有中樞結構體定義於 **`pkg/ai/types.go`**，移除 `Internal` 前綴以簡化命名。

### 4.1 Chat 家族 (Stateless)
*   `ChatRequest`: 對齊 OpenAI Chat Completion 格式。
*   `ChatResponse`: 統一的對話響應。
*   `StreamChunk`: SSE 串流的中樞格式。

### 4.2 Responses 家族 (Stateful - Phase 1.5)
*   `ResponsesRequest`: 對齊 OpenAI Responses API 格式。
*   `ResponsesResponse`: Responses 家族響應。

### 4.3 Error 處理
*   `AIError`: 統一的錯誤物件，包含 `StatusCode`、`Type`、`Message`。

## 5. LLMAdapter Interface

介面參考 GoModel 設計，走功能導向並解耦傳輸協議。定義於 **`pkg/ai/adapter.go`**。

```go
type LLMAdapter interface {
    Name() string

    // --- Chat 家族 ---
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    StreamChat(ctx context.Context, req *ChatRequest) (io.ReadCloser, error)

    // --- Responses 家族 (Phase 1.5) ---
    Responses(ctx context.Context, req *ResponsesRequest) (*ResponsesResponse, error)
    StreamResponses(ctx context.Context, req *ResponsesRequest) (io.ReadCloser, error)
}
```

## 6. LLM Adapters Implementation

為對應 YAML 配置中的 `handler` 欄位，系統內建四種 `LLMAdapter` 實作。它們透過工廠模式或 Registry 機制（例如 `ai.GetAdapterFactory(handler)`）在 `AIProxy` 初始化時建立，並注入 `http.Client` 與認證資訊。

### 6.1 `openai-chat` (OpenAI Chat Completions)
*   **職責**：處理與 OpenAI 傳統 `/v1/chat/completions` API 以及相容後端（如 DeepSeek, Qwen）的通訊。
*   **轉譯特點**：由於系統中樞格式即為 OpenAI 格式，此 Adapter 的 `ChatRequest` ↔ Native JSON 轉換主要是零成本的序列化（Identity Transformation）。
*   **認證**：使用 `Authorization: Bearer <API_KEY>`。

### 6.2 `openai-responses` (OpenAI Stateful Responses - Phase 1.5)
*   **職責**：處理與 OpenAI 新版 `/v1/responses` API 的通訊。
*   **轉譯特點**：負責發送包含 `instructions`、`input` 等 stateful 欄位的請求，並處理伺服器端管理的對話歷史。
*   **認證**：使用 `Authorization: Bearer <API_KEY>`。

### 6.3 `anthropic` (Anthropic Messages)
*   **職責**：處理與 Anthropic `/v1/messages` API 的通訊。
*   **轉譯特點**：
    *   **System Message**：需將 `ChatRequest` 內部的 `role: "system"` 提取為 Anthropic 頂層的 `system` 欄位。
    *   **Tool Calls**：需在 `tool_use` 與 OpenAI function call 格式之間進行雙向轉換。
    *   **Streaming**：需解析複雜的 Anthropic SSE events (如 `message_start`, `content_block_delta`) 並轉為 Canonical SSE 格式。
*   **認證**：使用 `x-api-key: <API_KEY>` 與 `anthropic-version` header。

### 6.4 `gemini` (Google Gemini)
*   **職責**：處理與 Google Gemini `generateContent` API 的通訊。
*   **轉譯特點**：
    *   **Endpoint**：URL 需動態拼裝模型名稱（例如 `/v1beta/models/<model>:generateContent`）。
    *   **Messages**：需將 `ChatRequest` 的 messages 對映到 Gemini 的 `contents[]` 陣列，並提取系統提示至 `systemInstruction`。
*   **認證**：通常透過 URL parameter `?key=<API_KEY>` 或 `x-goog-api-key` header 傳遞。

## 7. Middleware: ai_transformer

`ai_transformer` 負責所有與「客戶端」對接的格式轉譯。

*   **Ingress (Before Next)**:
    *   根據路徑與參數識別 API 家族 (Chat / Responses)。
    *   轉譯 Client Raw Body ➜ `ChatRequest` 或 `ResponsesRequest`。
    *   將物件存入 Context，並標記 `AIFamily`。
*   **Egress (After Next)**:
    *   從 Context 讀取 `ChatResponse` 並轉譯為 Client 期待的格式。
    *   攔截 `*AIError` 並轉譯為 Client 期待的錯誤格式。

## 8. Initialization & Routing (Namespace Strategy)

為了完美重用 Bifrost 既有的 `pkg/gateway/service.go` 路由與監控邏輯，我們採用 **「隱式 Upstream 轉換」** 與 **「Namespace 隔離」** 策略。

1.  **啟動轉換**：系統載入配置時，會將 YAML 中的 `models` 區塊動態編譯為 Bifrost 標準的 `Upstream` 物件。
2.  **Namespace 隔離**：為了避免 AI 模型與傳統 HTTP 後端發生命名衝突，轉換後的 Upstream 名稱會強制加上 `ai:` 前綴（例如將 `gpt-4o` 存為 `ai:gpt-4o`）。
3.  **動態路由**：`ai_transformer` 攔截到請求後，將模型名稱加上前綴並寫入 Context（例如 `c.Set(variable.AIModelName, "ai:gpt-4o")`）。既有的 `Service.ServeHTTP` 透過動態變數即可無縫命中該 Upstream，無需修改任何核心路由邏輯。

## 9. AIProxy 實作

位於 **`pkg/proxy/ai/proxy.go`**。

### 9.1 核心行為與模型覆寫
1.  從 Context 讀取 `AIFamily` 與 `ChatRequest` / `ResponsesRequest`。
2.  **模型名稱的生命週期區分**：這是一個關鍵的業務邏輯。
    *   **Context 變數 (`variable.AIModelName`)**：始終保持為用戶最初請求的「虛擬模型」（例如 `ai:gpt-4o`）。這確保了後續的日誌、計費與路由語意保持一致，不被竄改。
    *   **Request 結構體 (`req.Model`)**：`AIProxy` 在調用 Adapter 前，**必須**將 `req.Model` 覆寫為從 `p.target` 解析出來的「實體模型名稱」（例如 `claude-3-5-sonnet`），確保上游 API 收到正確的型號。
3.  根據具體家族，調用 `LLMAdapter` 的對應方法（如 `adapter.Chat()`）。
4.  **監控與統計**：紀錄 `UpstreamDuration` 並在發生錯誤時呼叫 `p.AddFailedCount()`。

### 9.2 SSE 串流支援 (Streaming) 與緩衝解析
為解決框架預設的 Response 緩衝問題，確保 SSE 源源不絕輸出，AIProxy 會接管底層 Socket：
1.  呼叫 `adapter.StreamChat()` 取得 `io.ReadCloser`。
2.  設定 `Content-Type: text/event-stream` 與 `Cache-Control: no-cache`。
3.  使用 Hertz 的 `c.Response.HijackWriter` 接管寫入流。
4.  呼叫 `c.Flush()` 送出 Headers。
5.  透過 `io.Copy(c.GetWriter(), stream)` 將轉譯好的 Chunk 直接導向客戶端。

**註：串流穩定性 (Stickiness)**：由於 `Service.ServeHTTP` 的執行生命週期，單次請求在選定 Target 並進入 `AIProxy.ServeHTTP` 後，將與該 Target 保持強綁定直到連線關閉，確保 SSE 數據流的完整性。

### 9.3 邊界條件：未知欄位透傳 (Passthrough)
為保證 Gateway 對新 API 特性的透明度，`ChatRequest` 結構體中設計了 `UnknownFields map[string]any`。在 Ingress 轉譯時會將未映射的欄位收集至此；在 Egress 轉譯時，Adapter 需負責將這些欄位合併回原生的 JSON 中。

## 10. Usage Tracking & Observation

為了實現精確的計費、配額管理與日誌監控，系統採用 **「觀察者模式 (Observer Pattern)」** 來統一處理 Unary 與 Streaming 模式下的用量統計。

### 10.1 UsageMetadata (業務元數據)
定義於 `pkg/ai/types.go`，用於封裝「誰在何處使用了什麼」的上下文資訊。這確保了統計介面在未來新增業務欄位（如 AppID）時無需修改簽名。

### 10.2 UsageObserver 介面
定義於 `pkg/ai/observer.go`，供計費 (Billing) 或日誌 (Logging) 組件實作。
```go
type UsageObserver interface {
    OnUsage(ctx context.Context, metadata UsageMetadata, usage Usage)
}
```

### 10.3 統一統計機制 (The Tracking Flow)
*   **Unary (非串流)**：`AIProxy` 拿到 `ChatResponse` 物件後，直接從中提取 `Usage` 並呼叫 `observer.OnUsage()`。
*   **Streaming (串流)**：採用 **裝飾器模式 (Decorator Pattern)**。`AIProxy` 會使用 `ai.NewObservedStream` 包裝原本的 `io.ReadCloser`。
    *   當資料流過 `Read()` 方法時，包裝器會即時解析每一個 Chunk。
    *   **安全邊界**：網路讀取不保證對齊 SSE 事件邊界（`\n\n`）。實作時**必須**採用 Buffer（如 `bufio.Scanner` 或累積緩衝區）來安全切割事件後，才能進行 JSON 解析。
    *   當解析到最後一個包含 `usage` 欄位的 Chunk 時，自動觸發 `observer.OnUsage()`。

## 11. 修改清單 (Checklist)

| # | 目標 | 檔案路徑 |
|---|---|---|
| 1 | 定義 Canonical 類型 (包含 UnknownFields) 與 UsageMetadata | `pkg/ai/types.go` |
| 2 | 定義 Adapter, Observer 與安全切割的 SSE 觀察裝飾器 | `pkg/ai/adapter.go`, `pkg/ai/observer.go` |
| 3 | 實作 AIProxy (Proxy 介面) 並處理 Model 覆寫與 `coreai` 別名 | `pkg/proxy/ai/proxy.go` |
| 4 | 修改 Service 初始化邏輯 (Models to Upstreams) | `pkg/gateway/service.go`, `pkg/gateway/upstream.go` |
| 5 | 定義全域動態路由變數 `AIModelName` | `pkg/variable/keys.go` |
| 6 | 更新 Ingress Middleware | `pkg/middleware/aitransformer/ai_transformer.go` |
| 7 | 實作具體 Adapters (支援 Passthrough 合併) | `pkg/ai/adapter_openai.go`, `pkg/ai/adapter_anthropic.go` |
| 8 | 註冊組件 | `pkg/initialize/pkg.go` |
