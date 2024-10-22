package gateway

import (
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
)

func BenchmarkSaveByte(b *testing.B) {

	path := []byte(`/spot/orders/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`)

	c := app.NewContext(0)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b := []byte{}
		copy(b, path)
		c.Set(config.REQUEST_PATH, b)
	}
}

func BenchmarkSaveString(b *testing.B) {

	path := []byte(`/spot/orders/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx`)

	c := app.NewContext(0)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := string(path)
		c.Set(config.REQUEST_PATH, p)
	}
}
