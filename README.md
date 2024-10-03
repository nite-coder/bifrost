# Bifrost [![GoDoc][doc-img]][doc] [![Build][ci-img]][ci] [![GoReport][report-img]][report] [![Security][security-img]][security] [![Coverage Status][cov-img]][cov]

一個高效能低延遲的 API Gateway 函式庫，使用 Golang 開發

## 動機

1. `Lua` 生態較小, 很多第三方套件功能還不完善
1. 適合 `Go` 比 `Lua` 熟悉的夥伴, Golang 套件與經驗積累比較多
1. 需要把現有的 Golang 服務與網關整合降低請求延遲
1. 容易二次開發加入自訂功能
1. 高性能與低延遲特性

## Features

1. 支持用 `Go` 寫你的中間件 (`servers`, `routes`, `services` 都可以使用很彈性)
1. 高性能與低延遲特性 (底層用的 HTTP 框架是使用字節開源的 `Hertz` 框架, 經過大廠驗證)
1. 修改路由配置後毫秒級熱更新，請求不中斷
1. 內建 `Prometheus` 監控
1. 內建 `OpenTelemetry` 鏈路追蹤
1. 支持 `HTTP1.1`/`HTTP2`/`H2C`/`Webosocket`/`GRPC` 協議

## 參考對比

|                             | Bifrost | Nginx |
| :-------------------------- | :------: | :---: |
| Golang 寫中間件             |      ✅ |  ❌   |
| 套件生態支持豐富            |      ✅ |  ❌   |
| 毫秒級路由熱更新            |      ✅ |  ❌   |
| 新舊服務藍綠部屬            |      ✅ |  ❌   |
| 高連線池複用率              |      ✅ |  ❌   |
| 容易二次開發/整合現有服務   |      ✅ |  ❌   |
| 內建 Prometheus 監控        |      ✅ |  ❌   |
| 內建 OpenTelemetry 鏈路追蹤 |      ✅ |  ❌   |
| 上游 HTTP2 協議支持         |      ✅ |  ❌   |
| 高效能低延遲                |      ✅ |  ✅   |
| GRPC Load Balancer          |      ✅ |  ✅   |

## 技術文件

![flow](/docs/images/bifrost_arch.png)

`servers`: 服務器組態，支持 `middlwares` 使用，用來控制哪個端口需要對外等 \
`routes`: 路由組態，用來控制請求路徑的轉發規則到哪一個 `service` 上 \
`services`: 業務服務組態，用來控制服務配置，例如協議資訊等 \
`upstreams`: 上游組態，用來控制後端主機的負載均衡規則等

## Roadmap

1. support proxy protcol

## Credit

1. [CloudWeGo](https://www.cloudwego.io/)

[doc-img]: https://godoc.org/github.com/nite-coder/bifrost?status.svg
[doc]: https://pkg.go.dev/github.com/nite-coder/bifrost?tab=doc
[ci-img]: https://github.com/nite-coder/bifrost/actions/workflows/build.yml/badge.svg
[ci]: https://github.com/nite-coder/bifrost/actions
[report-img]: https://goreportcard.com/badge/github.com/nite-coder/bifrost
[report]: https://goreportcard.com/report/github.com/nite-coder/bifrost
[security-img]: https://github.com/nite-coder/bifrost/actions/workflows/codeql-analysis.yml/badge.svg
[security]: https://github.com/nite-coder/bifrost/security
[cov-img]: https://codecov.io/github/nite-coder/bifrost/graph/badge.svg
[cov]: https://codecov.io/github/nite-coder/bifrost
