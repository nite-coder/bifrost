package http

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"

	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/cloudwego/hertz/pkg/protocol"
	hclient "github.com/cloudwego/hertz/pkg/protocol/client"
	"github.com/cloudwego/hertz/pkg/protocol/suite"
	"github.com/nite-coder/bifrost/internal/pkg/hzadaptor"
)

// stdlibFactory implements suite.ClientFactory
type stdlibFactory struct {
	tlsConfig *tls.Config
}

// NewClientFactory creates a new stdlibFactory
func NewClientFactory(tlsConfig *tls.Config) suite.ClientFactory {
	return &stdlibFactory{
		tlsConfig: tlsConfig,
	}
}

func (f *stdlibFactory) NewHostClient() (hclient.HostClient, error) {
	return &stdlibHostClient{
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:   f.tlsConfig,
				ForceAttemptHTTP2: true,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

// stdlibHostClient implements protocol/client.HostClient
type stdlibHostClient struct {
	client *http.Client
}

func (c *stdlibHostClient) SetClientConfig(o *config.ClientOptions) {
	if o != nil {
		c.client.Timeout = o.ReadTimeout
		if o.TLSConfig != nil {
			if t, ok := c.client.Transport.(*http.Transport); ok {
				t.TLSClientConfig = o.TLSConfig
			}
		}
	}
}

func (c *stdlibHostClient) Do(ctx context.Context, req *protocol.Request, resp *protocol.Response) error {
	// 1. Convert Hertz Request to net/http Request
	stdReq, err := hzadaptor.ToHTTPRequest(ctx, req)
	if err != nil {
		return err
	}

	// 2. Execute Request
	stdResp, err := c.client.Do(stdReq)
	if err != nil {
		return err
	}
	defer stdResp.Body.Close()

	// 3. Convert net/http Response to Hertz Response
	resp.SetStatusCode(stdResp.StatusCode)

	hHeader := &resp.Header
	for k, vv := range stdResp.Header {
		for _, v := range vv {
			hHeader.Add(k, v)
		}
	}

	body, err := io.ReadAll(stdResp.Body)
	if err != nil {
		return err
	}
	resp.SetBody(body)

	// Copy Trailers
	for k, vv := range stdResp.Trailer {
		for _, v := range vv {
			_ = hHeader.Trailer().Add(k, v)
			// Also add to header for compatibility with some consumers that don't check trailers
			hHeader.Add(k, v)
		}
	}

	return nil
}

func (c *stdlibHostClient) SetDynamicConfig(dc *hclient.DynamicConfig) {
}

func (c *stdlibHostClient) ShouldRemove() bool {
	return false
}

func (c *stdlibHostClient) ConnectionCount() int {
	return 0
}

func (c *stdlibHostClient) CloseIdleConnections() {
	c.client.CloseIdleConnections()
}
