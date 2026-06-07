package metrics

import prom "github.com/prometheus/client_golang/prometheus"

var (
	// HTTPServerOpenConnections represents the number of open connections for servers.
	HTTPServerOpenConnections *prom.GaugeVec
	// HTTPServiceOpenConnections represents the number of open connections for services.
	HTTPServiceOpenConnections *prom.GaugeVec
)

func init() {
	HTTPServerOpenConnections = prom.NewGaugeVec(
		prom.GaugeOpts{
			Name: "http_server_open_connections",
			Help: "Number of open connections for servers",
		},
		[]string{"server_id"},
	)
	prom.MustRegister(HTTPServerOpenConnections)

	HTTPServiceOpenConnections = prom.NewGaugeVec(
		prom.GaugeOpts{
			Name: "http_service_open_connections",
			Help: "Number of open connections for services",
		},
		[]string{"service_id", "target"},
	)
	prom.MustRegister(HTTPServiceOpenConnections)
}
