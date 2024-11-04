package trafficsplitter

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestSplitter(t *testing.T) {

	options := &Options{
		Key: "$my_order",
		Destinations: []*Destination{
			{
				To:     "old_server",
				Weight: 90,
			},
			{
				To:     "new_server",
				Weight: 10,
			},
		},
	}

	m := NewMiddleware(options)

	hits := map[string]int{"old_server": 0, "new_server": 0}

	for i := 0; i < 1000; i++ {
		ctx := context.Background()
		hzCtx := app.NewContext(0)
		hzCtx.Request.SetMethod("POST")
		hzCtx.Request.URI().SetPath("/api/v1/hello")
		m.ServeHTTP(ctx, hzCtx)

		val := hzCtx.GetString(m.options.Key)
		hits[val]++
	}

	assert.InDelta(t, 900, hits["old_server"], 50)
	assert.InDelta(t, 100, hits["new_server"], 50)
	t.Log("old_server", hits["old_server"])
	t.Log("new_server", hits["new_server"])
}
