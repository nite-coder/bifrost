package gateway

import "time"

type EntryOptions struct {
	ID           string
	Bind         string
	ReusePort    bool
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	Middlewares  []MiddlwareOptions
	AccessLog    AccessLogOptions
}

type AccessLogOptions struct {
	Enabled    bool
	BufferSize int
	FilePath   string
	Template   string
}

type MiddlwareOptions struct {
	ID     string
	Kind   string
	Params map[string]any
}

type UpstreamStrategy string

const (
	FirstStrategy      UpstreamStrategy = "first"
	RandomStrategy     UpstreamStrategy = "random"
	RoundRobinStrategy UpstreamStrategy = "round_robin"
	WeightedStrategy   UpstreamStrategy = "weighted"
)

type BackendServerOptions struct {
	URL    string
	Weight int
}

type UpstreamOptions struct {
	ID              string
	ClientTransport string
	Strategy        UpstreamStrategy
	Servers         []BackendServerOptions
}

type ClientTransportOptions struct {
	InsecureSkipVerify bool
	MaxConnWaitTimeout time.Duration
	MaxConnsPerHost    int
	KeepAlive          bool
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	DailTimeout        time.Duration
}

type RouteOptions struct {
	ID          string
	Match       string
	Method      []string
	Entries     []string
	Middlewares []MiddlwareOptions
	Upstream    string
}

type Options struct {
	Entries          []EntryOptions
	Routes           []RouteOptions
	Middlewares      []MiddlwareOptions
	Upstreams        []UpstreamOptions
	ClientTransports []ClientTransportOptions
}
