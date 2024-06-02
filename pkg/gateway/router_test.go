package gateway

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestRouters(t *testing.T) {
	r := NewRouter()

	err := r.add(POST, "/spot/orders", nil)
	assert.NoError(t, err)

	err = r.add(POST, "/futures/acc*", nil)
	assert.NoError(t, err)

	m := r.find(POST, "/spot/orders")
	assert.Len(t, m, 1)

	m = r.find(POST, "/futures/account")
	assert.Len(t, m, 1)
}

// dummyHandler is a placeholder handler function
func dummyHandler(c context.Context, ctx *app.RequestContext) {
	ctx.String(200, "OK")
}

// BenchmarkFind benchmarks the find function
func BenchmarkFind(b *testing.B) {
	router := NewRouter()
	router.add(POST, "/spot/orders", dummyHandler)
	router.add(POST, "/spot2/orders", dummyHandler)
	router.add(POST, "/spot3/orders", dummyHandler)
	router.add(POST, "/spot4/orders", dummyHandler)
	router.add(POST, "/spot/5orders", dummyHandler)

	tests := []struct {
		method string
		path   string
	}{
		{POST, "/spot/orders"},
	}

	req := app.NewContext(1)
	req.Request.SetMethod("POST")
	req.URI().SetPath("/spot/orders")

	for _, tt := range tests {
		b.Run(tt.method+" "+tt.path, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				router.ServeHTTP(context.Background(), req)
			}
		})
	}
}
