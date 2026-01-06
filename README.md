# Bifrost [![GoDoc][doc-img]][doc] [![Build][ci-img]][ci] [![GoReport][report-img]][report] [![Security][security-img]][security] [![Coverage Status][cov-img]][cov] [![CodeWiki][codewiki-img]][codewiki]

A low-latency and high-throughput API gateway library in Go

## Reasons to use Bifrost

1. Low latency and high throughput.
1. Allows writing middlewares in native `Go`.
1. Easy to integrate with existing Go services.
1. Easy migration from Nginx. Leverage your Nginx experience.
1. Built as an SDK/library instead of an artifact.

## Feature highlights

1. Support for writing middleware in `Go`.
1. Low latency and high throughput (powered by the Hertz framework).
1. Millisecond-level graceful reload for route updates without interrupting requests.
1. Get gateway configurations from local files or configuration centers.
1. Built-in Prometheus monitoring.
1. Built-in OpenTelemetry tracing.
1. Supports `HTTP1.1`/`HTTP2`/`H2C`/`Webosocket`/`GRPC` protocols.
1. Multiple built-in balancer algorithm (`round_robin`, `random`, `weighted`, `chash`) and allow to build your own

## Comparative Analysis

|                                                                        | Bifrost | Nginx |
| :--------------------------------------------------------------------- | :-----: | :---: |
| SDK for building your own gateway                                      |   ✅    |  ❌   |
| Write your middlewares in `native Go` (not `TinyGo`)                   |   ✅    |  ❌   |
| Rich built-in middlewares                                              |   ✅    |  ❌   |
| Millisecond-level graceful reloads for `dynamic configuration` updates |   ✅    |  ❌   |
| Blue-green deployment for services                                     |   ✅    |  ❌   |
| High connection pool reuse rate                                        |   ✅    |  ❌   |
| Easy integration with existing Go programs                             |   ✅    |  ❌   |
| Built-in `Prometheus` monitoring                                       |   ✅    |  ❌   |
| Built-in `OpenTelemetry` tracing                                       |   ✅    |  ❌   |
| Native `k8s` service discovery                                         |   ✅    |  ❌   |
| Built-in web application firewall and support OWASP Core Rule Set      |   ✅    |  ❌   |
| HTTP2 upstream support                                                 |   ✅    |  ❌   |
| Multiple configuration providers                                       |   ✅    |  ❌   |
| Standardized and User-Friendly Configuration                           |   ✅    |  ❌   |
| Multiple built-in balancer algorithm and allow to build your own       |   ✅    |  ❌   |
| High concurrency and low latency                                       |   ✅    |  ✅   |
| GRPC Load Balancer                                                     |   ✅    |  ✅   |
| Background task support                                                |   ✅    |  ✅   |

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
[codewiki-img]: https://www.gstatic.com/_/boq-sdlc-agents-ui/_/r/YUi5dj2UWvE.svg
[codewiki]: https://codewiki.google/github.com/nite-coder/bifrost
