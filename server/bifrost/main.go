package main

import (
	"http-benchmark/pkg/gateway"
	"http-benchmark/pkg/middleware/timinglogger"
	"log/slog"

	"github.com/cloudwego/hertz/pkg/app"
)

func main() {

	_ = gateway.RegisterMiddleware("timing_logger", func(param map[string]any) (app.HandlerFunc, error) {
		m := timinglogger.NewMiddleware()
		return m.ServeHTTP, nil
	})

	bifrost, err := gateway.LoadFromConfig("./config.yaml")
	if err != nil {
		slog.Error("load config error", "error", err)
	}

	bifrost.Run()
}
