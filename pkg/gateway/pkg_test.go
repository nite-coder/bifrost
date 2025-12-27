package gateway

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestGatewayRun(t *testing.T) {

	options := config.NewOptions()

	watch := true
	options.Watch = &watch

	// setup server
	options.Servers["apiv1"] = config.ServerOptions{
		Bind:        "localhost:8080",
		ReusePort:   true,
		TCPQuickAck: true,
		TCPFastOpen: true,
		Backlog:     4096,
		PPROF:       true,
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Run panic: %v", r)
			}
		}()

		err := Run(options)
		assert.NoError(t, err)
	}()

	assert.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", "localhost:8080", 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		return false
	}, 10*time.Second, 100*time.Millisecond, "Server failed to start")
	shutdown(context.Background(), true)
}
