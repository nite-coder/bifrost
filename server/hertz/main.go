package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nite-coder/bifrost/internal/pkg/runtime"
	grpcproxy "github.com/nite-coder/bifrost/pkg/proxy/grpc"
	httpproxy "github.com/nite-coder/bifrost/pkg/proxy/http"

	configBifrost "github.com/nite-coder/bifrost/pkg/config"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	hertzslog "github.com/hertz-contrib/logger/slog"
)

func withDefaultServerHeader(disable bool) config.Option {
	return config.Option{F: func(o *config.Options) {
		o.NoDefaultServerHeader = disable
	}}
}

var (
	daemon = flag.Bool("d", false, "Run in daemon mode")
)

func main() {
	flag.Parse()

	if runtime.IsWorker() {
		if err := runWorker(); err != nil {
			log.Fatal(err)
		}
		return
	}

	// Master Mode
	master := runtime.NewMaster(&runtime.MasterOptions{
		Binary: os.Args[0],
		Args:   os.Args[1:],
		KeepAlive: &runtime.KeepAliveOptions{
			InitialBackoff: 1 * time.Second,
		},
	})
	if err := master.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func runWorker() error {
	ctx := context.Background()

	// 1. Setup Worker Control Plane
	socketPath := runtime.GetControlSocketPath()
	wcp := runtime.NewWorkerControlPlane(socketPath)
	if err := wcp.Connect(); err != nil {
		return fmt.Errorf("failed to connect to control plane: %w", err)
	}
	defer wcp.Close()

	if err := wcp.Register(); err != nil {
		return fmt.Errorf("failed to register with master: %w", err)
	}

	// 2. Start Control Plane Loop
	go func() {
		if err := wcp.Start(context.Background(), nil); err != nil {
			slog.Error("control plane loop exited", "error", err)
		}
	}()

	// 3. Create Listener
	zeroDT := runtime.New(runtime.Options{})
	listenOptions := &runtime.ListenerOptions{
		Network: "tcp",
		Address: ":8001",
	}

	listener, err := zeroDT.Listener(ctx, listenOptions)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
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

	opts := httpproxy.Options{
		Target:   "http://localhost:8000",
		Weight:   1,
		Protocol: configBifrost.ProtocolHTTP,
	}
	httpProxy, err := httpproxy.New(opts, nil)
	if err != nil {
		return err
	}

	grpcOption := grpcproxy.Options{
		Target:    "grpc://127.0.0.1:8501",
		TLSVerify: false,
	}

	grpcProxy, err := grpcproxy.New(grpcOption)
	if err != nil {
		return err
	}
	h.POST("/spot/orders", httpProxy.ServeHTTP)
	h.POST("/helloworld.Greeter/SayHello", grpcProxy.ServeHTTP)
	h.GET("/chunk", httpProxy.ServeHTTP)

	go func() {
		h.Spin()
	}()

	// 4. Notify Ready
	go func() {
		for {
			conn, err := net.Dial("tcp", ":8001")
			if err == nil {
				conn.Close()
				slog.Info("server ready", "bind", ":8001")
				_ = wcp.NotifyReady()
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// 5. Wait for Shutdown Signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("received shutdown signal", "pid", os.Getpid())
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := h.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	_ = zeroDT.Close(ctx)
	return nil
}
