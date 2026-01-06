package main

import (
	"os"

	"github.com/nite-coder/bifrost"
)

func main() {
	err := bifrost.Run()
	if err != nil {
		os.Exit(1)
	}
}
