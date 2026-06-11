# Spec & Plan Review: Service/Upstream Refactoring (v2)

**Reviewer**: Spec Review
**Reviewed docs**:
- `docs/superpowers/specs/2026-06-11-service-upstream-refactoring-design-v2.md`
- `docs/superpowers/plans/2026-06-11-service-upstream-refactoring.md`
**Date**: 2026-06-11 (updated review)
**Verdict**: APPROVED — 上一輪的 critical/moderate 問題已全數修復；2 個新 minor observations 供實作時注意。

---

## v2 變更摘要

與上一輪 review 比較，以下問題已在 v2 修復：

| 問題 | 狀態 |
|------|------|
| **CRITICAL**: `proxyByAddress` data race | ✅ 改用 `sync.Map`，熱路徑 `Load()` 無鎖 |
| **MODERATE**: Nacos/K8s Watch push→poll | ✅ Plan 明確要求 wrap instances 到 `DiscoveryResult`，保留 push 效率 |
| **MODERATE**: Broadcast drain 邏輯複雜 | ✅ 改為 simple drop（buffer=64 吸收併發，滿了就丟並 warn） |
| **MINOR**: `Subscribe()` 初始推送時序依賴 | ✅ 改用 `Endpoints()` 同步初始獲取，`Subscribe()` 不再發送初始狀態 |
| **MINOR**: Dead code error path | ⚠️ Spec 已改為 `return nil`，但 plan 程式碼中仍留著 `return fmt.Errorf(...)`（不影響執行，因為該路徑不會被走到） |

---

## 設計正確性確認

核心設計方向一致且正確：

1. **Upstream 統一管理 balancer + 健康狀態** — 解決重複 watcher 和碎片化健康狀態
2. **`sync.Map` 解決 data race** — `Load()` 熱路徑無鎖，`Store()`/`LoadAndDelete()` 在 update path 安全並發
3. **`Endpoints()` + `Subscribe()` 分離** — 初始狀態同步獲取，後續更新異步推送，無時序依賴
4. **Nacos/K8s 保留 push 模式** — 不在 Watch 中送 nil，而是包裝成 `DiscoveryResult` 推送
5. **Simple drop broadcast** — 比舊的 drain 邏輯更簡潔，buffer=64 在實務上足夠

---

## 新發現的 Minor Observations

### 1. 靜態 upstream 的 service 會為所有 upstream 建立 proxy（行為變化）

**位置**：Plan Task 6 Step 3, `newService()` 初始化

**現狀**：舊程式碼中，靜態 upstream 的 service 只訂閱**自己對應的那一個** upstream；動態 upstream（`$variable`）的 service 才訂閱全部 upstream。

**v2 plan 行為**：
```go
// 初始化時對 ALL upstreams 呼叫 updateEndpoints，建立所有 proxy
for _, u := range svc.bifrost.upstreamManager.List() {
    svc.updateEndpoints(u.options.ID, u.Endpoints())
}
```

這表示一個只使用單一 upstream 的靜態 service，也會為所有其他 upstream 建立 HTTP client / gRPC connection。proxy 不會被實際用到（只有 `ServeHTTP` 的 address lookup 會命中相關的 proxy），但會佔用記憶體和連線資源。

**建議**：實作時保留舊的行為分支 — 靜態 service 只初始化自己的 upstream，動態 service 才初始化全部。或者如果刻意要簡化程式碼，至少在 spec 中標註這個取捨。

### 2. `UpstreamManager.List()` 方法不存在

Plan 中多處使用 `s.bifrost.upstreamManager.List()` 來迭代所有 upstream，但當前 `UpstreamManager` 只有 `Get(id)` 方法。實作時需要新增 `List()` 方法（返回 `[]*Upstream`）。

---

## 已確認的設計優點（保持不變）

1. **Balancer 移到 Upstream** — 集中管理 per-upstream balancer 狀態，是共享健康狀態的關鍵
2. **`EndpointHash` 變更檢測** — 避免不必要的 balancer 重建和廣播，排除 State 是正確的（健康狀態變化不應觸發重建）
3. **`DiscoveryResult` 預分組** — 消除 `groupByTargetName()` 需求，provider 知道分組，upstream 只管處理
4. **`pkg/target` 解耦** — balancer 不再 import proxy，打破循環依賴
5. **向後相容** — config 不變，行為不變（weight、proxy lifecycle、health check 語義一致）
6. **TDD 嚴謹** — 每個步驟先寫測試再實作，覆蓋全面

---

## 總結

| Severity | Issue | Status |
|----------|-------|--------|
| ~~CRITICAL~~ | Data race `proxyByAddress` | ✅ Fixed: `sync.Map` |
| ~~MODERATE~~ | Nacos/K8s Watch push→poll | ✅ Fixed: wrap in `DiscoveryResult` |
| ~~MODERATE~~ | Broadcast drain complexity | ✅ Simplified: simple drop |
| ~~MINOR~~ | Subscribe initial timing | ✅ Fixed: `Endpoints()` sync fetch |
| ~~MINOR~~ | Dead code error path | ⚠️ Plan code mismatch with spec (harmless) |
| MINOR (new) | Static service builds all proxies | 實作時注意行為取捨 |
| MINOR (new) | `List()` method missing | 實作時新增 |

**最終結論**：設計方向正確，上一輪的 critical 和 moderate 問題已全數修復。可以進入實作階段。
