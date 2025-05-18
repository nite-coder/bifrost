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
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/nite-coder/bifrost/pkg/zero"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type ProxyHandler struct {
	proxy *httputil.ReverseProxy
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	zeroOpts := zero.Options{
		UpgradeSock: "./std.sock",
		PIDFile:     "./std.pid",
	}

	zeroDT := zero.New(zeroOpts)

	if *upgrade {
		if err := zeroDT.Upgrade(); err != nil {
			log.Fatalf("Upgrade failed: %v", err)
		}
		return
	}

	if *shudown {
		oldPID, err := zeroDT.GetPID()
		if err != nil {
			slog.Error("shutdown error", "error", err)
			return
		}
		_ = zeroDT.Quit(ctx, oldPID, true)
		return
	}

	done := make(chan bool, 1)
	if zeroDT.IsUpgraded() {
		go func() {
			<-done

			oldPID, err := zeroDT.GetPID()
			if err != nil {
				slog.Error("failed to upgrade", "error", err)
				return
			}

			err = zeroDT.Quit(ctx, oldPID, false)
			if err != nil {
				return
			}

			time.Sleep(5 * time.Second)

			if *daemon {
				err = zeroDT.WritePID()
				if err != nil {
					slog.Error("Upgrade process error", "error", err)
					return
				}
				if err := zeroDT.WaitForUpgrade(ctx); err != nil {
					slog.Error("Upgrade process error", "error", err)
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
					slog.Error("Upgrade process", "error", err)
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

	upstreamURL, err := url.Parse("http://localhost:8000")
	if err != nil {
		log.Fatalf("Failed to parse upstream URL: %v", err)
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
		ln, err := zeroDT.Listener(ctx, "tcp", ":8001", nil)
		if err != nil {
			return
		}
		slog.Info("starting server", "bind", ":8001", "isDaemon", *daemon)

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

	slog.Info("all servers are shutdown successfully", "pid", os.Getpid())
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
