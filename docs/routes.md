# Routes

Bifrost's routing design is similar to NGINX, with four main types:

1. Exact match `=`
1. Prefix match `^=`
1. Regular expression match `~`
1. General match

## Match Priority

Exact match > Prefix match > Regular expression match > General match

### Exact Match

The route begins with `=`. In the following example, the request will only be matched if it exactly matches `/spot/orders`.

```yaml
routes:
  testRoute:
    paths:
      - "= /spot/orders"
```

### Prefix Match

Paths that start with `^=` are prefix matches. In this example, any request starting with `/api/v1` will be matched.

```yaml
routes:
  testRoute:
    paths:
      - "^= /api/v1"
    service_id: service1
```

If a request meets two prefix match rules, the longest matching rule takes priority. In the example below, a request GET `/api/v1/orders/12345` will match `testRoute2`.

```yaml
routes:
  testRoute1:
    paths:
      - "^= /api/v1"
    service_id: service1

  testRoute2:
    paths:
      - "^= /api/v1/orders"
    service_id: service2
```

### Regular Expression Match

Paths that begin with `~` denote a regular expression match. In this example, requests with paths like `/api/v2/my_orders` will match. Multiple regex rules follow the order in the configuration file from top to bottom.

```yaml
routes:
  testRoute:
    paths:
      - "~ /api/(v1|v2)"
```

### General Match

In this example, any request starting with `/api/v3/time` will match `testRoute`.

```yaml
routes:
  testRoute:
    paths:
      - "/api/v3"
    service_id: service1
```

If a request matches two prefix rules, the longest rule takes priority. For example, GET `/api/v1/orders/12345` will match `testRoute2`.

```yaml
routes:
  testRoute1:
    paths:
      - /api/v1
    service_id: service1

  testRoute2:
    paths:
      - /api/v1/orders
    service_id: service2
```

## Route Rewriting

Built-in middlewares support common route rewriting.

### Add Prefix

Adds a prefix to the original request path before forwarding upstream.

Original request: `/foo` \
Forwarded upstream: `/api/v1/foo`

```yaml
routes:
  route1:
    paths:
      - /foo
    service_id: service1
    middlewares:
      - type: add_prefix
        params:
          prefix: /api/v1
```

### Strip Prefix

Removes a part of the original request path before forwarding upstream.

Original request: `/api/v1/payment` \
Forwarded upstream: `/payment`

```yaml
routes:
  route1:
    paths:
      - /foo
    service_id: service1
    middlewares:
      - type: strip_prefix
        params:
          prefixes:
            - /api/v1
```

### Replace Path

Replaces the entire original request path with a different path before forwarding upstream. If the original request includes a query string, it will also be forwarded.

Original request: `/api/v1/user?name=john` \
Forwarded upstream: `/hoo/user?name=john`

```yaml
routes:
  route1:
    paths:
      - /api/v1/user
    service_id: service1
    middlewares:
      - type: replace_path_regex
        params:
          regex: ^/api/v1/(.*)$
          replacement: /hoo/$1
```
