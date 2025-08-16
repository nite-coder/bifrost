package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/cloudwego/hertz/pkg/protocol/http1/resp"
	"github.com/hertz-contrib/http2/factory"
	hertzslog "github.com/hertz-contrib/logger/slog"
	"github.com/hertz-contrib/websocket"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/nite-coder/bifrost/pkg/middleware/cors"
)

var (
	delay = flag.Duration("delay", 0, "delay to mock business processing")
	tail  = flag.Duration("tail", 0, "1% long tail latency")
	nacos = flag.Bool("nacos", false, "enable nacos")
)

func WithDefaultServerHeader(disable bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.NoDefaultServerHeader = disable
	}}
}

const (
	bind        = ":8000"
	actionQuery = "action"
	order       = `{
		"id": "123456",
		"client_order_id": "123",
		"market": "BTC_USDT",
		"side": "buy",
		"amount": "0.001",
		"price": "65000",
		"tif": "fok",
	  }`

	orderbook = `{
		"current": 1711255163345,
		"update": 1711255163342,
		"asks": [
			[
				"63837.9",
				"0.25997"
			],
			[
				"63839.4",
				"0.14"
			],
			[
				"63839.5",
				"0.18"
			],
			[
				"63844.6",
				"0.00812"
			],
			[
				"63845.7",
				"0.01256"
			],
			[
				"63846.3",
				"0.02726"
			],
			[
				"63847.8",
				"0.12405"
			],
			[
				"63847.9",
				"0.20674"
			],
			[
				"63850.3",
				"0.14139"
			],
			[
				"63850.4",
				"0.23563"
			]
		],
		"bids": [
			[
				"63837.8",
				"0.96189"
			],
			[
				"63837.6",
				"0.00583"
			],
			[
				"63835.5",
				"0.47803"
			],
			[
				"63833.7",
				"0.06774"
			],
			[
				"63833",
				"0.25504"
			],
			[
				"63832.7",
				"0.02192"
			],
			[
				"63829.3",
				"0.05346"
			],
			[
				"63827",
				"0.10805"
			],
			[
				"63826.9",
				"0.05348"
			],
			[
				"63826.6",
				"0.18"
			]
		]
	}`
)

var (
	orderResp     []byte
	orderbookResp []byte
)

func main() {
	flag.Parse()

	orderResp = []byte(order)
	orderbookResp = []byte(orderbook)

	opts := []config.Option{
		server.WithHostPorts(bind),
		server.WithIdleTimeout(time.Second * 60),
		server.WithReadTimeout(time.Second * 30),
		server.WithWriteTimeout(time.Second * 30),
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		server.WithH2C(true),
		server.WithStreamBody(true),
		WithDefaultServerHeader(true),
	}
	h := server.New(opts...)
	h.Use(cors.Default())

	logger := hertzslog.NewLogger(hertzslog.WithOutput(io.Discard))
	hlog.SetLevel(hlog.LevelError)
	hlog.SetLogger(logger)
	hlog.SetSilentMode(true)

	h.AddProtocol("h2", factory.NewServerFactory())

	h.POST("/", echoHandler)
	h.Any("/spot/order", placeOrderHandler)
	h.Any("/spot/orders", placeOrderHandler)
	h.Any("/api/v1/spot/orders", placeOrderHandler)
	h.POST("/futures/usdt/orders", placeOrderHandler)
	h.POST("/options/orders", placeOrderHandler)
	h.GET("/order_book", orderBookHandler)
	h.DELETE("cancel_order", cancelOrderHandler)
	h.POST("/long", longHandler)
	h.GET("/dynamic_upstream", findUpstreamHandler)
	h.GET("/websocket", wssHandler)
	h.GET("/chunk", chunkHandler)

	h.GET("/users/:user_id/orders", func(c context.Context, ctx *app.RequestContext) {
		userID := ctx.Param("user_id")
		ctx.String(200, "orders:"+userID)
	})

	h.GET("/users/:name/orders1", func(c context.Context, ctx *app.RequestContext) {
		name := ctx.Param("name")
		ctx.String(200, "order1:"+name)
	})

	if *nacos {
		go registerNacosServiceProvider()
	}

	h.Spin()
}

func echoHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("text/plain; charset=utf8")
	ctx.Response.SetStatusCode(200)
	ctx.Response.SetBody(ctx.Request.Body())
}

var placeOrderCounter atomic.Uint64

func placeOrderHandler(c context.Context, ctx *app.RequestContext) {

	if (placeOrderCounter.Load() % 99) == 0 {
		if *tail > 0 {
			time.Sleep(*tail)
		} else {
			runtime.Gosched()
		}
	} else {
		if *delay > 0 {
			time.Sleep(*delay)
		} else {
			runtime.Gosched()
		}
	}

	placeOrderCounter.Add(1)

	ctx.SetContentType("application/json; charset=utf8")
	ctx.Response.SetStatusCode(200)
	ctx.Response.SetBody(orderResp)
}

func orderBookHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("application/json; charset=utf8")
	ctx.Response.SetStatusCode(200)
	ctx.Response.SetBody(orderbookResp)
}

func cancelOrderHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("application/json; charset=utf8")
	ctx.Response.SetStatusCode(200)
	ctx.Response.SetBody(orderResp)
}

func longHandler(c context.Context, ctx *app.RequestContext) {
	time.Sleep(10 * time.Second)
	ctx.SetContentType("text/plain; charset=utf8")
	ctx.String(200, "hello")
}

func findUpstreamHandler(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("text/plain; charset=utf8")
	ctx.String(200, "find upstream")
}

var upgrader = websocket.HertzUpgrader{
	CheckOrigin: func(r *app.RequestContext) bool {
		return true
	},
} // use default options

func wssHandler(c context.Context, ctx *app.RequestContext) {
	err := upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				// slog.ErrorContext(c, "read err:", "error", err)
				break
			}
			//slog.Info("recv", "msg", string(msg))

			err = conn.WriteMessage(websocket.TextMessage, orderResp)
			if err != nil {
				// slog.ErrorContext(c, "write err:", "error", err)
				break
			}
		}
	})

	if err != nil {
		slog.ErrorContext(c, "upgrade err:", "error", err)
		return
	}
}

func chunkHandler(ctx context.Context, c *app.RequestContext) {

	// Hijack the writer of response
	c.Response.HijackWriter(resp.NewChunkedBodyWriter(&c.Response, c.GetWriter()))

	for i := 0; i < 100; i++ {
		c.Write(orderResp) // nolint: errcheck
		c.Flush()          // nolint: errcheck
		time.Sleep(1 * time.Second)
	}
}

func registerNacosServiceProvider() {
	clientConfig := constant.ClientConfig{
		NamespaceId:         "public",           // Default namespace if not specified
		TimeoutMs:           5000,               // Request timeout in milliseconds
		NotLoadCacheAtStart: true,               // Do not load cache at startup
		LogDir:              "/tmp/nacos/log",   // Log directory
		CacheDir:            "/tmp/nacos/cache", // Cache directory
		LogLevel:            "debug",            // Log level
	}

	// Configure Nacos server address
	serverConfigs := []constant.ServerConfig{
		{
			IpAddr:      "127.0.0.1", // Nacos server IP address
			Port:        8848,        // Nacos server port
			ContextPath: "/nacos",    // Nacos context path
		},
	}

	// Initialize NamingClient
	namingClient, err := clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: serverConfigs,
		},
	)
	if err != nil {
		log.Fatalf("Failed to create Nacos naming client: %v", err)
	}

	// Service registration parameters
	serviceName := "order_service" // Service name
	ip := "localhost"              // Service IP address

	// Register service
	result, err := namingClient.RegisterInstance(vo.RegisterInstanceParam{
		ServiceName: serviceName,
		GroupName:   "DEFAULT_GROUP", // Default group name
		Ip:          ip,
		Port:        uint64(8000),
		Weight:      10,                                  // Weight, default is 1
		Enable:      true,                                // Enable status
		Healthy:     true,                                // Health status
		Ephemeral:   true,                                // Whether it's an ephemeral instance
		Metadata:    map[string]string{"version": "1.0"}, // Custom metadata
	})
	if err != nil {
		log.Fatalf("Failed to register service instance: %v", err)
	}

	fmt.Println("Service registered:", result)

	select {}
}
