## Context

Bifrost 是一個高效能 API Gateway，使用 Hertz 作為 HTTP/1 Server。當前透過 `hertz-contrib/http2` (fork 版本 `nite-coder/http2`) 支援 HTTP/2，但該套件已停止維護。

**現況**:

- Hertz + netpoll 提供極低延遲的 HTTP/1 處理
- HTTP/2 依賴不維護的 fork
- Go 1.25.4 已內建原生 HTTP/2 支援 (Go 1.24+)

## Goals / Non-Goals

**Goals:**

- 移除 `hertz-contrib/http2` 依賴，使用 Go 標準庫
- HTTP/1 only 模式維持零額外開銷
- 統一使用 Hertz handler chain (所有 middleware 不變)
- 架構可擴展支援 HTTP/3

**Non-Goals:**

- 本次不實現 HTTP/3 (僅確保架構可擴展)
- 不改變用戶配置格式
- 不修改 Hertz 核心程式碼
- **不修改 Hertz Client HTTP/2** - Client 端繼續使用 `hertz-contrib/http2`

## Decisions

### 1. 混合式 Server 架構

**決策**: 根據配置動態選擇 Server 類型

| 配置 | Server | 理由 |
|------|--------|------|
| `http2: false` | Hertz | 保持最佳 HTTP/1 性能 |
| `http2: true` | Go stdlib | 使用原生 HTTP/2 支援 |

**替代方案考慮**:

- ❌ 全部使用 Go stdlib: HTTP/1 會失去 Hertz/netpoll 性能優勢
- ❌ 維護 http2 fork: 長期維護成本高

### 2. Bridge 實現方式

**決策**: 使用 `adaptor.CopyToHertzRequest` + 自定義 Response 寫回

```go
func (b *hertzBridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    reqCtx := pool.Get().(*app.RequestContext)
    defer pool.Put(reqCtx)
    
    adaptor.CopyToHertzRequest(r, &reqCtx.Request)  // 官方 adaptor
    engine.ServeHTTP(ctx, reqCtx)                   // Hertz handler chain
    writeResponse(w, reqCtx)                        // 自定義寫回
}
```

**理由**:

- Request 方向使用官方 adaptor，減少維護
- Response 方向需自定義 (Hertz adaptor 無反向函數)

### 3. HTTP/2 協議配置

**決策**: 使用 Go 1.24+ 的 `http.Server.Protocols`

```go
var protocols http.Protocols
protocols.SetHTTP1(true)
protocols.SetHTTP2(true)
protocols.SetUnencryptedHTTP2(true)  // h2c

server := &http.Server{
    Protocols: &protocols,
    HTTP2:     &http.HTTP2Config{...},
}
```

**理由**: 純標準庫，無需 `golang.org/x/net/http2`

## Risks / Trade-offs

| Risk | Impact | Mitigation |
| ---- | ------ | ---------- |
| HTTP/2 橋接延遲 (+30-60μs) | 中 | 僅影響 HTTP/2 模式; HTTP/1 用戶無影響 |
| Response 寫回邏輯需自行維護 | 低 | 邏輯簡單，僅 header 複製 + body 寫入 |
| Breaking change for http2 users | 中 | 文檔說明遷移步驟 |
| gRPC Proxy 相容性 | 高 | 必須測試驗證 gRPC 請求能正確通過 Bridge |

## Verification Requirements

### gRPC Proxy 測試

必須驗證以下場景:

1. **gRPC Unary 請求**: 單一請求/回應 (Streaming 不支援)
2. **gRPC Headers**: 確保 `content-type: application/grpc` 正確傳遞
3. **gRPC Trailers**: 確保 trailer headers (`grpc-status`, `grpc-message`) 正確處理
4. **現有測試**: `pkg/proxy/grpc/proxy_test.go` 必須全部通過

## Architecture

```
         ┌─────────────┐
         │ Config      │
         │ http2: ?    │
         └──────┬──────┘
                │
     ┌──────────┴──────────┐
     │ http2: false        │ http2: true
     ▼                     ▼
┌─────────┐         ┌────────────────┐
│ Hertz   │         │ net/http.Server│
│ Server  │         │ (HTTP/1+HTTP/2)│
│         │         └───────┬────────┘
│         │                 │
│ Engine ◄├─────────────────┤ hertzBridge
│         │                 │
└─────────┘         ┌───────┴────────┐
                    │ adaptor.Copy   │
                    │ ToHertzRequest │
                    └────────────────┘
```

## Impact Analysis

### gRPC Proxy ✅ 不受影響

gRPC Proxy (`pkg/proxy/grpc/proxy.go`) 使用 `google.golang.org/grpc` client 連接上游:

```go
client, err := grpc.NewClient(addr.Host, grpcOptions...)
```

**工作流程**:

1. HTTP/2 Server 接收 gRPC 請求
2. Bridge 將 `http.Request` 轉為 Hertz `RequestContext`
3. gRPC Proxy handler 從 `RequestContext` 提取 gRPC 資料
4. 使用 `grpc.Client` 轉發到上游

結論: 只要 Bridge 正確傳遞 HTTP/2 headers (包括 gRPC 專用的 `content-type: application/grpc`)，gRPC Proxy 無需修改。

### Hertz Client HTTP/2 ⚠️ 保持不變

HTTP Client (`pkg/proxy/http/client.go`) 往上游發送仍使用 `hertz-contrib/http2`:

```go
c.SetClientFactory(factory.NewClientFactory(http2Config.WithAllowHTTP(true)))
```

**決策**: 本次 (Phase 1) 只遷移 Server 端，Client 端繼續使用 `hertz-contrib/http2`。

**Phase 2 規劃**: 創建自己的 HTTP/2 ClientFactory

```go
// 未來實現方向
type StdlibHTTP2ClientFactory struct {}

func (f *StdlibHTTP2ClientFactory) NewHostClient() (client.HostClient, error) {
    // 使用 Go stdlib http.Transport + HTTP/2
    // 包裝成 Hertz HostClient 介面
}
```

**理由**:

- 分離關注點，降低風險
- Client 需求與 Server 不同
- 保留 Hertz Client 性能優勢，僅替換 HTTP/2 協議層

## Open Questions

1. ~~是否需要 h2c 支援?~~ → 是，使用 `SetUnencryptedHTTP2(true)`
2. HTTP/2 配置參數是否暴露給用戶? → 建議暴露 `max_concurrent_streams` 等基本參數
3. ~~Client HTTP/2 如何處理?~~ → Phase 2 創建自己的 HTTP/2 ClientFactory
