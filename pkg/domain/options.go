package domain

import "time"

type Options struct {
	Entries     []EntryOptions     `yaml:"entries" json:"entries"`
	Routes      []RouteOptions     `yaml:"routes" json:"routes"`
	Middlewares []MiddlwareOptions `yaml:"middlewares" json:"middlewares"`
	Upstreams   []UpstreamOptions  `yaml:"upstreams" json:"upstreams"`
	Transports  []TransportOptions `yaml:"transports" json:"transports"`
}

type LoggingOtions struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	Level    string `yaml:"level" json:"level"`
	Type     string `yaml:"type" json:"type"`
	FilePath string `yaml:"file_path" json:"file_path"`
}

type EntryOptions struct {
	ID           string             `yaml:"id" json:"id"`
	Bind         string             `yaml:"bind" json:"bind"`
	ReusePort    bool               `yaml:"reuse_port" json:"reuse_port"`
	ReadTimeout  time.Duration      `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration      `yaml:"write_timeout" json:"write_timeout"`
	IdleTimeout  time.Duration      `yaml:"idle_timeout" json:"idle_timeout"`
	Middlewares  []MiddlwareOptions `yaml:"middlewares" json:"middlewares"`
	Logging      *LoggingOtions     `yaml:"logging" json:"logging"`
	AccessLog    AccessLogOptions   `yaml:"access_log" json:"access_log"`
}

type EscapeType string

const (
	NoneEscape    EscapeType = "none"
	DefaultEscape EscapeType = "default"
	JSONEscape    EscapeType = "json"
)

type AccessLogOptions struct {
	Enabled    bool          `yaml:"enabled" json:"enabled"`
	BufferSize int           `yaml:"buffer_size" json:"buffer_size"`
	FilePath   string        `yaml:"file_path" json:"file_path"`
	Template   string        `yaml:"template" json:"template"`
	TimeFormat string        `yaml:"time_format" json:"time_format"`
	Escape     EscapeType    `yaml:"escape" json:"escape"`
	Flush      time.Duration `yaml:"flush" json:"flush"`
}

type MiddlwareOptions struct {
	ID     string         `yaml:"id" json:"id"`
	Kind   string         `yaml:"kind" json:"kind"`
	Params map[string]any `yaml:"params" json:"params"`
	Link   string         `yaml:"link" json:"link"`
}

type UpstreamStrategy string

const (
	FirstStrategy      UpstreamStrategy = "first"
	RandomStrategy     UpstreamStrategy = "random"
	RoundRobinStrategy UpstreamStrategy = "round_robin"
	WeightedStrategy   UpstreamStrategy = "weighted"
)

type BackendServerOptions struct {
	URL    string `yaml:"url" json:"url"`
	Weight int    `yaml:"weight" json:"weight"`
}

type UpstreamOptions struct {
	ID              string                 `yaml:"id" json:"id"`
	ClientTransport string                 `yaml:"client_transport" json:"client_transport"`
	Strategy        UpstreamStrategy       `yaml:"strategy" json:"strategy"`
	Servers         []BackendServerOptions `yaml:"servers" json:"servers"`
}

type TransportOptions struct {
	ID                  string         `yaml:"id" json:"id"`
	InsecureSkipVerify  *bool          `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
	MaxConnWaitTimeout  *time.Duration `yaml:"max_conn_wait_timeout" json:"max_conn_wait_timeout"`
	MaxIdleConnsPerHost *int           `yaml:"max_idle_conns_per_host" json:"max_idle_conns_per_host"`
	KeepAlive           *bool          `yaml:"keep_alive" json:"keep_alive"`
	ReadTimeout         *time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout        *time.Duration `yaml:"write_timeout" json:"write_timeout"`
	DailTimeout         *time.Duration `yaml:"dail_timeout" json:"dail_timeout"`
}

type RouteOptions struct {
	ID          string             `yaml:"id" json:"id"`
	Match       string             `yaml:"match" json:"match"`
	Methods     []string           `yaml:"methods" json:"methods"`
	Entries     []string           `yaml:"entries" json:"entries"`
	Middlewares []MiddlwareOptions `yaml:"middlewares" json:"middlewares"`
	Upstream    string             `yaml:"upstream" json:"upstream"`
}
