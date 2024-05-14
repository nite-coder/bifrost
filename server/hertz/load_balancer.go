package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/client"
	"github.com/cloudwego/hertz/pkg/common/config"
	"github.com/hertz-contrib/reverseproxy"
)

type LoadBalancer struct {
	opts    LoadBalancerOptions
	proxies []*reverseproxy.ReverseProxy
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
		client.WithClientReadTimeout(time.Second * 3),
		client.WithWriteTimeout(time.Second * 3),
		client.WithMaxIdleConnDuration(60 * time.Second),
		client.WithMaxConnsPerHost(2000),
		client.WithNoDefaultUserAgentHeader(true),
		client.WithDisableHeaderNamesNormalizing(true),
		client.WithDisablePathNormalizing(true),
		client.WithDialTimeout(5 * time.Second),
	}

	lb := &LoadBalancer{
		opts:    opts,
		proxies: make([]*reverseproxy.ReverseProxy, 0),
	}

	for _, inst := range opts.Instances {
		// TODO: need to verify upstream address schema
		proxy, err := reverseproxy.NewSingleHostReverseProxy(inst.Address, clientOpts...)
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
		fmt.Println("proxy done")
		dur := time.Since(start)
		ctx.Set("$upstream_response_time", dur.Seconds())

	} else {
		fmt.Println("no upstream found")
	}
}
