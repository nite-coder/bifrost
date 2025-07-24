package main

import (
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/gateway"
	"github.com/nite-coder/bifrost/pkg/initialize"
)

func main() {
	_ = initialize.Bifrost()

	options, err := config.Load("./config.yaml")
	if err != nil {
		panic(err)
	}

	err = gateway.Run(options)
	if err != nil {
		panic(err)
	}
}
