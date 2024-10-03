package config

import "time"

type Options struct {
	Version         string                      `yaml:"version" json:"version"`
	PIDFile         string                      `yaml:"pid_file" json:"pid_file"`
	UpgradeSock     string                      `yaml:"upgrade_sock" json:"upgrade_sock"`
	Gopool          bool                        `yaml:"gopool" json:"gopool"`
	IsDaemon        bool                        `yaml:"-" json:"-"`
	User            string                      `yaml:"user" json:"user"`
	Group           string                      `yaml:"group" json:"group"`
	Providers       ProviderOtions              `yaml:"providers" json:"providers"`
	TimerResolution time.Duration               `yaml:"timer_resolution " json:"timer_resolution "`
	Logging         LoggingOtions               `yaml:"logging" json:"logging"`
	Metrics         MetricsOptions              `yaml:"metrics" json:"metrics"`
	Tracing         TracingOptions              `yaml:"tracing" json:"tracing"`
	AccessLogs      map[string]AccessLogOptions `yaml:"access_logs" json:"access_logs"`
	Servers         map[string]ServerOptions    `yaml:"servers" json:"servers"`
	Routes          map[string]RouteOptions     `yaml:"routes" json:"routes"`
	Middlewares     map[string]MiddlwareOptions `yaml:"middlewares" json:"middlewares"`
	Services        map[string]ServiceOptions   `yaml:"services" json:"services"`
	Upstreams       map[string]UpstreamOptions  `yaml:"upstreams" json:"upstreams"`
}

func NewOptions() Options {
	mainOptions := Options{
		Version:     "1",
		AccessLogs:  make(map[string]AccessLogOptions),
		Servers:     make(map[string]ServerOptions),
		Routes:      make(map[string]RouteOptions),
		Middlewares: make(map[string]MiddlwareOptions),
		Services:    make(map[string]ServiceOptions),
		Upstreams:   make(map[string]UpstreamOptions),
	}

	return mainOptions
}

type ProviderOtions struct {
	File FileProviderOptions `yaml:"file" json:"file"`
}

type FileProviderOptions struct {
	Enabled    bool     `yaml:"enabled" json:"enabled"`
	Paths      []string `yaml:"paths" json:"paths"`
	Watch      bool     `yaml:"watch" json:"watch"`
	Extensions []string `yaml:"extensions" json:"extensions"`
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
	OTLP OTLPOptions `yaml:"otlp" json:"otlp"`
}

type OTLPOptions struct {
	Enabled bool            `yaml:"enabled" json:"enabled"`
	HTTP    OTLPHTTPOptions `yaml:"http" json:"http"`
	GRPC    OTLPGRPCOptions `yaml:"grpc" json:"grpc"`
}

type OTLPHTTPOptions struct {
	Endpoint string `yaml:"endpoint" json:"endpoint"`
	Insecure bool   `yaml:"insecure" json:"insecure"`
}

type OTLPGRPCOptions struct {
	Endpoint string `yaml:"endpoint" json:"endpoint"`
	Insecure bool   `yaml:"insecure" json:"insecure"`
}

type ServerOptions struct {
	ID                 string               `yaml:"-" json:"-"`
	Bind               string               `yaml:"bind" json:"bind"`
	TLS                TLSOptions           `yaml:"tls" json:"tls"`
	ReusePort          bool                 `yaml:"reuse_port" json:"reuse_port"`
	TCPQuickAck        bool                 `yaml:"tcp_quick_ack" json:"tcp_quick_ack"`
	TCPFastOpen        bool                 `yaml:"tcp_fast_open" json:"tcp_fast_open"`
	Backlog            int                  `yaml:"backlog" json:"backlog"`
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
	Graceful  time.Duration `yaml:"graceful" json:"graceful"`
	Idle      time.Duration `yaml:"idle" json:"idle_timeout"`
	KeepAlive time.Duration `yaml:"keepalive" json:"keepalive"`
	Read      time.Duration `yaml:"read" json:"read"`
	Write     time.Duration `yaml:"write" json:"write"`
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
	Target      string        `yaml:"target" json:"target"`
	MaxFails    uint          `yaml:"max_fails" json:"max_fails"`
	FailTimeout time.Duration `yaml:"fail_timeout" json:"fail_timeout"`
	Weight      uint32          `yaml:"weight" json:"weight"`
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
	ProtocolHTTP  Protocol = "http"
	ProtocolHTTP2 Protocol = "http2"
	ProtocolGRPC  Protocol = "grpc"
)

type ServiceOptions struct {
	ID              string                `yaml:"-" json:"-"`
	TLSVerify       bool                  `yaml:"tls_verify" json:"tls_verify"`
	MaxConnsPerHost *int                  `yaml:"max_conns_per_host" json:"max_conns_per_host"`
	Protocol        Protocol              `yaml:"protocol" json:"protocol"`
	Url             string                `yaml:"url" json:"url"`
	Timeout         ServiceTimeoutOptions `yaml:"timeout" json:"timeout"`
	Middlewares     []MiddlwareOptions    `yaml:"middlewares" json:"middlewares"`
}

type ServiceTimeoutOptions struct {
	Read        time.Duration `yaml:"read" json:"read"`
	Write       time.Duration `yaml:"write" json:"write"`
	Dail        time.Duration `yaml:"dail" json:"dail"`
	MaxConnWait time.Duration `yaml:"max_conn_wait" json:"max_conn_wait"`
	GRPC        time.Duration `yaml:"grpc" json:"grpc"`
}

type TLSOptions struct {
	MinVersion string `yaml:"min_version" json:"min_version"`
	CertPEM    string `yaml:"cert_pem" json:"cert_pem"`
	KeyPEM     string `yaml:"key_pem" json:"key_pem"`
}
