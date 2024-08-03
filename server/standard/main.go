package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

type ProxyHandler struct {
	proxy *httputil.ReverseProxy
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//log.Printf("Received request: %s %s %s", r.Method, r.URL, r.Proto)

	// if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
	// 	r.Header.Set("X-Forwarded-Proto", "h2c")
	// }

	h.proxy.ServeHTTP(w, r)
}

var (
	daemon  = flag.Bool("d", false, "Run in daemon mode")
	upgrade = flag.Bool("u", false, "Perform a hot upgrade")
	shudown = flag.Bool("stop", false, "shutdown server")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	zeroDT := New(ZeroDownTimeOptions{
		SocketPath: "./std.sock",
		PIDFile:    "./std.pid",
	})

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

			time.Sleep(2 * time.Second)

			_ = zeroDT.WritePID(ctx)

			if err := zeroDT.WaitForUpgrade(ctx); err != nil {
				log.Fatalf("Upgrade process error: %v", err)
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

			if err := zeroDT.WaitForUpgrade(ctx); err != nil {
				log.Fatalf("Upgrade process error: %v", err)
			}
		}()

		err := startup(ctx, zeroDT, done)
		if err != nil {
			slog.Error("Failed to start server", "error", err)
			return
		}
	}
}

func startup(ctx context.Context, zeroDT *ZeroDownTime, done chan bool) error {
	if *daemon && os.Getenv("DAEMONIZED") == "" {
		daemonize(zeroDT)
		return nil
	}

	slog.Info("starting server", "bind", ":8001", "isDaemon", *daemon)

	upstreamURL, err := url.Parse("http://localhost:8000")
	if err != nil {
		log.Fatalf("Failed to parse upstream URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(upstreamURL)
	proxy.Transport = &http.Transport{
		MaxConnsPerHost: 2048,
	}

	// proxy.Transport = &http2.Transport{
	// 	AllowHTTP: true,
	// 	DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
	// 		return net.Dial(network, addr)
	// 	},
	// 	TLSClientConfig: &tls.Config{
	// 		InsecureSkipVerify: true,
	// 	},
	// }

	// handler := &ProxyHandler{
	// 	upstream: upstreamURL,
	// 	proxy:    proxy,
	// }

	// h2s := &http2.Server{}
	// h1s := &http.Server{
	// 	Addr:    fmt.Sprintf(":%d", *port),
	// 	Handler: h2c.NewHandler(handler, h2s),
	// }

	httpServer := &http.Server{
		Handler: proxy,
	}

	go func() {
		ln, err := zeroDT.Listener(":8001")
		if err != nil {
			return
		}
		log.Println("Starting proxy server on :8001")

		if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Error starting server: %v\n", err)
		}
	}()

	go func() {
		for {
			conn, err := net.Dial("tcp", ":8001")
			if err == nil {
				conn.Close()
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

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to shutdown server", "error", err)
		return err
	}

	_ = zeroDT.Close(ctx)

	slog.Info("server is shutdown successfully", "pid", os.Getpid())
	return nil
}

func daemonize(zeroDT *ZeroDownTime) {
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

	_ = zeroDT.WritePID(nil)

	os.Exit(0)
}
