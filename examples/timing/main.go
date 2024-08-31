package main

import (
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/gateway"

	"github.com/cloudwego/hertz/pkg/app"
)

func main() {
	options, err := config.Load("./config.yaml")
	if err != nil {
		panic(err)
	}

	err = registerMiddlewares()
	if err != nil {
		panic(err)
	}

	err = gateway.Run(options)
	if err != nil {
		panic(err)
	}
}

func registerMiddlewares() error {
	err := gateway.RegisterMiddleware("timing", func(param map[string]any) (app.HandlerFunc, error) {
		m := TimingMiddleware{}
		return m.ServeHTTP, nil
	})
	if err != nil {
		return err
	}

	return nil
}
