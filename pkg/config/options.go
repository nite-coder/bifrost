package config

import "time"

type Options struct {
	Providers   ProviderOtions              `yaml:"providers" json:"providers"`
	Logging     LoggingOtions               `yaml:"logging" json:"logging"`
	Metrics     MetricsOptions              `yaml:"metrics" json:"metrics"`
	Tracing     TracingOptions              `yaml:"tracing" json:"tracing"`
	AccessLogs  map[string]AccessLogOptions `yaml:"access_logs" json:"access_logs"`
	Servers     map[string]ServerOptions    `yaml:"servers" json:"servers"`
	Routes      map[string]RouteOptions     `yaml:"routes" json:"routes"`
	Middlewares map[string]MiddlwareOptions `yaml:"middlewares" json:"middlewares"`
	Services    map[string]ServiceOptions   `yaml:"services" json:"services"`
	Upstreams   map[string]UpstreamOptions  `yaml:"upstreams" json:"upstreams"`
}

type ProviderOtions struct {
	File FileProviderOptions `yaml:"file" json:"file"`
}

type FileProviderOptions struct {
	Enabled bool     `yaml:"enabled" json:"enabled"`
	Paths   []string `yaml:"paths" json:"paths"`
	Watch   bool     `yaml:"watch" json:"watch"`
}

type MetricsOptions struct {
	Prometheus PrometheusOptions `yaml:"prometheus" json:"prometheus"`
}

type LoggingOtions struct {
	Level   string `yaml:"level" json:"level"`
	Handler string `yaml:"handler" json:"handler"`
	Output  string `yaml:"output" json:"output"`
}

type PrometheusOptions struct {
	Enabled bool      `yaml:"enabled" json:"enabled"`
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

type ServerOptions struct {
	ID                 string               `yaml:"-" json:"-"`
	Bind               string               `yaml:"bind" json:"bind"`
	TLS                TLSOptions           `yaml:"tls" json:"tls"`
	ReusePort          bool                 `yaml:"reuse_port" json:"reuse_port"`
	HTTP2              bool                 `yaml:"http2" json:"http2"`
	Middlewares        []MiddlwareOptions   `yaml:"middlewares" json:"middlewares"`
	Logging            LoggingOtions        `yaml:"logging" json:"logging"`
	Timeout            ServerTimeoutOptions `yaml:"timeout" json:"timeout"`
	MaxRequestBodySize int                  `yaml:"max_request_body_size" json:"max_request_body_size"`
	ReadBufferSize     int                  `yaml:"read_buffer_size" json:"read_buffer_size"`
	PPROF              bool                 `yaml:"pprof" json:"pprof"`
	AccessLogID        string               `yaml:"access_log_id" json:"access_log_id"`
}

type ServerTimeoutOptions struct {
	GracefulTimeOut  time.Duration `yaml:"graceful_timeout" json:"graceful_timeout"`
	IdleTimeout      time.Duration `yaml:"idle_timeout" json:"idle_timeout"`
	KeepAliveTimeout time.Duration `yaml:"keepalive_timeout" json:"keepalive_timeout"`
	ReadTimeout      time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout     time.Duration `yaml:"write_timeout" json:"write_timeout"`
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
	Type   string         `yaml:"type" json:"type"`
	Params map[string]any `yaml:"params" json:"params"`
	Use    string         `yaml:"use" json:"use"`
}

type UpstreamStrategy string

const (
	RandomStrategy     UpstreamStrategy = "random"
	RoundRobinStrategy UpstreamStrategy = "round_robin"
	WeightedStrategy   UpstreamStrategy = "weighted"
	HashingStrategy    UpstreamStrategy = "hashing"
)

type TargetOptions struct {
	Target string `yaml:"target" json:"target"`
	Weight int    `yaml:"weight" json:"weight"`
}

type UpstreamOptions struct {
	ID       string           `yaml:"-" json:"-"`
	Strategy UpstreamStrategy `yaml:"strategy" json:"strategy"`
	HashOn   string           `yaml:"hash_on" json:"hash_on"`
	Targets  []TargetOptions  `yaml:"targets" json:"targets"`
}

type RouteOptions struct {
	ID          string             `yaml:"-" json:"-"`
	Methods     []string           `yaml:"methods" json:"methods"`
	Paths       []string           `yaml:"paths" json:"paths"`
	Servers     []string           `yaml:"servers" json:"servers"`
	Middlewares []MiddlwareOptions `yaml:"middlewares" json:"middlewares"`
	ServiceID   string             `yaml:"service_id" json:"service_id"`
}

type Protocol string

const (
	ProtocolHTTP Protocol = "http"
)

type ServiceOptions struct {
	ID                  string                `yaml:"-" json:"-"`
	TLSVerify           bool                  `yaml:"tls_verify" json:"tls_verify"`
	MaxIdleConnsPerHost *int                  `yaml:"max_idle_conns_per_host" json:"max_idle_conns_per_host"`
	Protocol            Protocol              `yaml:"protocol" json:"protocol"`
	Url                 string                `yaml:"url" json:"url"`
	Timeout             ServiceTimeoutOptions `yaml:"timeout" json:"timeout"`
	Middlewares         []MiddlwareOptions    `yaml:"middlewares" json:"middlewares"`
}

type ServiceTimeoutOptions struct {
	ReadTimeout        time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout       time.Duration `yaml:"write_timeout" json:"write_timeout"`
	DailTimeout        time.Duration `yaml:"dail_timeout" json:"dail_timeout"`
	MaxConnWaitTimeout time.Duration `yaml:"max_conn_wait_timeout" json:"max_conn_wait_timeout"`
}

type TLSOptions struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	MinVersion string `yaml:"min_version" json:"min_version"`
	CertPEM    string `yaml:"cert_pem" json:"cert_pem"`
	KeyPEM     string `yaml:"key_pem" json:"key_pem"`
}
