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
	GetInstances(ctx context.Context, serviceName string) ([]Instancer, error)
	Watch(ctx context.Context, serviceName string) (<-chan []Instancer, error)
}

type Instancer interface {
	Address() net.Addr
	Weight() uint32
	Tag(key string) (value string, exist bool)
}

type Instance struct {
	addr     net.Addr
	weight   uint32
	metadata map[string]string
}

func NewInstance(addr net.Addr, weight uint32) *Instance {
	return &Instance{
		addr:     addr,
		weight:   weight,
		metadata: map[string]string{},
	}
}

func (i *Instance) Address() net.Addr {
	return i.addr
}

func (i *Instance) Weight() uint32 {
	return i.weight
}

func (i *Instance) Tag(key string) (value string, exist bool) {
	val, found := i.metadata[key]
	return val, found
}

func (i *Instance) SetTag(key string, value string) {
	i.metadata[key] = value
}
