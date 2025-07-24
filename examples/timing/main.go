package main

import (
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/gateway"
	"github.com/nite-coder/bifrost/pkg/initialize"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

func registerMiddlewares() error {
	err := middleware.RegisterMiddleware("timing", func(param any) (app.HandlerFunc, error) {
		m := TimingMiddleware{}
		return m.ServeHTTP, nil
	})

	if err != nil {
		return err
	}

	return nil
}

func main() {
	_ = initialize.Bifrost()

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
