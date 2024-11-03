package main

import (
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/gateway"
	"github.com/nite-coder/bifrost/pkg/middleware"
	"github.com/cloudwego/hertz/pkg/app"
)

func registerMiddlewares() error {
	err := middleware.RegisterMiddleware("timing", func(param map[string]any) (app.HandlerFunc, error) {
		m := TimingMiddleware{}
		return m.ServeHTTP, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func main() {
	err := registerMiddlewares()
	if err != nil {
		panic(err)
	}

	options, err := config.Load("./config.yaml")
	if err != nil {
		panic(err)
	}

	err = gateway.Run(options)
	if err != nil {
		panic(err)
	}
}
