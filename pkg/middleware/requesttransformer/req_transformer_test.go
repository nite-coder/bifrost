package requesttransformer

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestRemove(t *testing.T) {

	options := Options{
		Remove: RemoveOptions{
			Headers:     []string{"x-user-id"},
			Querystring: []string{"mode"},
		},
	}

	m := NewMiddleware(options)

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo?mode=1")
	hzCtx.Request.Header.Set("x-user-id", "1")
	m.ServeHTTP(ctx, hzCtx)

	userID := hzCtx.Request.Header.Get("x-user-id")
	assert.Empty(t, userID)

	mode := hzCtx.Request.URI().QueryArgs().Has("mode")
	assert.False(t, mode)
}

func TestAdd(t *testing.T) {

	options := Options{
		Add: AddOptions{
			Headers: map[string]string{
				"source": "web",
			},
			Querystring: map[string]string{
				"mode": "1",
			},
		},
	}

	m := NewMiddleware(options)

	ctx := context.Background()
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo")
	m.ServeHTTP(ctx, hzCtx)

	mode := hzCtx.Query("mode")
	assert.Equal(t, "1", mode)
}
