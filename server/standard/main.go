package main

import (
	"flag"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

var (
	port     = flag.Int("port", 8001, "The proxy server port")
	upstream = flag.String("upstream", "http://localhost:8000", "The upstream server URL")
)

type ProxyHandler struct {
	upstream *url.URL
	proxy    *httputil.ReverseProxy
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//log.Printf("Received request: %s %s %s", r.Method, r.URL, r.Proto)

	// if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
	// 	r.Header.Set("X-Forwarded-Proto", "h2c")
	// }

	h.proxy.ServeHTTP(w, r)
}

func main() {
	flag.Parse()

	upstreamURL, err := url.Parse(*upstream)
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

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	})

	// 創建並啟動 http.Server
	server := &http.Server{
		Addr:    ":8001",
		Handler: handler,
	}

	log.Println("Starting proxy server on :8001")
	log.Fatal(server.ListenAndServe())
}
