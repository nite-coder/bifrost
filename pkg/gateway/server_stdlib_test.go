package gateway

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/nite-coder/bifrost/pkg/config"
	proxygrpc "github.com/nite-coder/bifrost/pkg/proxy/grpc"
	"github.com/nite-coder/bifrost/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Reusing gRPC test server logic from pkg/proxy/grpc/proxy_test.go logic
type testGrpcServer struct {
	proto.UnimplementedGreeterServer
}

func (s *testGrpcServer) SayHello(ctx context.Context, in *proto.HelloRequest) (*proto.HelloReply, error) {
	// Verify Metadata from Client
	md, ok := metadata.FromIncomingContext(ctx)
	headerMD := metadata.Pairs("server-name", "bifrost-stdlib")

	if ok {
		userIDs := md.Get("user_id")
		if len(userIDs) > 0 && userIDs[0] == "12345" {
			// If expected user_id is found, send back a verified signal in header
			headerMD.Set("x-user-verified", "true")
		}
	}

	// Send all headers at once
	_ = grpc.SendHeader(ctx, headerMD)

	// Send a trailer
	_ = grpc.SetTrailer(ctx, metadata.Pairs("trailer-key", "trailer-val"))

	return &proto.HelloReply{Message: "Hello " + in.Name}, nil
}

func startTestBackend(t *testing.T, port string) {
	lis, err := net.Listen("tcp", port)
	require.NoError(t, err)
	s := grpc.NewServer()
	proto.RegisterGreeterServer(s, &testGrpcServer{})
	go func() {
		_ = s.Serve(lis)
	}()
	t.Cleanup(func() { s.Stop() })
}

func TestStdlibServer_GRPC_Integration(t *testing.T) {
	// 1. Start backend gRPC server
	backendAddr := "127.0.0.1:9090"
	startTestBackend(t, backendAddr)

	// 2. Setup Hertz Engine with gRPC Proxy
	h := server.New(server.WithHostPorts(":0")) // Port doesn't matter for Hertz, we use stdlib listener

	proxyOpts := proxygrpc.Options{
		Target:    "grpc://" + backendAddr,
		TLSVerify: false,
		Timeout:   time.Second,
	}
	proxyHandler, err := proxygrpc.New(proxyOpts)
	require.NoError(t, err)

	h.Use(proxyHandler.ServeHTTP)

	// 3. Create Stdlib Server using our new function (unexported, but valid in same package)
	// We need a real listener for stdlib server to serve on
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	serverAddr := ln.Addr().String()

	mockOptions := &config.ServerOptions{
		Timeout: config.ServerTimeoutOptions{
			Read:  time.Second,
			Write: time.Second,
			Idle:  time.Second,
		},
	}
	stdlibSrv := NewStdlibServer(h, mockOptions, nil)

	go func() {
		_ = stdlibSrv.Serve(ln)
	}()
	defer stdlibSrv.Close()

	// Wait a bit for server start
	time.Sleep(100 * time.Millisecond)

	// 4. Create proper gRPC Client to talk to Stdlib Server
	// Note: We used SetUnencryptedHTTP2(true) so h2c should work
	conn, err := grpc.NewClient(serverAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := proto.NewGreeterClient(conn)

	// 5. Make Request with Metadata
	md := metadata.Pairs("user_id", "12345")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	var header, trailer metadata.MD
	resp, err := client.SayHello(ctx, &proto.HelloRequest{Name: "World"},
		grpc.Header(&header), grpc.Trailer(&trailer))

	require.NoError(t, err, "gRPC call failed")

	// 6. Verify Results
	assert.Equal(t, "Hello World", resp.Message)

	// Check Headers
	values := header.Get("server-name")
	if assert.NotEmpty(t, values) {
		assert.Equal(t, "bifrost-stdlib", values[0])
	}

	// Verify if Upstream received user_id and returned x-user-verified
	verified := header.Get("x-user-verified")
	if assert.NotEmpty(t, verified) {
		assert.Equal(t, "true", verified[0])
	}

	// Check Trailers (Critical for this migration)
	// Note: grpc-status is standard trailer. We added custom one too.
	// gRPC client merges trailers.
	// trailer-key must be present if our bridge copied it correctly.
	tValues := trailer.Get("trailer-key")
	if assert.NotEmpty(t, tValues) {
		assert.Equal(t, "trailer-val", tValues[0])
	}
}

func TestStdlibServer_NonGRPC_ContentLength(t *testing.T) {
	h := server.New()
	h.GET("/json", func(ctx context.Context, c *app.RequestContext) {
		c.Response.Header.SetContentType("application/json")
		c.Response.SetBody([]byte(`{"status":"ok"}`))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	mockOptions := &config.ServerOptions{
		Timeout: config.ServerTimeoutOptions{
			Read:  time.Second,
			Write: time.Second,
			Idle:  time.Second,
		},
	}
	stdlibSrv := NewStdlibServer(h, mockOptions, nil)
	go func() {
		_ = stdlibSrv.Serve(ln)
	}()
	defer stdlibSrv.Close()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://" + ln.Addr().String() + "/json")
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.Header.Get("Content-Length") == "" {
		for k, v := range resp.Header {
			t.Logf("Header: %s = %v", k, v)
		}
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Equal(t, "15", resp.Header.Get("Content-Length")) // `{"status":"ok"}` is 15 bytes
}
