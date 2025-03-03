# Get Started

## Introduction

This guide will teach you how to set up a simple API gateway locally.

Prerequisites:

1. Occupy the local `127.0.0.1:8001` port (HTTP requests will be be sent to here).
1. Assume two upstream service IPs: `127.0.0.1:8000` and `127.0.0.1:80`.

Requirement:

Forward all HTTP requests to the backend upstream services. With two upstream servers, use the `round_robin` algorithm to evenly distribute requests between both upstream services.

## Steps

1. Create a configuration file config.yaml (currently only the yaml format is supported).

  ```yaml
  version: 1
  watch: true

  servers:
    api_server:
      bind: "127.0.0.1:8001"

  routes:
    all_routes:
      paths:
        - /
      service_id: test_service

  services:
    test_service:
      url: "http://test_upstream:8000"

  upstreams:
    test_upstream:
      targets:
        - target: "127.0.0.1:8000"
  ```

2. Create a `main.go` file.

 ```Go
package main

import (
 "github.com/nite-coder/bifrost/pkg/config"
 "github.com/nite-coder/bifrost/pkg/gateway"
 "github.com/nite-coder/bifrost/pkg/initialize"
)

func main() {
 _ = initialize.Middleware()

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

3. Modify your `go.mod` by adding the following two lines.

```sh
replace github.com/cloudwego/hertz v0.9.6 => github.com/nite-coder/hertz v0.9.6
replace github.com/hertz-contrib/http2 v0.1.8 => github.com/nite-coder/http2 v0.1.8
```

4. Run

 ```shell
 go run .
 ```

You will then see that the service is successfully running, and you can send requests to the local port 8001.

 ```sh
 time=2024-08-25T09:54:19.014Z level=INFO msg="starting server" id=api_server bind=127.0.0.1:8001 transporter=netpoll
 time=2024-08-25T09:54:19.015Z level=INFO msg="bifrost started successfully" pid=3667
 ```
