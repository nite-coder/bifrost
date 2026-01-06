# Command Line Interface

Bifrost runs in foreground mode using the Master-Worker architecture for high availability and zero-downtime upgrades.

| Flag              | Description                                      |
| :---------------- | :----------------------------------------------- |
| `-t`, `--test`    | Test if the configuration file format is correct |
| `-c`, `--conf`    | Specify the configuration file path              |
| `-v`, `--version` | Show version information                         |

## Starting Bifrost

### Foreground Mode (All Environments)

```bash
./bifrost -c conf/config.yaml
```

The Master process runs in the foreground and spawns a Worker process. All logs are output to stdout/stderr.

### With Systemd

```bash
systemctl start bifrost
```

Systemd manages the process lifecycle. Logs are captured by Journald:

```bash
journalctl -u bifrost -f
```

### With Docker

```bash
docker run -d bifrost:latest -c /etc/bifrost/config.yaml
```

Logs are captured by Docker:

```bash
docker logs -f <container_id>
```

## Stopping Bifrost

### Via Signals

*   **SIGINT (Ctrl+C)** or **SIGTERM**: Graceful Shutdown. The Master process instructs the Worker to finish active requests before exiting.

### With Systemd

```bash
systemctl stop bifrost
```

### With Docker/Kubernetes

The container orchestrator sends `SIGTERM` to the process.

## Zero Downtime Upgrade (Hot Reload)

Bifrost supports hot reloading to apply new configurations without dropping connections.

### Via Signal

Send `SIGHUP` to the Master process:

```bash
kill -HUP $(pgrep -f 'bifrost.*-c')
```

### With Systemd

```bash
systemctl reload bifrost
```

### How it works

1.  Master receives `SIGHUP`.
2.  Master spawns a new Worker process with the new configuration.
3.  New Worker inherits active listeners from the old Worker (Zero Downtime).
4.  Once the new Worker is ready, Master gracefully shuts down the old Worker.
