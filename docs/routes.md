# Routes

Bifrost's routing design is similar to NGINX, with four main types:

1. Exact match `=`
1. Prefix match `^~`
1. Regular expression match `~` (case sensitive)
1. Regular expression match `~*` (case insensitive)
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

Paths that start with `^~` are prefix matches. In this example, any request starting with `/api/v1` will be matched.

```yaml
routes:
  testRoute:
    paths:
      - "^~ /api/v1"
    service_id: service1
```

If a request meets two prefix match rules, the longest matching rule takes priority. In the example below, a request GET `/api/v1/orders/12345` will match `testRoute2`.

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

Paths that begin with `~` denote a regular expression match (case sensitive). In this example, requests with paths like `/api/v2/my_orders` will match. Multiple regex rules follow the order in the configuration file from top to bottom.

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
