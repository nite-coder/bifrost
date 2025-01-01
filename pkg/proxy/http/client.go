package http

import (
	"time"

	"github.com/cloudwego/hertz/pkg/app/client"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
	http2Config "github.com/hertz-contrib/http2/config"
	"github.com/hertz-contrib/http2/factory"
)

func DefaultClientOptions() []hzconfig.ClientOption {
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
		client.WithResponseBodyStream(true),
	}
}

type ClientOptions struct {
	IsTracingEnabled bool
	IsHTTP2          bool
	HZOptions        []hzconfig.ClientOption
}

func NewClient(opts ClientOptions) (*client.Client, error) {
	c, err := client.NewClient(opts.HZOptions...)
	if err != nil {
		return nil, err
	}

	if opts.IsHTTP2 {
		c.SetClientFactory(factory.NewClientFactory(http2Config.WithAllowHTTP(true)))
	}

	return c, nil
}
