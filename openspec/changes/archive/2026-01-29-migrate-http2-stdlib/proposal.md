## Why

`hertz-contrib/http2` 套件已停止維護，Bifrost 目前使用的是 fork 版本 (`nite-coder/http2`)，長期依賴不維護的套件會帶來安全性和相容性風險。Go 1.24+ 標準庫已原生支援 HTTP/2 (包含 h2c)，可完全替代現有方案。

## What Changes

- **Phase 1 (本次)**: Server 端 HTTP/2 遷移至 Go stdlib
- **Phase 2 (未來)**: Client 端 HTTP/2 遷移 - 創建自己的 HTTP/2 ClientFactory
- 新增混合式 Server 架構:
  - `http2: false` → 使用 Hertz Server (零延遲開銷)
  - `http2: true` → 使用 Go stdlib `net/http.Server` + Hertz Bridge
- 使用 `adaptor.CopyToHertzRequest` 進行 Request 轉換
- 統一使用 Hertz `app.HandlerFunc` 作為 Handler (所有 middleware 維持不變)
- 架構可擴展支援 HTTP/3 (quic-go)

## Capabilities

### New Capabilities

- `http2-stdlib-server`: Go 標準庫 HTTP/2 Server 實現，使用 `http.Server.Protocols` 配置 HTTP/1+HTTP/2+h2c，透過 bridge 層連接到 Hertz Engine

### Modified Capabilities

(無需修改現有 spec，這是純實現層面的變更)

## Impact

**Code Changes**:

- `pkg/gateway/server_http.go`: 新增條件邏輯，根據 `http2` 配置選擇 Server 類型
- `pkg/gateway/server_stdlib.go` (新檔案): Go stdlib HTTP Server + Hertz bridge 實現
- `go.mod`: 移除 `hertz-contrib/http2` 和 `replace` 指令

**Dependencies**:

- Phase 1 完成後: `hertz-contrib/http2` 仍保留用於 Client HTTP/2
- Phase 2 完成後: 完全移除 `hertz-contrib/http2`

**Performance**:

- HTTP/1 only: 零影響 (仍使用 Hertz)
- HTTP/1+HTTP/2: 約 +30-60μs 橋接開銷

**Breaking Changes**:

- HTTP/2 實現從 Hertz 原生改為 Go stdlib bridge
- 用戶如有直接使用 `hertz-contrib/http2` 的代碼需要移除
