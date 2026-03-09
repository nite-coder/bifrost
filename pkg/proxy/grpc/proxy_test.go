package grpc

import (
	"context"
	"log"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/common/adaptor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/nite-coder/bifrost/proto"
)

type grpcTestServer struct {
	proto.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer.
func (s *grpcTestServer) SayHello(ctx context.Context, in *proto.HelloRequest) (*proto.HelloReply, error) {
	// received metadata
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "failed to get metadata")
	}
	userID := md.Get("user_id")
	if len(userID) == 0 || userID[0] != "5478" {
		return nil, status.Errorf(codes.Unauthenticated, "the user_id is empty or  invalid")
	}

	name := in.GetName()
	log.Printf("Received: %v", name)

	if name == "err" {
		st := status.New(codes.InvalidArgument, "oops....something wrong")

		// add error detail
		detail := &errdetails.ErrorInfo{
			Reason: "INVALID_NAME",
			Domain: "example.com",
			Metadata: map[string]string{
				"message": "Hello error",
			},
		}
		st, _ = st.WithDetails(detail)

		return nil, st.Err()
	}

	if name == "sleep" {
		time.Sleep(3 * time.Second)
	}

	// create and send header
	header := metadata.Pairs("server-name", "bifrost")
	err := grpc.SendHeader(ctx, header)
	if err != nil {
		return nil, err
	}

	return &proto.HelloReply{Message: "Hello " + name}, nil
}

func createGrpcServer() {
	lis, err := net.Listen("tcp", "127.0.0.1:8500")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	proto.RegisterGreeterServer(s, &grpcTestServer{})
	go func() {
		err := s.Serve(lis)
		if err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func TestGRPCProxy(t *testing.T) {
	createGrpcServer()

	ctx := context.Background()

	proxyOptions := Options{
		Target:           "grpc://127.0.0.1:8500",
		TLSVerify:        false,
		Timeout:          1 * time.Second,
		IsTracingEnabled: true,
		Weight:           1,
	}
	proxy, err := New(proxyOptions)
	assert.NoError(t, err)

	httpServer := server.New(
		server.WithH2C(true),
		server.WithHostPorts(":8001"),
		server.WithStreamBody(true),
		server.WithExitWaitTime(1*time.Second),
	)
	httpServer.Use(proxy.ServeHTTP)
	hsrv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := app.NewContext(0)
			err := adaptor.CopyToHertzRequest(r, &c.Request)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			httpServer.ServeHTTP(context.Background(), c)
			c.Response.Header.VisitAll(func(k, v []byte) {
				w.Header().Add(string(k), string(v))
			})

			// Handle Trailers
			trailers := c.Response.Header.Trailer()
			trailers.VisitAll(func(k, v []byte) {
				w.Header().Add("Trailer", string(k))
			})

			w.WriteHeader(c.Response.StatusCode())
			_, _ = w.Write(c.Response.Body())

			// Write Trailer values
			trailers.VisitAll(func(k, v []byte) {
				if len(v) > 0 {
					w.Header().Set(string(k), string(v))
				}
			})
		}),
		Protocols: &http.Protocols{},
	}
	hsrv.Protocols.SetHTTP1(true)
	hsrv.Protocols.SetHTTP2(true)
	hsrv.Protocols.SetUnencryptedHTTP2(true)

	ln, err := net.Listen("tcp", "127.0.0.1:8001")
	require.NoError(t, err)

	go hsrv.Serve(ln)
	assert.Eventually(t, func() bool {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:8001", 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "Server failed to start")

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
	}()

	conn, err := grpc.NewClient("127.0.0.1:8001",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	assert.NoError(t, err)
	defer conn.Close()

	t.Run("success case", func(t *testing.T) {
		client := proto.NewGreeterClient(conn)

		md := metadata.Pairs("user_id", "5478")
		ctx := metadata.NewOutgoingContext(ctx, md)

		var header, trailer metadata.MD // variable to store header and trailer

		resp, err := client.SayHello(ctx,
			&proto.HelloRequest{Name: "Bifrost"},
			grpc.Header(&header),
			grpc.Trailer(&trailer))

		require.NoError(t, err)
		assert.Equal(t, "Hello Bifrost", resp.GetMessage())
		assert.Equal(t, "bifrost", header["server-name"][0])
	})

	t.Run("error case", func(t *testing.T) {
		client := proto.NewGreeterClient(conn)

		md := metadata.Pairs("user_id", "54781")
		ctx := metadata.NewOutgoingContext(ctx, md)

		_, err := client.SayHello(ctx, &proto.HelloRequest{Name: "Bifrost"})
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.Unauthenticated, st.Code())
		assert.Equal(t, "the user_id is empty or  invalid", st.Message())

		md = metadata.Pairs("user_id", "5478")
		ctx = metadata.NewOutgoingContext(ctx, md)
		_, err = client.SayHello(ctx, &proto.HelloRequest{Name: "err"})
		st, ok = status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
		assert.Equal(t, "oops....something wrong", st.Message())
	})

	t.Run("upstream timeout", func(t *testing.T) {
		client := proto.NewGreeterClient(conn)

		md := metadata.Pairs("user_id", "5478")
		ctx := metadata.NewOutgoingContext(ctx, md)
		_, err := client.SayHello(ctx, &proto.HelloRequest{Name: "sleep"})
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.DeadlineExceeded, st.Code())
	})
}

