package gateway

import (
	"context"
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

	time.Sleep(3 * time.Second)
	shutdown(context.Background(), true)
}
