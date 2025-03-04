# Routes

Bifrost's routing rule is similar to NGINX, with four main types:

1. Exact match `=`: Highest priority, matches the URI exactly
1. Preferential prefix match `^~`: If a match is found, stop searching
1. Regular expression match `~` (case sensitive):
1. Regular expression match `~*` (case insensitive):
1. Preifx match `/{your_prefix}`:

## Match Priority

Exact match > Prefix match > Regular expression match > General match

### Exact Match

A path that begins with `=` is an exact match. In the following example, the request will be matched only if it exactly equals `/spot/orders`.

```yaml
routes:
  testRoute:
    paths:
      - "= /spot/orders"
```

### Preferential prefix Match

A path that starts with `^~` is a prefix match. In this example, any request starting with `/api/v1` will be matched.

```yaml
routes:
  testRoute:
    paths:
      - "^~ /api/v1"
    service_id: service1
```

If a request matches two prefix rules, the longest matching rule takes priority. In the example below, a `GET /api/v1/orders/12345` request will match `testRoute2`.

```yaml
routes:
  testRoute1:
    paths:
      - "^~ /api/v1"
    service_id: service1

  testRoute2:
    paths:
      - "^~ /api/v1/orders"
    service_id: service2
```

### Regular Expression Match

A path that begins with `~` denotes a regular expression match (case-sensitive). In this example, requests with paths like `/api/v2/my_orders` will match. When multiple regex rules are defined, they are evaluated in the order they appear in the configuration file, from top to bottom.

```yaml
routes:
  testRoute:
    paths:
      - "~ /api/(v1|v2)"
```

### Preifx Match

In this example, any request starting with `/api/v3/time` will match `testRoute`.

```yaml
routes:
  testRoute:
    paths:
      - "/api/v3"
    service_id: service1
```

If a request matches two prefix rules, the longest matching rule takes priority. In the example below, a `GET /api/v1/orders/12345` request will match `testRoute2`.

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
