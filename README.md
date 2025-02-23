# Bifrost [![GoDoc][doc-img]][doc] [![Build][ci-img]][ci] [![GoReport][report-img]][report] [![Security][security-img]][security] [![Coverage Status][cov-img]][cov]

A low latency and high throughput API Gateway library written in Go.

## Goal

1. Low latency and high throughput.
1. Allows writing middlewares in native `Go`.
1. Easy to integrate with existing Go services.
1. Easy migration from Nginx. Leverage your Nginx experience.
1. Built as an SDK/library instead of an artifact.

## Features

1. Support for writing middleware in `Go`.
1. Low latency and high throughput (powered by the Hertz framework).
1. Millisecond-level hot reloads for route updates without interrupting requests.
1. Get gateway configurations from local files or configuration centers.
1. Built-in Prometheus monitoring.
1. Built-in OpenTelemetry tracing.
1. Supports `HTTP1.1`/`HTTP2`/`H2C`/`Webosocket`/`GRPC` protocols.

## Comparative Analysis

|                                                 | Bifrost | Nginx |
| :---------------------------------------------- | :-----: | :---: |
| SDK mode support for custom your gateway        |   ✅    |  ❌   |
| Middleware support and written in native `Go`   |   ✅    |  ❌   |
| Rich `Go` middleware ecosystem                  |   ✅    |  ❌   |
| Millisecond-level hot reloads for route updates |   ✅    |  ❌   |
| Blue-green deployment for services              |   ✅    |  ❌   |
| High connection pool reuse rate                 |   ✅    |  ❌   |
| Easy integration with existing Go programs      |   ✅    |  ❌   |
| Built-in `Prometheus` monitoring                |   ✅    |  ❌   |
| Built-in `OpenTelemetry` tracing                |   ✅    |  ❌   |
| HTTP2 upstream support                          |   ✅    |  ❌   |
| Multiple configuration providers                |   ✅    |  ❌   |
| Standardized and User-Friendly Configuration    |   ✅    |  ❌   |
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
1. [Providers](./docs/providers.md)
1. [Routes](./docs/routes.md)
1. [Middlewares](./docs/middlewares.md)
1. [Directive](./docs/directive.md)

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
