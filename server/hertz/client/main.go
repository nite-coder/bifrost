package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app/client"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/protocol"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	reqHelper "github.com/cloudwego/hertz/pkg/protocol/http1/req"
	respHelper "github.com/cloudwego/hertz/pkg/protocol/http1/resp"
)

func main() {

	clientOpts := newDefaultClientOptions()
	c, _ := client.NewClient(clientOpts...)

	dailer := c.GetOptions().Dialer
	conn, err := dailer.DialConnection("tcp", "127.0.0.1:8000", 10*time.Second, nil)
	if err != nil {
		panic(err)
	}

	req := &protocol.Request{}
	resp := &protocol.Response{}
	defer func() {
		protocol.ReleaseRequest(req)
		protocol.ReleaseResponse(resp)
	}()
	req.SetRequestURI("http://127.0.0.1:8000/websocket")
	req.SetMethod(consts.MethodGet)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "x3JJHMbDL1EzLkh9GBhXDw==")

	err = reqHelper.Write(req, conn)
	if err != nil {
		panic(err)
	}

	err = conn.Flush()
	if err != nil {
		panic(err)
	}

	err = respHelper.ReadHeaderAndLimitBody(resp, conn, 1024000)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode() != 101 {
		slog.Info("status code", "code", resp.StatusCode())
		return
	}

	reader := bufio.NewReader(conn)

	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("connection is closed")
			} else {
				fmt.Println("read err:", err)
			}
			return
		}
		fmt.Println("received:", strings.TrimSpace(message))
	}

}

func newDefaultClientOptions() []hzconfig.ClientOption {
	return []hzconfig.ClientOption{
		client.WithNoDefaultUserAgentHeader(true),
		client.WithDisableHeaderNamesNormalizing(true),
		client.WithDisablePathNormalizing(true),
		client.WithDialTimeout(10 * time.Second),
		client.WithClientReadTimeout(60 * time.Second),
		client.WithWriteTimeout(60 * time.Second),
		client.WithMaxIdleConnDuration(120 * time.Second),
		client.WithKeepAlive(true),
		client.WithMaxConnsPerHost(1024),
	}
}
