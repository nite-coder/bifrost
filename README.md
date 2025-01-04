# Bifrost [![GoDoc][doc-img]][doc] [![Build][ci-img]][ci] [![GoReport][report-img]][report] [![Security][security-img]][security] [![Coverage Status][cov-img]][cov]

A high-performance, low-latency API Gateway library developed in Golang.

## Motivation

1. `Lua` is more suitable for writing simple business logic. However, if you encounter complex scenarios that require asynchronous processing, unit testing, etc., `Lua` might not be the best choice.
1. Ideal for people who are more familiar with `Go` development.
1. Integrating existing Go services with the gateway can reduce request latency.
1. Easy to extend with custom features for secondary development.
1. Designed for high performance and low latency.

## Features

1. Support for writing middleware in `Go`.
1. High performance and low latency (powered by the Hertz framework).
1. Millisecond-level hot reloads for route updates without interrupting requests.
1. Built-in Prometheus monitoring.
1. Built-in OpenTelemetry tracing.
1. Supports `HTTP1.1`/`HTTP2`/`H2C`/`Webosocket`/`GRPC` protocols.

## Comparative Analysis

|                                                 | Bifrost | Nginx |
| :---------------------------------------------- | :-----: | :---: |
| SDK mode support for custom your gateway        |   ✅    |  ❌   |
| Middleware support                              |   ✅    |  ❌   |
| Middleware written in `Go`                      |   ✅    |  ❌   |
| Rich middleware ecosystem                       |   ✅    |  ❌   |
| Millisecond-level hot reloads for route updates |   ✅    |  ❌   |
| Blue-green deployment for services              |   ✅    |  ❌   |
| High connection pool reuse rate                 |   ✅    |  ❌   |
| Easy integration with existing Go programs      |   ✅    |  ❌   |
| Built-in `Prometheus` monitoring                |   ✅    |  ❌   |
| Built-in `OpenTelemetry` tracing                |   ✅    |  ❌   |
| HTTP2 upstream support                          |   ✅    |  ❌   |
| High concurrency and low latency                |   ✅    |  ✅   |
| GRPC Load Balancer                              |   ✅    |  ✅   |
| Sticky session                                  |   ✅    |  ✅   |
| Background task support                         |   ✅    |  ✅   |

## Overview

![flow](/docs/images/bifrost_arch.png)

`servers`: Server configuration, supports middlewares, controlling which port to expose \
`routes`: Route configuration, controls request path forwarding rules to specific services \
`services`: Service configuration, controls service details such as protocol information \
`upstreams`: Upstream configuration, manages load balancing rules for backend hosts

## Get Started

[Set up a high-performance API gateway in 5 minutes](/docs/get_started.md)

## Documents

1. [Configuration](./docs/configuration.md)
1. [Routes](./docs/routes.md)
1. [Middlewares](./docs/middlewares.md)

## Roadmap

1. Cluster management support.

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
