package grpc

import (
	"context"
	"log"
	"net"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/hertz-contrib/http2/factory"
	"github.com/nite-coder/bifrost/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type grpcTestServer struct {
	proto.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
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
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()
}

func TestGRPCProxy(t *testing.T) {
	createGrpcServer()

	ctx := context.Background()

	proxyOptions := Options{
		Target:    "grpc://127.0.0.1:8500",
		TLSVerify: false,
		Timeout:   1 * time.Second,
		Weight:    1,
	}
	proxy, err := New(proxyOptions)
	assert.NoError(t, err)

	httpServer := server.New(
		server.WithH2C(true),
		server.WithHostPorts(":8001"),
		server.WithStreamBody(true),
		server.WithExitWaitTime(1*time.Second),
	)
	httpServer.AddProtocol("h2", factory.NewServerFactory())
	httpServer.Use(proxy.ServeHTTP)

	go httpServer.Spin()
	time.Sleep(1 * time.Second)

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
		ctx = metadata.NewOutgoingContext(ctx, md)

		var header, trailer metadata.MD // variable to store header and trailer

		resp, err := client.SayHello(ctx,
			&proto.HelloRequest{Name: "Bifrost"},
			grpc.Header(&header),
			grpc.Trailer(&trailer))

		assert.NoError(t, err)
		assert.Equal(t, "Hello Bifrost", resp.Message)
		assert.Equal(t, "bifrost", header["server-name"][0])
	})

	t.Run("error case", func(t *testing.T) {
		client := proto.NewGreeterClient(conn)

		md := metadata.Pairs("user_id", "54781")
		ctx = metadata.NewOutgoingContext(ctx, md)

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
		ctx = metadata.NewOutgoingContext(ctx, md)
		_, err := client.SayHello(ctx, &proto.HelloRequest{Name: "sleep"})
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.DeadlineExceeded, st.Code())
	})
}
