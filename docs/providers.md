# Providers

Providers enable integration with various services, managing configuration files and service discovery

## Built-In Providers

The currently supported providers are listed below. You can combine multiple providers for use.

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
      - ".json"
    paths:
      - "./conf"
```

| Field      | Type       | Default                | Description                                               |
| ---------- | ---------- | ---------------------- | --------------------------------------------------------- |
| enabled    | `bool`     | `false`                | Enables the file provider                                 |
| extensions | `[]string` | `.yaml`,`.yml`, `json` | Allowed file extensions                                   |
| paths      | `[]string` |                        | Directories or files to be loaded.  (Skip subdirectories) |

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

| Field                     | Type       | Default              | Description                                                                                                               |
| ------------------------- | ---------- | -------------------- | ------------------------------------------------------------------------------------------------------------------------- |
| nacos.config.enabled      | `bool`     | `false`              | Enables the nacos provider                                                                                                |
| nacos.config.namespace_id | `string`   |                      | the namespaceId of Nacos. when namespace is public, fill in the blank string here                                         |
| nacos.config.context_path | `string`   | `/nacos`             | the nacos server contextpath                                                                                              |
| nacos.config.cache_dir    | `string`   | `./logs/nacos/cache` | the directory for persist nacos service info                                                                              |
| nacos.config.log_dir      | `string`   | `./logs`             | the directory for log                                                                                                     |
| nacos.config.log_level    | `string`   | `info`               | the level of log, it's must be `debug`,`info`,`warn`, `error`                                                             |
| nacos.config.namespace_id | `string`   |                      | the namespaceId of Nacos. when namespace is `public`, fill in the blank string here                                       |
| nacos.config.watch        | `bool`     |                      | Determines if configuration hot reload is enabled. When enabled, `dynamic configuration` changes take effect immediately. |
| nacos.config.timeout      | `string`   | `10s`                | timeout for requesting Nacos server                                                                                       |
| nacos.config.endpoints    | `[]string` |                      | list of nacos server endpoints (http only)                                                                                |
| nacos.config.files        | `[]file`   |                      | list of configuration files in nacos                                                                                      |
