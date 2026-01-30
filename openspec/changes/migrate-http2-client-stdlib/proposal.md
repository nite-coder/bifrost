## Why

`hertz-contrib/http2` 套件已停止維護，Bifrost 目前使用的 Client 端仍依賴此套件 (及其 fork)。為了確保像 Server 端一樣的安全性、穩定性和可維護性，我們需要將 Client 端的 HTTP/2 實作也遷移至 Go 標準庫 (`net/http`)。此外，目前 gRPC Proxy 缺乏針對 HTTP/2 Upstream 的整合測試，這是一個潛在的風險點。

## What Changes

- **新增 HTTP/2 ClientFactory**: 實作一個遵循 Hertz `client.HostClient` 介面的 Adapter，底層使用 Go 標準庫 `http.Client` 配 `http2.Transport` (或 Go 1.24+ `http.Transport`)。
- **替換 HTTP/2 Client**: 修改 `pkg/proxy/http/client.go`，將原有的 `hertz-contrib/http2` Factory 替換為新實作。
- **新增整合測試**: 在 `pkg/proxy/http` 或 `pkg/gateway` 中新增針對 HTTP/2 Upstream 的整合測試，驗證 Proxy 行為 (包含 gRPC)。
- **移除依賴**: 完成後從 `go.mod` 中徹底移除 `hertz-contrib/http2` 和 `nite-coder/http2`。

## Capabilities

### New Capabilities

- `http2-stdlib-client`: 新增基於 Go 標準庫的 Hertz Client Factory 實作，支援 HTTP/2 轉發。

### Modified Capabilities

(無，這是纯實現層面的變更，替換現有的 HTTP/2 實現)

## Impact

- **Code Changes**:
  - `pkg/proxy/http/client.go`: 使用新 Factory。
  - `pkg/common/adaptor` (可能需要): 如果需要更多 Request/Response 轉換輔助。
  - `pkg/proxy/http/http2_test.go` (新增): 針對 HTTP/2 Upstream 的測試。
- **Dependencies**: 移除 `hertz-contrib/http2`。
- **Performance**: 預期與 Server 端遷移類似，會有微小的轉換開銷，但獲得穩定性與標準庫支援。
