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

	upgrader := NewUpgrader(&UpgraderOption{
		Bind:       ":8001",
		SocketPath: "./std.sock",
		PIDFile:    "./std.pid",
	})

	if *upgrade {
		if err := upgrader.Upgrade(); err != nil {
			log.Fatalf("Upgrade failed: %v", err)
		}
		return
	}

	if *daemon {
		upgrader.isDaemon = true
	}

	if *shudown {
		_ = upgrader.Shutdown(ctx)
		return
	}

	done := make(chan bool, 1)
	if os.Getenv("UPGRADE") != "" {
		var err error
		listenerFile := os.NewFile(3, "")
		upgrader.listener, err = net.FileListener(listenerFile)
		if err != nil {
			log.Fatalf("Failed to create listener from file: %v", err)
			return
		}

		go func() {
			<-done

			err := upgrader.Shutdown(ctx)
			if err != nil {
				return
			}

			time.Sleep(2 * time.Second)

			// 写入PID文件
			pid := os.Getpid()
			err = os.WriteFile(upgrader.Options.PIDFile, []byte(fmt.Sprintf("%d", pid)), 0644)
			if err != nil {
				log.Printf("Failed to write PID file: %v", err)
			}
			slog.Info("Write PID file", "path", upgrader.Options.PIDFile, "pid", pid)

			if err := upgrader.WaitForUpgrade(ctx); err != nil {
				log.Fatalf("Upgrade process error: %v", err)
			}
		}()

		slog.Info("get file listener")
		err = startup(ctx, upgrader, done)
		if err != nil {
			slog.Error("Failed to start server", "error", err)
			return
		}
	} else {
		go func() {
			<-done

			if err := upgrader.WaitForUpgrade(ctx); err != nil {
				log.Fatalf("Upgrade process error: %v", err)
			}
		}()

		err := startup(ctx, upgrader, done)
		if err != nil {
			slog.Error("Failed to start server", "error", err)
			return
		}
	}
}

func startup(ctx context.Context, upgrader *Upgrader, done chan bool) (err error) {
	if upgrader.isDaemon && os.Getenv("DAEMONIZED") == "" {
		daemonize(upgrader)
		return nil
	}

	if upgrader.listener == nil {
		upgrader.listener, err = net.Listen("tcp", upgrader.Options.Bind)
		if err != nil {
			return err
		}
	}
	slog.Info("starting server", "bind", ":8001", "isDaemon", upgrader.isDaemon)

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

	upgrader.server = &http.Server{
		Handler: proxy,
	}

	go func() {
		log.Println("Starting proxy server on :8001")
		if err := upgrader.server.Serve(upgrader.listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Error starting server: %v\n", err)
		}
	}()

	go func() {
		for {
			conn, err := net.Dial("tcp", upgrader.Options.Bind)
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

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	_ = upgrader.Close(ctx)

	slog.Info("server is closed", "pid", os.Getpid())
	return nil
}

func daemonize(upgrader *Upgrader) {
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

	// 写入PID文件
	pid := cmd.Process.Pid
	err = os.WriteFile(upgrader.Options.PIDFile, []byte(fmt.Sprintf("%d", pid)), 0644)
	if err != nil {
		log.Printf("Failed to write PID file: %v", err)
	}

	os.Exit(0)
}
