package gateway

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/common/config"
)

type LoadBalancer struct {
	opts    LoadBalancerOptions
	proxies []*ReverseProxy
}

type Instance struct {
	Address string
	Weight  int
}

type LoadBalancerOptions struct {
	ID        string
	Instances []Instance
}

func NewLoadBalancer(opts LoadBalancerOptions) *LoadBalancer {

	clientOpts := []config.ClientOption{
		client.WithNoDefaultUserAgentHeader(true),
		client.WithDisableHeaderNamesNormalizing(true),
		client.WithDisablePathNormalizing(true),

		client.WithDialTimeout(3 * time.Second),
		client.WithClientReadTimeout(3 * time.Second),
		client.WithWriteTimeout(3 * time.Second),
		//client.WithMaxConnWaitTimeout(3 * time.Second),
		client.WithMaxConnsPerHost(math.MaxInt),

		//client.WithMaxIdleConns(100),
		//client.WithMaxIdleConnDuration(60 * time.Second),
		//client.WithKeepAlive(true),
		//client.WithMaxConnDuration(120 * time.Second),
	}

	lb := &LoadBalancer{
		opts:    opts,
		proxies: make([]*ReverseProxy, 0),
	}

	for _, inst := range opts.Instances {
		// TODO: need to verify upstream address schema
		proxy, err := NewSingleHostReverseProxy(inst.Address, clientOpts...)
		if err != nil {
			panic(err)
		}
		lb.proxies = append(lb.proxies, proxy)
	}

	return lb
}

func (lb *LoadBalancer) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	ctx.Abort()
	proxy := lb.proxies[0]
	if proxy != nil {
		start := time.Now()
		proxy.ServeHTTP(c, ctx)
		//fmt.Println("proxy done")
		dur := time.Since(start)
		ctx.Set(UPSTREAM_RESPONSE_TIME, dur.Seconds())

	} else {
		fmt.Println("no upstream found")
	}
}
