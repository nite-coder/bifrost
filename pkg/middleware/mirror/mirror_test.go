package mirror

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/balancer/roundrobin"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/gateway"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/stretchr/testify/assert"
)

func TestMirror(t *testing.T) {
	_ = Init()
	_ = roundrobin.Init()
	options := config.NewOptions()

	// setup service
	options.Services["mirror_svc1"] = config.ServiceOptions{
		URL: "http://127.0.0.1:8000",
	}

	bifrost, err := gateway.NewBifrost(options, false)
	assert.NoError(t, err)
	gateway.SetBifrost(bifrost)

	h := middleware.Factory("mirror")

	params := map[string]any{
		"service_id": "mirror_svc1",
	}

	m, err := h(params)
	assert.NoError(t, err)

	ctx := context.Background()
	hzCtx := app.NewContext(0)

	var hit atomic.Int32
	hzCtx.SetHandlers([]app.HandlerFunc{func(ctx context.Context, c *app.RequestContext) {
		hit.Add(1)
	}})

	m(ctx, hzCtx)

	assert.Eventually(t, func() bool {
		return hit.Load() == 1
	}, 2*time.Second, 100*time.Millisecond)
}
