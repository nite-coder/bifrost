package provider

import (
	"context"
	"errors"
	"net"
)

// ErrWatchNotSupported is returned when a service discovery provider does not support watching.
var ErrWatchNotSupported = errors.New("watch is not supported")

// ChangeFunc is a function that is called when configuration changes.
type ChangeFunc func() error

// Provider defines the interface for configuration providers.
type Provider interface {
	Watch() error
	SetOnChanged(f ChangeFunc)
}

// GetInstanceOptions defines the options for retrieving service instances.
type GetInstanceOptions struct {
	Namespace string
	Name      string
	Group     string
}

// DiscoveryResult preserves the target→instances grouping from discovery.
type DiscoveryResult struct {
	Target string            // hostname:port (from config TargetOptions.Target, or discovery service name)
	Weight uint32            // target-level weight
	Tags   map[string]string // target-level tags
	Nodes  []Instancer       // resolved instances for this target
}

// ServiceDiscovery defines the interface for service discovery.
type ServiceDiscovery interface {
	GetInstances(ctx context.Context, options GetInstanceOptions) ([]DiscoveryResult, error)
	Watch(ctx context.Context, options GetInstanceOptions) (<-chan []DiscoveryResult, error)
	Close() error
}

// Instancer defines the interface for a service instance.
type Instancer interface {
	Address() net.Addr
	Weight() uint32
	Tag(key string) (value string, exist bool)
	Tags() map[string]string
}

// Instance represents a single service instance with an address and metadata.
type Instance struct {
	addr     net.Addr
	metadata map[string]string
	weight   uint32
}

// NewInstance creates a new service instance with the given address and weight.
func NewInstance(addr net.Addr, weight uint32) *Instance {
	return &Instance{
		addr:     addr,
		weight:   weight,
		metadata: make(map[string]string),
	}
}

// Address returns the network address of the instance.
func (i *Instance) Address() net.Addr {
	return i.addr
}

// Weight returns the relative weight of the instance.
func (i *Instance) Weight() uint32 {
	if i.weight <= 0 {
		return 1
	}
	return i.weight
}

// Tag retrieves a specific metadata tag value from the instance.
func (i *Instance) Tag(key string) (value string, exist bool) {
	val, found := i.metadata[key]
	return val, found
}

// SetTag sets a metadata tag for the instance.
func (i *Instance) SetTag(key string, value string) {
	i.metadata[key] = value
}

// Tags returns all metadata tags of the instance.
func (i *Instance) Tags() map[string]string {
	return i.metadata
}
