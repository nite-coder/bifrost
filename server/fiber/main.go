package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/valyala/fasthttp"
)

func main() {
	// Initialize a new Fiber app
	app := fiber.New(fiber.Config{
		CaseSensitive: true,
		StrictRouting: true,
		//Prefork:       true,
		Immutable: false,
	})

	// if you need to use global self-custom client, you should use proxy.WithClient.
	proxy.WithClient(&fasthttp.Client{
		NoDefaultUserAgentHeader: true,
		DisablePathNormalizing:   true,
		MaxConnsPerHost:          4000,
	})

	// Make proxy requests while following redirects
	app.Post("/spot/orders", func(c *fiber.Ctx) error {
		return proxy.Do(c, "http://localhost:8000/spot/orders")
	})

	log.Fatal(app.Listen(":8001"))
}
