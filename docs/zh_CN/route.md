# 路由

Bifrost 的路由設計跟 nginx 類似，主要分四種

1. 精準匹配 `=` ( exact match )
1. 前缀匹配 `^=` ( prefix match )
1. 正则匹配 `~`( regexp Match )
1. 通用匹配 （genernal Match）

## 匹配優先順序

精準匹配 > 前缀匹配 > 正则匹配 > 通配符匹配

## 精準匹配

Path 的首字符是用 `=`，下面這個範例如果請求跟 `/spot/orders` 完全一樣才會被匹配到

```yaml
routes:
  testRoute:
    paths:
      - "= /spot/orders"
```

## 前缀匹配

Path 的起頭是用 `^=`，下面這個這個範例配置如果請求是 `/api/v1` 開頭的都會被匹配到

```yaml
routes:
  testRoute:
    paths:
      - "^= /api/v1"
    service_id: service1
```

如果有一個請求，同時符合兩個前缀匹配規則，則會以規則路由最長的優先級最高，下面範例如果有一個請求是 `GET /api/v1/orders/12345`，兩個規則都符合時取最長的也就是 `testRoute2` 會被選中

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

## 正則匹配

Path 的首字符是用 `~` 代表正則匹配，下面這個範例如果請求的路徑是 `/api/v2/my_orders` 的則會符合下面的正則規則，多個正則規則執行的優先順序是依照配置文件的由上往下的順序

```yaml
routes:
  testRoute:
    paths:
      - "~ /api/(v1|v2)"
```

## 通用匹配

下面這個這個範例如果請求是 `/api/v3/time` 開頭的都會匹配到 `testRoute`

```yaml
routes:
  testRoute:
    paths:
      - "/api/v3"
    service_id: service1
```

如果有一個請求，同時符合兩個前缀匹配規則，則會以規則路由最長的優先級最高，下面範例如果有一個請求是 `GET /api/v1/orders/12345`，兩個規則都符合時取最長的也就是 `testRoute2` 會被選中

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
