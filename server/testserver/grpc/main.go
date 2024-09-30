package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/nite-coder/bifrost/proto"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var (
	port = flag.Int("port", 8501, "The server port")
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	proto.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *proto.HelloRequest) (*proto.HelloReply, error) {
	name := in.GetName()

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

	log.Printf("Received: %v", name)
	return &proto.HelloReply{Message: "Hello " + name}, nil
}

func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	reflection.Register(s)
	proto.RegisterGreeterServer(s, &server{})
	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
