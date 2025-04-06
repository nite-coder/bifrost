package provider

import (
	"context"
	"net"
)

type ChangeFunc func() error

type Provider interface {
	Watch() error
	SetOnChanged(ChangeFunc)
}

type ServiceDiscovery interface {
	GetInstances(ctx context.Context, serviceName string) ([]Instance, error)
	Watch(ctx context.Context, serviceName string) (<-chan []Instance, error)
}

type Instance interface {
	Address() net.Addr
	Weight() int
	Tag(key string) (value string, exist bool)
}
