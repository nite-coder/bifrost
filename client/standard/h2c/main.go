package main

import (
	"bytes"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"

	"golang.org/x/net/http2"
	"google.golang.org/protobuf/proto"

	model "github.com/nite-coder/bifrost/proto"
)

func main() {
	client := &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}

	reqMsg := model.HelloRequest{Name: "hertz"}
	data, err := proto.Marshal(&reqMsg)
	if err != nil {
		panic(err)
	}

	framedData := addGrpcPrefix(data)

	req, err := http.NewRequest(
		"POST",
		"http://localhost:8003/helloworld.Greeter/SayHello",
		bytes.NewReader(framedData),
	)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/grpc")
	req.Header.Set("te", "trailers")

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	responseData := removeGrpcPrefix(body)

	replyData := model.HelloReply{}
	err = proto.Unmarshal(responseData, &replyData)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Response: %v\n", replyData.GetMessage())

	for key, val := range resp.Header {
		// Logic using key
		// And val if you need it
		fmt.Printf("Header: %s: %v\n", key, val)
	}

	// handle trailer
	for k, v := range resp.Trailer {
		fmt.Printf("Trailer2: %s: %v\n", k, v)
	}
}

func addGrpcPrefix(data []byte) []byte {
	if len(data) > math.MaxUint32-5 {
		// This should not happen in this example, but good for security
		return data
	}
	prefix := make([]byte, 5+len(data))
	binary.BigEndian.PutUint32(prefix[1:], uint32(len(data)))
	copy(prefix[5:], data)
	return prefix
}

func removeGrpcPrefix(data []byte) []byte {
	if len(data) < 5 {
		return data
	}
	return data[5:]
}
