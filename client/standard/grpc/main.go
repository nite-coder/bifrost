package main

import (
	"context"
	"http-benchmark/proto"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	client, err := grpc.NewClient("127.0.0.1:8001", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}

	gClient := proto.NewGreeterClient(client)

	req := proto.HelloRequest{
		Name: "gprc test",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reply, err := gClient.SayHello(ctx, &req)
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			slog.Error("fail to say hello", "code", st.Code(), "error", st.Message())
			return
		}
	}

	slog.Info("result:", "msg", reply.Message)
}
