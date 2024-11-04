package trafficsplitter

import (
	"crypto/rand"
	"math/big"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/mitchellh/mapstructure"
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
	_ = middleware.RegisterMiddleware("traffic_splitter", func(params map[string]any) (app.HandlerFunc, error) {

		opts := &Options{}

		config := &mapstructure.DecoderConfig{
			Metadata: nil,
			Result:   opts,
			TagName:  "mapstructure",
		}

		decoder, err := mapstructure.NewDecoder(config)
		if err != nil {
			return nil, err
		}

		if err := decoder.Decode(params); err != nil {
			return nil, err
		}

		m := NewMiddleware(opts)

		return m.ServeHTTP, nil
	})
}
