## Context

目前 Bifrost Client 端 HTTP/2 實作依賴 `hertz-contrib/http2`，該庫已停止維護。雖然 Server 端已遷移至 Stdlib，Client 端仍需跟進以統一技術棧。目標是利用 Go 1.24+ `net/http` 原生 HTTP/2 支援來替換舊實作。

## Goals / Non-Goals

**Goals:**

- 實作基於 `net/http` 的 HTTP/2 Client Factory。
- 確保 gRPC Trailers (如 `grpc-status`) 能正確通過 Proxy 傳遞。
- 增加針對 HTTP/2 Upstream 的整合測試。
- 移除 `hertz-contrib/http2` 依賴。

**Non-Goals:**

- 不改變 HTTP/1.1 的 Client 實作 (仍使用 Hertz 預設的 netpoll 實作)。

## Decisions

### 1. 使用 Go 1.24+ `http.Transport`

由於 Go 1.24+ 的 `http.Transport` 已原生並預設支援 HTTP/2 (無需手動配置 `http2.Transport` 除非需特殊調整)，我們將直接使用 `http.Transport` 並配置 TLS 相關參數。

- **配置**: `ForceAttemptHTTP2` 將設為 true (預設值)，並確保 `TLSClientConfig` 正確設置。

### 2. ClientFactory Adapter 實作

我們將實作一個新的 `stdlibFactory`，它需要滿足 Hertz 的 `client.ClientFactory` 介面，並返回滿足 `client.HostClient` 介面的物件。

- **核心邏輯**: `Do` 方法中，使用 `adaptor.GetCompatRequest` 將 Hertz `protocol.Request` 轉為 `http.Request`，執行 `http.Client.Do`，再將 `http.Response` 寫回 Hertz `protocol.Response`。
- **Trailers**: 需特別處理 Response Trailers，從 `http.Response.Trailer` 複製回 Hertz Response Headers 設定為 Trailer。

### 3. 測試策略

由於目前缺乏 HTTP/2 Upstream 測試，我們將在 `pkg/proxy/http` 下新增 `proxy_http2_test.go`。

- **Upstream**: 使用 `net/http` 啟動一個只支援 HTTP/2 的 Test Server (使用 `httptest` + TLS)。
- **驗證點**: 一般 Request/Response、Streaming Body、Trailers (gRPC 模擬)。

## Risks / Trade-offs

- **Performance**: `adaptor` 的轉換可能有微小開銷，但相較於維護性與正確性提升，這是可接受的。且 Server 端已驗證此模式可行。
- **Compatibility**: 需確保 `adaptor` 對於小眾 Header 或特殊行為的兼容性。

## Migration Plan

直接替換 Codebase 中的 Factory 引用，並通過測試驗證。無數據遷移需求。
