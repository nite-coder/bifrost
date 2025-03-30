package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/hertz/pkg/app/client"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func main() {

	options := []hzconfig.ClientOption{
		client.WithNoDefaultUserAgentHeader(true),
		client.WithDisableHeaderNamesNormalizing(true),
		client.WithDisablePathNormalizing(true),
		client.WithDialTimeout(10 * time.Second),
		client.WithClientReadTimeout(60 * time.Second),
		client.WithWriteTimeout(60 * time.Second),
		client.WithMaxIdleConnDuration(120 * time.Second),
		client.WithKeepAlive(true),
		client.WithMaxConnsPerHost(1024),
		//client.WithDialer(standard.NewDialer()),
		client.WithTLSConfig(&tls.Config{
			// when you use ip address to connect to server, you need to set the ServerName to the domain name you want to use
			ServerName:         "echo.free.beeceptor.com", // magic part
			InsecureSkipVerify: false,
		}),
	}

	c, _ := client.NewClient(options...)

	req := &protocol.Request{}
	req.SetMethod(consts.MethodPost)
	req.SetRequestURI("https://147.182.252.2/spot/orders")
	req.Header.SetProtocol("HTTP/1.1")
	req.Header.Set("Host", "echo.free.beeceptor.com")
	req.SetIsTLS(true)

	// req.SetBody([]byte(`{"name": "John Doe"}`))
	// req.Header.Set("content-type", "application/json")

	resp := &protocol.Response{}

	err := c.Do(context.Background(), req, resp)
	if err != nil {
		log.Fatalf("Error sending request: %v", err)
	}

	fmt.Printf("Response: %v\n", string(resp.Body()))

}
