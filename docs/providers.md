# Providers

Providers enable integration with various services, managing configuration files and service discovery

## Built-In Providers

The currently supported providers are listed below. You can also combine multiple providers for use.

* [File](#file): Get configurations from files
* [Nacos](#nacos): Get configurations from nacos.

## File

Get gateway configurations from files

Example:

```yaml
providers:
  file:
    enabled: true
    extensions:
      - ".yaml"
      - ".yml"
    paths:
      - "./conf"
```

| Field      | Type       | Default        | Description                                                            |
| ---------- | ---------- | -------------- | ---------------------------------------------------------------------- |
| enabled    | `bool`     | `false`        | Enables the file provider                                              |
| extensions | `[]string` | `.yaml`,`.yml` | Allowed file extensions (`yaml` format is tested)                      |
| paths      | `[]string` |                | Directories or files to be loaded.  (Skip subdirectories)              |
| watch      | `bool`     | `true`         | if `true`, changes to `dynamic configuration` take effect immediately. |

## Nacos

Get gateway configurations from from nacos configuration center. (http protocol only)

Example:

```yaml
providers:
  nacos:
    config:
      enabled: true
      namespace_id: df137446-c324-4e03-85e4-4ce32432ca53
      timeout: 3s
      endpoints:
        - "http://127.0.0.1:8848"
      files:
        - data_id: "routes.yaml"
          group: "DEFAULT_GROUP"
```

| Field               | Type       | Default              | Description                                                                         |
| ------------------- | ---------- | -------------------- | ----------------------------------------------------------------------------------- |
| config.enabled      | `bool`     | `false`              | Enables the nacos provider                                                          |
| config.username     | `string`   |                      | Username of nacos server                                                            |
| config.password     | `string`   |                      | Password of nacos server                                                            |
| config.namespace_id | `string`   |                      | the namespaceId of Nacos. when namespace is public, fill in the blank string here   |
| config.context_path | `string`   | `/nacos`             | the nacos server contextpath                                                        |
| config.cache_dir    | `string`   | `./logs/nacos/cache` | the directory for persist nacos service info                                        |
| config.log_dir      | `string`   | `./logs`             | the directory for log                                                               |
| config.log_level    | `string`   | `info`               | the level of log, it's must be `debug`,`info`,`warn`, `error`                       |
| config.namespace_id | `string`   |                      | the namespaceId of Nacos. when namespace is `public`, fill in the blank string here |
| config.watch        | `bool`     | `true`               | if `true`, changes to `dynamic configuration` take effect immediately.              |
| config.timeout      | `string`   | `10s`                | timeout for requesting Nacos server                                                 |
| config.endpoints    | `[]string` |                      | list of nacos server endpoints (http only)                                          |
| config.files        | `[]file`   |                      | list of configuration files in nacos                                                |
