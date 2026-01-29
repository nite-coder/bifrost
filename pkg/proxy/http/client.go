package http

import (
	"time"

	"github.com/cloudwego/hertz/pkg/app/client"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
)

func DefaultClientOptions() []hzconfig.ClientOption {
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
		client.WithResponseBodyStream(true),
	}
	return options
}

type ClientOptions struct {
	HZOptions []hzconfig.ClientOption
	IsHTTP2   bool
}

func NewClient(opts ClientOptions) (*client.Client, error) {
	c, err := client.NewClient(opts.HZOptions...)
	if err != nil {
		return nil, err
	}
	if opts.IsHTTP2 {
		c.SetClientFactory(NewClientFactory())
	}
	return c, nil
}
