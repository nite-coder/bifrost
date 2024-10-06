# Get Started

## 介紹

將在本地建立一個 api gateway 端口使用 `8001`，把全部的 HTTP 請求轉發到後端的上游服務, 假設上游服務有兩台主機, 分別是 `127.0.0.1:8000`, `127.0.0.1:80`, 將採用 `round_robin` 分流算法把請求平均分到兩台的上游服務

這邊有兩種配置模式，一種是使用配置文件 (目前只支持 `yaml`)，另一種是使用代碼模式

## 配置文件模式

1. 建立一個配置文件 `config.yaml`

	```yaml
	providers:
	  file:
		enabled: true
		extensions:
		- ".yaml"
		- ".yml"
		- ".json"
		paths:
		- "."
		watch: true

	servers:
	  api-server:
		bind: ":8001"

	routes:
	  test-route:
		paths:
		- /
		service_id: test-service

	services:
	  test-service:
		url: "http://test-upstream:8000"

	upstreams:
	  test-upstream:
		targets:
		- target: "127.0.0.1:8000"
	```

1. 建立一個 `main.go`

	```Go
	func main() {
		options, err := config.Load("./config.yaml")
		if err != nil {
			panic(err)
		}

		err = gateway.Run(options)
		if err != nil {
			panic(err)
		}
	}
	```
1. 執行

	```shell
	go run .
	```
	您將看到，這樣表示服務已成功運行, 你可以發送請求到本地端口 `8001`
	```shell
	time=2024-08-25T09:54:19.014Z level=INFO msg="starting server" id=test_server bind=:8001
	time=2024-08-25T09:54:19.015Z level=INFO msg="bifrost started successfully" pid=5825
	```


## 代碼模式

1. 建立一個 `main.go`

```Go
func main() {

  options := config.NewOptions()

 // setup upstream
 options.Upstreams["test_upstream"] = config.UpstreamOptions{
  Strategy: config.RoundRobinStrategy,
  Targets: []config.TargetOptions{
   {
    Target: "127.0.0.1:8000",
   },
   {
    Target: "127.0.0.1:80",
   },
  },
 }

 // setup service
 options.Services["test_service"] = config.ServiceOptions{
  Url: "http://test_upstream:8000",
 }

 // setup route
 options.Routes["all_routes"] = config.RouteOptions{
  Paths: []string{
   "/",
  },
  ServiceID: "test_service",
 }

 // setup server
 options.Servers["api_server"] = config.ServerOptions{
  Bind: ":8001",
 }

 err := gateway.Run(options)
 if err != nil {
  panic(err)
 }
}
```
2. 執行

```shell
go run .
```
您將看到，這樣表示服務已成功運行, 你可以發送請求到本地端口 `8001`
```shell
time=2024-08-25T09:54:19.014Z level=INFO msg="starting server" id=api_server bind=:8001 transporter=netpoll
time=2024-08-25T09:54:19.015Z level=INFO msg="bifrost started successfully" pid=5825
```