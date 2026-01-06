package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nite-coder/bifrost/internal/pkg/runtime"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type ProxyHandler struct {
	proxy *httputil.ReverseProxy
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.proxy.ServeHTTP(w, r)
}

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
		if err := wcp.Start(context.Background(), nil); err != nil { // nil handler as we don't need to handle incoming FDs in this simple example
			slog.Error("control plane loop exited", "error", err)
		}
	}()

	// 3. Create Listener
	zeroDT := runtime.New(runtime.Options{})
	listenOptions := &runtime.ListenerOptions{
		Network: "tcp",
		Address: ":8001",
	}
	ln, err := zeroDT.Listener(ctx, listenOptions)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	upstreamURL, err := url.Parse("http://localhost:8000")
	if err != nil {
		return fmt.Errorf("failed to parse upstream URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(upstreamURL)
	proxy.Transport = &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		},
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
		},
	}

	handler := &ProxyHandler{
		proxy: proxy,
	}

	h2s := &http2.Server{}
	httpServer := &http.Server{
		ReadTimeout: 10 * time.Second,
		Handler:     h2c.NewHandler(handler, h2s),
	}

	go func() {
		slog.Info("starting server", "bind", ":8001")
		if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Error starting server: %v\n", err)
		}
	}()

	// 4. Notify Ready
	// Simple check: try to dial the port
	go func() {
		for {
			conn, err := net.Dial("tcp", ":8001")
			if err == nil {
				conn.Close()
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

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	// Close any other resources if needed
	_ = zeroDT.Close(ctx) // Closes listeners
	return nil
}
