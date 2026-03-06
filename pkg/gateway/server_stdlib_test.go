package gateway

import (
	"context"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/tracer"
	"github.com/cloudwego/hertz/pkg/common/tracer/stats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/nite-coder/bifrost/pkg/config"
	proxygrpc "github.com/nite-coder/bifrost/pkg/proxy/grpc"
	"github.com/nite-coder/bifrost/proto"
)

// Reusing gRPC test server logic from pkg/proxy/grpc/proxy_test.go logic.
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

	return &proto.HelloReply{Message: "Hello " + in.GetName()}, nil
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
	stdlibSrv := NewStdlibServer(h, mockOptions, nil, nil)

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
	assert.Equal(t, "Hello World", resp.GetMessage())

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
	stdlibSrv := NewStdlibServer(h, mockOptions, nil, nil)
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

// spyTracer is a minimal tracer.Tracer that counts Start/Finish invocations and
// captures the last request context seen in Finish so the test can inspect
// the HTTPStart / HTTPFinish events.
type spyTracer struct {
	startCalls  atomic.Int64
	finishCalls atomic.Int64
	lastCtx     *app.RequestContext
}

func (s *spyTracer) Start(ctx context.Context, c *app.RequestContext) context.Context {
	s.startCalls.Add(1)
	return ctx
}

func (s *spyTracer) Finish(_ context.Context, c *app.RequestContext) {
	s.finishCalls.Add(1)
	s.lastCtx = c
}

// TestStdlibServer_TracerInvoked ensures that, when tracers are registered with
// the HertzBridge, both tracer.Start and tracer.Finish are called for every
// HTTP request (including HTTP/2 / gRPC traffic) and that the TraceInfo on the
// RequestContext carries the HTTPStart and HTTPFinish events.
func TestStdlibServer_TracerInvoked(t *testing.T) {
	spy := &spyTracer{}

	h := server.New(server.WithHostPorts(":0"))
	h.GET("/ping", func(ctx context.Context, c *app.RequestContext) {
		c.Response.SetBodyString("pong")
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

	stdlibSrv := NewStdlibServer(h, mockOptions, nil, []tracer.Tracer{spy})
	go func() { _ = stdlibSrv.Serve(ln) }()
	defer stdlibSrv.Close()
	time.Sleep(50 * time.Millisecond)

	resp, err := http.Get("http://" + ln.Addr().String() + "/ping")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Give the deferred doFinish goroutine a chance to run.
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, int64(1), spy.startCalls.Load(), "tracer.Start should be called once")
	assert.Equal(t, int64(1), spy.finishCalls.Load(), "tracer.Finish should be called once")

	require.NotNil(t, spy.lastCtx, "lastCtx must be set by Finish")
	ti := spy.lastCtx.GetTraceInfo()
	require.NotNil(t, ti, "TraceInfo must be present on RequestContext")

	httpStart := ti.Stats().GetEvent(stats.HTTPStart)
	httpFinish := ti.Stats().GetEvent(stats.HTTPFinish)
	assert.NotNil(t, httpStart, "HTTPStart event should be recorded")
	assert.NotNil(t, httpFinish, "HTTPFinish event should be recorded")
	assert.True(t, httpFinish.Time().After(httpStart.Time()),
		"HTTPFinish must be after HTTPStart")
}
