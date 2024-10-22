# Start and Stop

Bifrost supports command-line functions for starting, testing, and more. For a complete API Gateway setup, refer to [Gateway](./../../examples/gateway/main.go)

## Start

| Command Parameter |   Default    | Description                                      |
| :---------------- | :----------: | :----------------------------------------------- |
| -d, --daemon      |    false     | Run in background mode                           |
| -t, --test        |    false     | Test if the configuration file format is correct |
| -c, --conf        | empty string | Specify the configuration file path              |
| -u, --upgrade     |    false     | Perform a hot reload upgrade                     |

## Stop

This section will introduce how to upgrade the API Gateway and stop the daemon process.

### SIGINT: Fast shutdown

When the SIGINT (ctrl + c) signal is received, the server will exit immediately, without delay, and all ongoing requests will be interrupted. This is generally not recommended, as it can disrupt in-progress requests.

### SIGTERM: Graceful shutdown

Upon receiving the SIGTERM signal, the server will notify all services to shut down, wait for the preset timeout period, and then exit. This behavior grants a grace period for requests to complete before shutting down gracefully.

### SIGQUIT: Graceful upgrade

When the server receives a signal similar to SIGTERM, it transfers all listening sockets to a new Bifrost instance, ensuring no downtime during the upgrade process. For more details, refer to the graceful upgrade section.
