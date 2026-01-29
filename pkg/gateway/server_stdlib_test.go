package gateway

import (
	"context"
	"net"
	"testing"
	"time"

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
	// Send a header
	_ = grpc.SendHeader(ctx, metadata.Pairs("server-name", "bifrost-stdlib"))
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

	// 5. Make Request
	var header, trailer metadata.MD
	resp, err := client.SayHello(context.Background(), &proto.HelloRequest{Name: "World"},
		grpc.Header(&header), grpc.Trailer(&trailer))

	require.NoError(t, err, "gRPC call failed")

	// 6. Verify Results
	assert.Equal(t, "Hello World", resp.Message)

	// Check Headers
	values := header.Get("server-name")
	if assert.NotEmpty(t, values) {
		assert.Equal(t, "bifrost-stdlib", values[0])
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