func TestProxyTags(t *testing.T) {
	proxyOptions := Options{
		Target:           "grpc://127.0.0.1:8500",
		TLSVerify:        false,
		Timeout:          1 * time.Second,
		IsTracingEnabled: true,
		Weight:           1,
		Tags: map[string]string{
			"id": "123",
		},
	}
	proxy, err := New(proxyOptions)
	assert.NoError(t, err)

	val, found := proxy.Tag("id")
	assert.True(t, found)
	assert.Equal(t, "123", val)
}

type mockClientConn struct {
	grpc.ClientConnInterface
}

func (m *mockClientConn) Invoke(
	ctx context.Context,
	method string,
	args any,
	reply any,
	opts ...grpc.CallOption,
) error {
	return nil
}

func TestGRPCProxy_PanicOnInvalidPayload(t *testing.T) {
	// Setup
	proxy := &GRPCProxy{
		client: &mockClientConn{}, // minimal mock
		options: &Options{
			Timeout: time.Second,
		},
	}

	// Case: Payload declares length 100, but provides only 10 bytes (header 5 + body 5)
	// gRPC Header: [0 (compression), 0, 0, 0, 100 (length)]
	malformedBody := make([]byte, 10)
	frameHeader := []byte{0, 0, 0, 0, 100} // Length 100
	copy(malformedBody, frameHeader)
	// Remaining 5 bytes are zeros. Total len 10.
	// 5 + 100 = 105. Slice [5:105] will panic on capacity 10 if not checked.

	ctx := context.Background()
	c := app.NewContext(0)
	c.Request.SetRequestURI("/TestService/TestMethod")
	c.Request.SetBody(malformedBody)

	// Execute
	proxy.ServeHTTP(ctx, c)

	// Assert: No panic should occur.
	// Check if status code became 400 (as per our fix)
	assert.Equal(t, 400, c.Response.StatusCode(), "Should return 400 Bad Request for invalid payload length")
}

func TestGRPCProxy_ErrorStatusPlacement(t *testing.T) {
	// Setup
	mockConn := &mockClientConnError{err: status.Error(codes.Unauthenticated, "test error")}
	proxy := &GRPCProxy{
		client: mockConn,
		options: &Options{
			Timeout: time.Second,
		},
	}

	ctx := context.Background()
	c := app.NewContext(0)
	// Minimal valid gRPC body (5 bytes header + 0 bytes body)
	c.Request.SetBody([]byte{0, 0, 0, 0, 0})
	c.Request.SetRequestURI("/TestService/TestMethod")

	// Execute
	proxy.ServeHTTP(ctx, c)

	// Assert
	// grpc-status should NOT be in the regular headers
	assert.Equal(t, "", string(c.Response.Header.Peek("grpc-status")), "grpc-status should not be in headers")
	assert.Equal(t, "", string(c.Response.Header.Peek("grpc-message")), "grpc-message should not be in headers")

	// grpc-status SHOULD be in the trailers
	trailers := c.Response.Header.Trailer()
	assert.Equal(
		t,
		strconv.Itoa(int(codes.Unauthenticated)),
		string(trailers.Peek("grpc-status")),
		"grpc-status should be in trailers",
	)
	assert.Equal(t, "test error", string(trailers.Peek("grpc-message")), "grpc-message should be in trailers")
}

type mockClientConnError struct {
	grpc.ClientConnInterface
	err error
}

func (m *mockClientConnError) Invoke(
	ctx context.Context,
	method string,
	args any,
	reply any,
	opts ...grpc.CallOption,
) error {
	return m.err
}
