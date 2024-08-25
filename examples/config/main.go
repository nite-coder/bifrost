package main

import (
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/gateway"
)

func main() {
	options, err := config.Load("./config.yaml")
	if err != nil {
		panic(err)
	}

	err = gateway.Run(options)
	if err != nil {
		panic(err)
	}
}
