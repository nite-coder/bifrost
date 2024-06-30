# Get Started

1. 建立一個 `main.go`

```Go
func main() {
	bifrost, err = gateway.LoadFromConfig("./config.yaml")
	if err != nil {
		slog.Error("fail to start bifrost", "error", err)
		return
	}

	bifrost.Run()
}
```

1. 建立一個 config.yaml

```yaml
entries:
  extenal:
    bind: ":8001"

routes:
  hello:
    paths:
      - /spot/orders
    service_id: default

services:
  testServer:
    url: http://localhost:8000

```

1. 運行

```shell
go run .
```
