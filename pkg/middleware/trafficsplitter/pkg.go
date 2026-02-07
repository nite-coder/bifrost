package trafficsplitter

import (
	"crypto/rand"
	"math/big"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

var getRandomNumber = func(max int64) (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}

func Init() error {
	return middleware.RegisterTyped([]string{"traffic_splitter"}, func(opts Options) (app.HandlerFunc, error) {
		m := NewMiddleware(&opts)
		return m.ServeHTTP, nil
	})
}
