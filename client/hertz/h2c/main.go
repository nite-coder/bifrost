package main

import (
	"context"
	"encoding/binary"
	"fmt"
	model "http-benchmark/proto"
	"io"
	"log"

	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/hertz-contrib/http2/config"
	"github.com/hertz-contrib/http2/factory"
	"google.golang.org/protobuf/proto"
)

func main() {

	c, _ := client.NewClient()
	c.SetClientFactory(factory.NewClientFactory(config.WithAllowHTTP(true)))

	reqMsg := model.HelloRequest{Name: "hertz"}
	data, err := proto.Marshal(&reqMsg)
	if err != nil {
		panic(err)
	}

	framedData := addGrpcPrefix(data)

	req := &protocol.Request{}
	req.SetMethod(consts.MethodPost)
	req.SetRequestURI("http://localhost:8003/helloworld.Greeter/SayHello")
	req.SetBody(framedData)

	req.Header.Set("content-type", "application/grpc")
	req.Header.Set("te", "trailers")

	resp := &protocol.Response{}

	err = c.Do(context.Background(), req, resp)
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}

	body, err := io.ReadAll(resp.BodyStream())
	if err != nil {
		log.Fatalf("Error reading response: %v", err)
	}

	responseData := removeGrpcPrefix(body)

	replyData := model.HelloReply{}
	err = proto.Unmarshal(responseData, &replyData)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Response: %v\n", replyData.Message)

	resp.Header.Trailer().VisitAll(func(key, value []byte) {
		fmt.Printf("Trailer: %s: %s\n", key, value)
	})
}

func addGrpcPrefix(data []byte) []byte {
	prefix := make([]byte, 5)
	binary.BigEndian.PutUint32(prefix[1:], uint32(len(data)))
	return append(prefix, data...)
}

func removeGrpcPrefix(data []byte) []byte {
	if len(data) < 5 {
		return data
	}
	return data[5:]
}
