package balancer

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/proxy"
)

var (
	ErrNoAvailable                                  = errors.New("no available upstream at the moment")
	balancers      map[string]CreateBalancerHandler = make(map[string]CreateBalancerHandler)
)

type CreateBalancerHandler func(proxies []proxy.Proxy, params any) (Balancer, error)

type Balancer interface {
	Proxies() []proxy.Proxy
	Select(ctx context.Context, hzCtx *app.RequestContext) (proxy.Proxy, error)
}

func Register(name string, h CreateBalancerHandler) error {
	if _, found := balancers[name]; found {
		return fmt.Errorf("balancer '%s' already exists", name)
	}

	balancers[name] = h
	return nil
}

func Factory(name string) CreateBalancerHandler {
	if name == "" {
		name = "round_robin"
	}

	return balancers[name]
}
