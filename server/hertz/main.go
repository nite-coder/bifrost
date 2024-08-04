package main

import (
	"context"
	"flag"
	"fmt"
	"http-benchmark/pkg/gateway"
	"http-benchmark/pkg/zero"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	configBifrost "http-benchmark/pkg/config"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	configHTTP2 "github.com/hertz-contrib/http2/config"
	"github.com/hertz-contrib/http2/factory"
	hertzslog "github.com/hertz-contrib/logger/slog"
)

func withDefaultServerHeader(disable bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.NoDefaultServerHeader = disable
	}}
}

var (
	daemon  = flag.Bool("d", false, "Run in daemon mode")
	upgrade = flag.Bool("u", false, "Perform a hot upgrade")
	shudown = flag.Bool("stop", false, "shutdown server")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	zeroOpts := zero.Options{
		SocketPath: "./hertz.sock",
		PIDFile:    "./hertz.pid",
	}

	zeroDT := zero.New(zeroOpts)

	if *upgrade {
		if err := zeroDT.Upgrade(); err != nil {
			log.Fatalf("Upgrade failed: %v", err)
		}
		return
	}

	if *shudown {
		_ = zeroDT.Shutdown(ctx)
		return
	}

	done := make(chan bool, 1)
	if zeroDT.IsUpgraded() {
		go func() {
			<-done

			err := zeroDT.Shutdown(ctx)
			if err != nil {
				return
			}

			time.Sleep(5 * time.Second)

			if *daemon {
				if err := zeroDT.WaitForUpgrade(ctx); err != nil {
					slog.Error("Upgrade process error: %v", err)
					return
				}
			}
		}()

		err := startup(ctx, zeroDT, done)
		if err != nil {
			slog.Error("Failed to start server", "error", err)
			return
		}
	} else {
		go func() {
			<-done

			if *daemon {
				if err := zeroDT.WaitForUpgrade(ctx); err != nil {
					slog.Error("failed to upgrade process", "error", err)
					return
				}
			}
		}()

		err := startup(ctx, zeroDT, done)
		if err != nil {
			slog.Error("Failed to start server", "error", err)
			return
		}
	}
}

func startup(ctx context.Context, zeroDT *zero.ZeroDownTime, done chan bool) error {
	if *daemon && os.Getenv("DAEMONIZED") == "" {
		daemonize()
		return nil
	}

	listener, err := zeroDT.Listener(ctx, "tcp", ":8001", nil)
	if err != nil {
		slog.Error("failed to create listener", "error", err)
	}

	logger := hertzslog.NewLogger(hertzslog.WithOutput(io.Discard))
	hlog.SetLevel(hlog.LevelError)
	hlog.SetLogger(logger)
	hlog.SetSilentMode(true)

	hzOpts := []config.Option{
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		server.WithSenseClientDisconnection(true),
		server.WithReadTimeout(time.Second * 3),
		server.WithWriteTimeout(time.Second * 3),
		server.WithKeepAlive(true),
		server.WithALPN(true),
		server.WithStreamBody(true),
		server.WithH2C(true),
		server.WithExitWaitTime(1 * time.Second),
		server.WithListener(listener),
		withDefaultServerHeader(true),
	}

	h := server.New(hzOpts...)

	http2opts := []configHTTP2.Option{}
	h.AddProtocol("h2", factory.NewServerFactory(http2opts...))

	// defaultClientOptions := []config.ClientOption{
	// 	client.WithNoDefaultUserAgentHeader(true),
	// 	client.WithDisableHeaderNamesNormalizing(true),
	// 	client.WithDisablePathNormalizing(true),
	// 	client.WithMaxConnsPerHost(math.MaxInt),
	// 	client.WithDialTimeout(3 * time.Second),
	// 	client.WithClientReadTimeout(3 * time.Second),
	// 	client.WithWriteTimeout(3 * time.Second),
	// 	client.WithKeepAlive(true),
	// }

	//proxy, _ := reverseproxy.NewSingleHostReverseProxy("http://localhost:8000", defaultClientOptions...)

	opts := gateway.ProxyOptions{
		Target:   "http://localhost:8000",
		Weight:   1,
		Protocol: configBifrost.ProtocolHTTP,
	}
	proxy, err := gateway.NewReverseProxy(opts, nil)
	if err != nil {
		return err
	}
	h.POST("/spot/orders", proxy.ServeHTTP)

	go func() {
		h.Spin()
	}()

	go func() {
		for {
			conn, err := net.Dial("tcp", ":8001")
			if err == nil {
				conn.Close()
				slog.Info("starting server", "bind", ":8001", "isDaemon", *daemon)
				done <- true
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-sigChan

	slog.Info("received shutdown signal", "pid", os.Getpid())

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := h.Shutdown(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to shutdown server", "error", err)
		return err
	}

	_ = zeroDT.Close(ctx)

	slog.Info("server is shutdown successfully", "pid", os.Getpid())
	return nil

}

func daemonize() {
	cmd := exec.Command(os.Args[0], os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "DAEMONIZED=1")

	err := cmd.Start()
	if err != nil {
		log.Fatalf("Failed to start daemon: %v", err)
	}

	fmt.Printf("Daemon process started with PID %d\n", cmd.Process.Pid)

	os.Exit(0)
}
