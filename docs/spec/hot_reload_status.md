# Hot Reload 狀態報告

## 目前狀態
**已修復 (Fixed)**

## 1. 觀察到的問題 (Observed Issue)
在執行 `systemctl reload bifrost` 觸發熱重載時，新啟動的 Worker 進程無法順利接手，並在日誌中出現以下錯誤：
```
bind: address already in use
```
這導致新 Worker 啟動失敗或退出，舊 Worker 繼續運行（或在此之前已停止導致服務中斷）。

## 2. 原因分析 (Root Cause)
1.  **FD 傳遞成功但缺乏元數據**：Master 進程成功將監聽 Socket 的 File Descriptors (FDs) 透過 Unix Domain Socket 傳遞給了新 Worker。
2.  **映射失敗**：新 Worker 雖然收到了 FDs（如 FD 3, 4），但 `ZeroDownTime` 庫在初始化監聽器時，缺乏足夠的資訊（Metadata）來將這些 FD 對應到具體的監聯地址（如 `127.0.0.1:8080`）。
3.  **錯誤的行為**：由於無法識別繼承的 FDs，`ZeroDownTime` 判定為「非繼承模式」或「找不到對應的繼承 Socket」，因此嘗試重新呼叫 `net.Listen` 綁定端口。
4.  **端口衝突**：由於舊 Worker 尚未完全退出且仍佔用該端口（或 Socket 選項限制），導致重新綁定失敗，拋出 "Address already in use"。

## 3. 修復方案 (Solution)
我們實作了一套新的 **元數據傳輸機制 (Metadata Transfer Protocol)**：

1.  **Master 端修正 (`pkg/zero/master.go`)**:
    - 在傳遞 FDs 的同時，將對應的 Listener Keys (地址字串) 編碼為 JSON 並進行 Base64 加密。
    - 透過環境變數 `BIFROST_LISTENER_KEYS` 將此編碼後的字串傳遞給新 Worker 進程。

2.  **Control Plane 協議升級 (`pkg/zero/controlplane.go`)**:
    - 更新 `ControlMessage` 結構，增加 `Payload` 欄位以支援未來傳輸更多元數據。
    - 修正了 `SendFDs` 介面以支援傳遞 Keys。

3.  **Worker 端修正 (`pkg/zero/worker_fd.go`)**:
    - 實作 `InheritedListeners()` 函式。
    - 啟動時讀取 `BIFROST_LISTENER_KEYS`，解析出 Keys 列表。
    - 依序將 Keys 與繼承的 FDs (從 FD 3 開始) 進行一對一映射，建立 `Key -> File` 的對照表。

4.  **ZeroDownTime 整合 (`pkg/zero/zero.go`)**:
    - 修改 `Listener()` 方法，優先呼叫 `InheritedListeners()`。
    - 如果存在繼承的 FD，直接使用 `net.FileListener(f)` 恢復監聽器，**不再嘗試重新 Bind 端口**。

## 4. 驗證結果 (Verification)
1.  **Systemd Reload**:
    - 執行 `systemctl reload bifrost` 成功。
    - 日誌顯示：`successfully received transferred FDs count=1 keys=1`。
    - 新 Worker 成功啟動並接手流量 (`worker ready`)，舊 Worker 優雅退出。
2.  **單元測試 (Unit Tests)**:
    - 修正了 `pkg/zero` 下 `controlplane_test.go` 與 `master_test.go` 的簽名錯誤，測試通過。
    - 新增 `worker_fd_test.go`，驗證 `BIFROST_LISTENER_KEYS` 解析邏輯正確。
    - 修正 `pkg/resolver` 測試以支援無網路環境，確保 `make test` 通過。

## 5. 結論
Hot Reload 機制現已修復並通過驗證，"Address already in use" 問題已解決。

---

## 6. Code Review (2026-01-05)

**評審者**: 資深網關架構專家  
**評審範圍**: Master-Worker 熱更新實作 (`pkg/zero/`, `server/bifrost/main.go`, `init/systemd/bifrost.service`)

---

### 6.1 評審結果: ✅ **通過 (Approved with Minor Recommendations)**

整體實作品質優秀，符合規格書設計，核心功能正確實現。以下為詳細分析：

---

### 6.2 優點 (Strengths)

| 項目 | 評價 |
|------|------|
| **FD 傳遞機制** | ✅ 使用 `SCM_RIGHTS` + `Sendmsg/Recvmsg`，符合 Unix 最佳實踐 |
| **元數據傳輸** | ✅ `BIFROST_LISTENER_KEYS` 解決了 FD 與地址映射的關鍵問題 |
| **Abstract Namespace UDS** | ✅ 避免文件系統競態問題，可靠性高 |
| **Pipe 同步 Daemon** | ✅ `daemon.go` 正確實現 Parent-Child Pipe 同步，解決 Systemd Race Condition |
| **KeepAlive 策略** | ✅ 指數退避 + 頻率限制設計合理，防止 Restart Storm |
| **測試覆蓋** | ✅ 所有 `pkg/zero/` 測試通過 (`go test -race`)，包含併發測試 |
| **日誌聚合** | ✅ Worker 繼承 Master 的 stdout/stderr，零損耗 FD 繼承 |
| **Systemd 整合** | ✅ `Type=forking` + PIDFile 設計正確 |

---



### 6.4 安全性檢查 (Security)

| 項目 | 狀態 |
|------|------|
| FD 洩漏 | ✅ Go 1.12+ 自動設置 `CLOEXEC`，`handleFDTransfer` 中正確關閉未使用的 FD |
| 權限 | ✅ PID 文件使用 0644，Lock 文件使用 0600 |
| 信號處理 | ✅ 正確使用 `signal.Notify` 並 `defer signal.Stop` |
| 殭屍進程 | ✅ 使用 `cmd.Wait()` 收割子進程 |

---

### 6.5 與規格書對照 (Spec Compliance)

| 規格要求 | 實作狀態 |
|----------|----------|
| PID 恆定 (Master) | ✅ 完成 |
| CLI: `./bifrost` (前台) | ✅ 完成 |
| CLI: `./bifrost -m -d` (後台 Master) | ✅ 完成 |
| CLI: `-u` / `-s` 控制命令 | ✅ 完成 |
| UDS Abstract Namespace | ✅ 完成 |
| FD 傳遞 via `ExtraFiles` | ✅ 完成 |
| Pipe 同步 Daemon | ✅ 完成 |
| KeepAlive 重啟策略 | ✅ 完成 |
| 日誌 FD 繼承 | ✅ 完成 |

---

### 6.6 結論

**Code Review 結果: ✅ 通過**

實作完整且正確，符合 `hot_reload.md` 規格書設計。所有單元測試通過，包含 Race Condition 檢測。所有 Code Review 建議事項皆已修復。

---

*Review Date: 2026-01-05*  
*Reviewer: Gateway Architecture Expert*
