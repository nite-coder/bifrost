package http

import (
	"crypto/tls"
	"time"

	"github.com/cloudwego/hertz/pkg/app/client"
	hzconfig "github.com/cloudwego/hertz/pkg/common/config"
)

const (
	defaultDialTimeout         = 10 * time.Second
	defaultReadTimeout         = 60 * time.Second
	defaultWriteTimeout        = 60 * time.Second
	defaultMaxIdleConnDuration = 120 * time.Second
	defaultMaxConnsPerHost     = 1024
)

// DefaultClientOptions returns a set of default Hertz client options for proxying.
func DefaultClientOptions() []hzconfig.ClientOption {
	options := []hzconfig.ClientOption{
		client.WithNoDefaultUserAgentHeader(true),
		client.WithDisableHeaderNamesNormalizing(true),
		client.WithDisablePathNormalizing(true),
		client.WithDialTimeout(defaultDialTimeout),
		client.WithClientReadTimeout(defaultReadTimeout),
		client.WithWriteTimeout(defaultWriteTimeout),
		client.WithMaxIdleConnDuration(defaultMaxIdleConnDuration),
		client.WithKeepAlive(true),
		client.WithMaxConnsPerHost(defaultMaxConnsPerHost),
		client.WithResponseBodyStream(true),
	}
	return options
}

// ClientOptions defines the configuration for a Hertz HTTP client.
type ClientOptions struct {
	HZOptions []hzconfig.ClientOption
	IsHTTP2   bool
}

// NewClient creates a new Hertz client with the given options.
func NewClient(opts ClientOptions) (*client.Client, error) {
	var tlsConfig *tls.Config
	wrappedOptions := make([]hzconfig.ClientOption, len(opts.HZOptions))
	for i, opt := range opts.HZOptions {
		copyOpt := opt
		wrappedOptions[i] = hzconfig.ClientOption{
			F: func(o *hzconfig.ClientOptions) {
				copyOpt.F(o)
				if o.TLSConfig != nil {
					tlsConfig = o.TLSConfig
				}
			},
		}
	}

	c, err := client.NewClient(wrappedOptions...)
	if err != nil {
		return nil, err
	}
	if opts.IsHTTP2 {
		c.SetClientFactory(NewClientFactory(tlsConfig))
	}
	return c, nil
}
