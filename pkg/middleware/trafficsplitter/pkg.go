package trafficsplitter

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

func getRandomNumber(max int64) (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}

func init() {
	_ = middleware.Register([]string{"traffic_splitter"}, func(params any) (app.HandlerFunc, error) {
		if params == nil {
			return nil, errors.New("traffic_splitter middleware params is empty or invalid")
		}

		opts := &Options{}

		err := mapstructure.Decode(params, &opts)
		if err != nil {
			return nil, fmt.Errorf("traffic_splitter middleware params is invalid: %w", err)
		}

		m := NewMiddleware(opts)
		return m.ServeHTTP, nil
	})
}
