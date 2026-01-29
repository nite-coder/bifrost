# http2-stdlib-server

Go 標準庫 HTTP/2 Server 實現，透過 bridge 層連接到 Hertz Engine。

## ADDED Requirements

### Requirement: Hybrid Server Selection

系統 SHALL 根據配置動態選擇 Server 類型：

- 當 `http2: false` 時，使用 Hertz Server
- 當 `http2: true` 時，使用 Go stdlib `net/http.Server`

#### Scenario: HTTP/1 only mode

- **WHEN** 配置 `http2: false`
- **THEN** 系統使用 Hertz Server 處理請求，無額外延遲開銷

#### Scenario: HTTP/1 + HTTP/2 mode

- **WHEN** 配置 `http2: true`
- **THEN** 系統使用 Go stdlib `net/http.Server` 處理請求

---

### Requirement: Hertz Bridge Request Conversion

系統 SHALL 使用 `adaptor.CopyToHertzRequest` 將 `http.Request` 轉換為 Hertz `RequestContext`。

#### Scenario: HTTP/2 request conversion

- **WHEN** Go stdlib server 接收到 HTTP/2 請求
- **THEN** 系統使用 `adaptor.CopyToHertzRequest` 填充 Hertz `RequestContext`
- **AND** 請求傳遞給 Hertz Engine 的 handler chain

#### Scenario: Headers preserved

- **WHEN** HTTP/2 請求包含任意 headers
- **THEN** 所有 headers 正確傳遞到 Hertz `RequestContext`

---

### Requirement: Hertz Bridge Response Writing

系統 SHALL 將 Hertz `Response` 寫回 `http.ResponseWriter`。

#### Scenario: Response headers and body

- **WHEN** Hertz handler chain 完成處理
- **THEN** 系統將 response headers 複製到 `http.ResponseWriter`
- **AND** 系統將 response body 寫入 `http.ResponseWriter`

#### Scenario: Response status code

- **WHEN** Hertz handler 設置 status code
- **THEN** 該 status code 正確寫入 HTTP/2 response

---

### Requirement: HTTP/2 Protocol Configuration

系統 SHALL 使用 Go 1.24+ 的 `http.Server.Protocols` 配置 HTTP/2。

#### Scenario: HTTP/2 over TLS

- **WHEN** 配置 TLS 且 `http2: true`
- **THEN** Server 支援 HTTP/2 over TLS (h2)

#### Scenario: HTTP/2 over cleartext (h2c)

- **WHEN** 配置 `http2: true` 且未使用 TLS
- **THEN** Server 使用 `SetUnencryptedHTTP2(true)` 支援 h2c

---

### Requirement: Unified Handler Chain

系統 SHALL 統一使用 Hertz `Engine.ServeHTTP` 處理所有請求。

#### Scenario: Middleware execution

- **WHEN** HTTP/2 請求通過 bridge 進入系統
- **THEN** 所有 Hertz middleware 按順序執行

#### Scenario: Existing handlers unchanged

- **WHEN** 用戶已定義 Hertz handlers
- **THEN** 這些 handlers 無需任何修改即可處理 HTTP/2 請求

---

### Requirement: gRPC Proxy Compatibility

系統 SHALL 確保 gRPC Unary 請求能通過 bridge 正確處理。

#### Scenario: gRPC content-type header

- **WHEN** gRPC 請求到達 bridge
- **THEN** `content-type: application/grpc` header 正確傳遞

#### Scenario: gRPC trailer headers

- **WHEN** gRPC 回應包含 trailer headers
- **THEN** `grpc-status` 和 `grpc-message` 正確寫入 HTTP/2 response trailers

#### Scenario: Existing gRPC tests pass

- **WHEN** 執行 `pkg/proxy/grpc/proxy_test.go`
- **THEN** 所有測試通過
