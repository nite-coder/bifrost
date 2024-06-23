package config

import "time"

type Options struct {
	Providers   Provider                    `yaml:"providers" json:"providers"`
	Logging     LoggingOtions               `yaml:"logging" json:"logging"`
	AccessLogs  map[string]AccessLogOptions `yaml:"access_logs" json:"access_logs"`
	Metrics     MetricOptions               `yaml:"metrics" json:"metrics"`
	Tracing     TracingOptions              `yaml:"tracing" json:"tracing"`
	Entries     map[string]EntryOptions     `yaml:"entries" json:"entries"`
	Routes      map[string]RouteOptions     `yaml:"routes" json:"routes"`
	Middlewares map[string]MiddlwareOptions `yaml:"middlewares" json:"middlewares"`
	Services    map[string]ServiceOptions   `yaml:"services" json:"services"`
	Upstreams   map[string]UpstreamOptions  `yaml:"upstreams" json:"upstreams"`
}

type Provider struct {
	File FileProviderOptions `yaml:"file" json:"file"`
}

type FileProviderOptions struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Paths   []string `yaml:"paths" json:"paths"`
	Watch   bool     `yaml:"watch" json:"watch"`
}

type MetricOptions struct {
	Prometheus PrometheusOptions `yaml:"prometheus" json:"prometheus"`
}

type LoggingOtions struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Level   string `yaml:"level" json:"level"`
	Handler string `yaml:"handler" json:"handler"`
	Output  string `yaml:"output" json:"output"`
}

type PrometheusOptions struct {
	Enabled bool      `yaml:"enabled" json:"enabled"`
	Bind    string    `yaml:"bind" json:"bind"`
	Path    string    `yaml:"path" json:"path"`
	Buckets []float64 `yaml:"buckets" json:"buckets"`
}

type TracingOptions struct {
	Enabled bool        `yaml:"enabled" json:"enabled"`
	OTLP    OTLPOptions `yaml:"otlp" json:"otlp"`
}

type OTLPOptions struct {
	HTTP OTLPHTTPOptions `yaml:"http" json:"http"`
	GRPC OTLPGRPCOptions `yaml:"grpc" json:"grpc"`
}

type OTLPHTTPOptions struct {
	Endpoint string `yaml:"endpoint" json:"endpoint"`
}

type OTLPGRPCOptions struct {
	Endpoint string `yaml:"endpoint" json:"endpoint"`
}

type EntryOptions struct {
	ID          string             `yaml:"-" json:"-"`
	Bind        string             `yaml:"bind" json:"bind"`
	TLS         TLSOptions         `yaml:"tls" json:"tls"`
	ReusePort   bool               `yaml:"reuse_port" json:"reuse_port"`
	HTTP2       bool               `yaml:"http2" json:"http2"`
	IdleTimeout time.Duration      `yaml:"idle_timeout" json:"idle_timeout"`
	Middlewares []MiddlwareOptions `yaml:"middlewares" json:"middlewares"`
	Logging     LoggingOtions      `yaml:"logging" json:"logging"`
	AccessLogID string             `yaml:"access_log_id" json:"access_log_id"`
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
	Output     string        `yaml:"output" json:"output"`
	Template   string        `yaml:"template" json:"template"`
	TimeFormat string        `yaml:"time_format" json:"time_format"`
	Escape     EscapeType    `yaml:"escape" json:"escape"`
	Flush      time.Duration `yaml:"flush" json:"flush"`
}

type MiddlwareOptions struct {
	ID     string         `yaml:"-" json:"-"`
	Kind   string         `yaml:"kind" json:"kind"`
	Params map[string]any `yaml:"params" json:"params"`
	Link   string         `yaml:"link" json:"link"`
}

type UpstreamStrategy string

const (
	RandomStrategy     UpstreamStrategy = "random"
	RoundRobinStrategy UpstreamStrategy = "round_robin"
	WeightedStrategy   UpstreamStrategy = "weighted"
)

type TargetOptions struct {
	Target string `yaml:"target" json:"target"`
	Weight int    `yaml:"weight" json:"weight"`
}

type UpstreamOptions struct {
	ID       string           `yaml:"-" json:"-"`
	Strategy UpstreamStrategy `yaml:"strategy" json:"strategy"`
	Targets  []TargetOptions  `yaml:"targets" json:"targets"`
}

type RouteOptions struct {
	ID          string              `yaml:"-" json:"-"`
	Hosts       []string            `yaml:"hosts" json:"hosts"`
	Methods     []string            `yaml:"methods" json:"methods"`
	Paths       []string            `yaml:"paths" json:"paths"`
	Headers     map[string][]string `yaml:"headers" json:"headers"`
	Entries     []string            `yaml:"entries" json:"entries"`
	Middlewares []MiddlwareOptions  `yaml:"middlewares" json:"middlewares"`
	ServiceID   string              `yaml:"service_id" json:"service_id"`
}

type Protocol string

const (
	ProtocolHTTP Protocol = "http"
)

type ServiceOptions struct {
	ID                  string             `yaml:"-" json:"-"`
	TLSVerify           bool               `yaml:"tls_verify" json:"tls_verify"`
	MaxConnWaitTimeout  *time.Duration     `yaml:"max_conn_wait_timeout" json:"max_conn_wait_timeout"`
	MaxIdleConnsPerHost *int               `yaml:"max_idle_conns_per_host" json:"max_idle_conns_per_host"`
	ReadTimeout         *time.Duration     `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout        *time.Duration     `yaml:"write_timeout" json:"write_timeout"`
	DailTimeout         *time.Duration     `yaml:"dail_timeout" json:"dail_timeout"`
	Protocol            Protocol           `yaml:"protocol" json:"protocol"`
	Url                 string             `yaml:"url" json:"url"`
	Middlewares         []MiddlwareOptions `yaml:"middlewares" json:"middlewares"`
}



type TLSOptions struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	MinVersion string `yaml:"min_version" json:"min_version"`
	CertPEM    string `yaml:"cert_pem" json:"cert_pem"`
	KeyPEM     string `yaml:"key_pem" json:"key_pem"`
}
