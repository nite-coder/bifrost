# Command Line Interface

Bifrost supports command-line functions for starting, testing, and managing the gateway lifecycle. 
The gateway automatically uses the Master-Worker architecture for high availability and zero-downtime upgrades.

| Flag                  | Description                                            |
| :-------------------- | :----------------------------------------------------- |
| `-d`, `--daemon`      | Run Master in background (Daemon mode)                 |
| `-u`, `--upgrade`     | Trigger a hot reload (Zero Downtime Upgrade)           |
| `-s`, `--stop`        | Gracefully stop the running instance                   |
| `-t`, `--test`        | Test if the configuration file format is correct       |
| `-c`, `--conf`        | Specify the configuration file path                    |
| `-v`, `--version`     | Show version information                               |

## Starting Bifrost

### Foreground (Development)

To run Bifrost in the foreground (useful for development):

```bash
./bifrost -c conf/config.yaml
```

The Master process will start in the foreground and spawn a Worker process.

### Daemon Mode (Production)

To run Bifrost in the background as a daemon:

```bash
./bifrost -d -c conf/config.yaml
```

The Master process will fork to the background, write a PID file, and manage the Worker process.

## Stopping Bifrost

### Via CLI

You can use the `-s` flag to stop the running daemon:

```bash
./bifrost -s
```

### Via Signals (Manual)

*   **SIGINT (Ctrl+C)** or **SIGTERM**: Graceful Shutdown. The Master process will instruct the Worker to finish active requests (within a timeout) before exiting.

## Zero Downtime Upgrade (Hot Reload)

Bifrost supports hot reloading to apply new configurations or upgrade the binary without dropping connections.

### Via CLI

```bash
./bifrost -u
```

### Via Signal

Send `SIGHUP` to the Master process PID:

```bash
kill -HUP <MASTER_PID>
```

### How it works

1.  Master receives the upgrade signal.
2.  Master spawns a new Worker process with the new configuration/binary.
3.  New Worker inherits active listeners from the old Worker (Zero Downtime).
4.  Once the new Worker is ready, Master gracefully shuts down the old Worker.
