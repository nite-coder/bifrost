package config

import (
	"time"

	"gopkg.in/yaml.v3"
)

type Options struct {
	Watch           *bool                       `json:"watch"            yaml:"watch"`
	AccessLogs      map[string]AccessLogOptions `json:"access_logs"      yaml:"access_logs"`
	Servers         map[string]ServerOptions    `json:"servers"          yaml:"servers"`
	RoutesMap       *yaml.Node                  `json:"routes"           yaml:"routes"`
	Middlewares     map[string]MiddlwareOptions `json:"middlewares"      yaml:"middlewares"`
	Services        map[string]ServiceOptions   `json:"services"         yaml:"services"`
	Upstreams       map[string]UpstreamOptions  `json:"upstreams"        yaml:"upstreams"`
	Providers       ProviderOptions             `json:"providers"        yaml:"providers"`
	configPath      string                      `json:"-"                yaml:"-"`
	User            string                      `json:"user"             yaml:"user"`
	Group           string                      `json:"group"            yaml:"group"`
	Metrics         MetricsOptions              `json:"metrics"          yaml:"metrics"`
	Logging         LoggingOptions              `json:"logging"          yaml:"logging"`
	Routes          []*RouteOptions             `json:"-"                yaml:"-"`
	Redis           []RedisOptions              `json:"redis"            yaml:"redis"`
	Resolver        ResolverOptions             `json:"resolver"         yaml:"resolver"`
	Tracing         TracingOptions              `json:"tracing"          yaml:"tracing"`
	Default         DefaultOptions              `json:"default"          yaml:"default"`
	EventLoops      int                         `json:"event_loops"      yaml:"event_loops"`
	TimerResolution time.Duration               `json:"timer_resolution" yaml:"timer_resolution"`
	SkipResolver    bool                        `json:"-"                yaml:"-"`
	Gopool          bool                        `json:"gopool"           yaml:"gopool"`
}

// NewOptions creates a new Options instance with default values.
func NewOptions() Options {
	return Options{
		Gopool:      true,
		AccessLogs:  make(map[string]AccessLogOptions),
		Servers:     make(map[string]ServerOptions),
		Routes:      make([]*RouteOptions, 0),
		Middlewares: make(map[string]MiddlwareOptions),
		Services:    make(map[string]ServiceOptions),
		Upstreams:   make(map[string]UpstreamOptions),
	}
}

// UnmarshalYAML custom unmarshaler for Options.
func (opt *Options) UnmarshalYAML(value *yaml.Node) error {
	type options Options
	var defaults options
	err := value.Decode(&defaults)
	if err != nil {
		return err
	}
	*opt = Options(defaults)

	// due to map is unordered in Go, therefore, we need to parse `routes` section in yaml manually
	for idx, node := range value.Content {
		if node.Value == "routes" && len(value.Content) >= idx+1 {
			routesMapNode := value.Content[idx+1]
			for i, routeNode := range routesMapNode.Content {
				if routeNode.Tag != "!!str" {
					continue // Node is a map, so it is read out at key.
				}
				var routeOption RouteOptions
				routeOption.ID = routeNode.Value
				if len(routesMapNode.Content) >= i+1 {
					err := routesMapNode.Content[i+1].Decode(&routeOption)
					if err != nil {
						return err
					}
					opt.Routes = append(opt.Routes, &routeOption)
				}
			}
		}
	}
	return nil
}

func (opt *Options) IsWatch() bool {
	if opt.Watch == nil {
		return true
	}
	return *opt.Watch
}

func (opt *Options) ConfigPath() string {
	return opt.configPath
}

type ExperimentOptions struct {
	ChunkedTransfer bool `json:"chunked_transfer" yaml:"chunked_transfer"`
}
type ProviderOptions struct {
	K8S   K8SProviderOptions   `json:"k8s"   yaml:"k8s"`
	Nacos NacosProviderOptions `json:"nacos" yaml:"nacos"`
	File  FileProviderOptions  `json:"file"  yaml:"file"`
	DNS   DNSProviderOptions   `json:"dns"   yaml:"dns"`
}
type FileProviderOptions struct {
	Paths      []string `json:"paths"      yaml:"paths"`
	Extensions []string `json:"extensions" yaml:"extensions"`
	Enabled    bool     `json:"enabled"    yaml:"enabled"`
}
type File struct {
	DataID string `json:"data_id" yaml:"data_id"`
	Group  string `json:"group"   yaml:"group"`
}
type NacosConfigOptions struct {
	Watch       *bool         `json:"watch"        yaml:"watch"`
	Username    string        `json:"username"     yaml:"username"`
	Password    string        `json:"password"     yaml:"password"`
	NamespaceID string        `json:"namespace_id" yaml:"namespace_id"`
	Prefix      string        `json:"prefix"       yaml:"prefix"`
	LogLevel    string        `json:"log_level"    yaml:"log_level"`
	LogDir      string        `json:"log_dir"      yaml:"log_dir"`
	CacheDir    string        `json:"cache_dir"    yaml:"cache_dir"`
	Endpoints   []string      `json:"endpoints"    yaml:"endpoints"`
	Files       []*File       `json:"files"        yaml:"files"`
	Timeout     time.Duration `json:"timeout"      yaml:"timeout"`
	Enabled     bool          `json:"enabled"      yaml:"enabled"`
}
type NacosDiscoveryOptions struct {
	Username    string        `json:"username"     yaml:"username"`
	Password    string        `json:"password"     yaml:"password"`
	NamespaceID string        `json:"namespace_id" yaml:"namespace_id"`
	Prefix      string        `json:"prefix"       yaml:"prefix"`
	LogDir      string        `json:"log_dir"      yaml:"log_dir"`
	LogLevel    string        `json:"log_level"    yaml:"log_level"`
	CacheDir    string        `json:"cache_dir"    yaml:"cache_dir"`
	Endpoints   []string      `json:"endpoints"    yaml:"endpoints"`
	Timeout     time.Duration `json:"timeout"      yaml:"timeout"`
	Enabled     bool          `json:"enabled"      yaml:"enabled"`
}
type NacosProviderOptions struct {
	Config    NacosConfigOptions    `json:"config"    yaml:"config"`
	Discovery NacosDiscoveryOptions `json:"discovery" yaml:"discovery"`
}
type DNSProviderOptions struct {
	Servers []string      `json:"servers" yaml:"servers"`
	Valid   time.Duration `json:"valid"   yaml:"valid"`
	Enabled bool          `json:"enabled" yaml:"enabled"`
}
type K8SProviderOptions struct {
	APIServer string `json:"api_server" yaml:"api_server"`
	Enabled   bool   `json:"enabled"    yaml:"enabled"`
}
type MetricsOptions struct {
	Prometheus PrometheusOptions  `json:"prometheus" yaml:"prometheus"`
	OTLP       OTLPMetricsOptions `json:"otlp"       yaml:"otlp"`
}

type OTLPMetricsOptions struct {
	ServiceName string        `json:"service_name" yaml:"service_name"`
	Endpoint    string        `json:"endpoint"     yaml:"endpoint"`
	Flush       time.Duration `json:"flush"        yaml:"flush"`
	Timeout     time.Duration `json:"timeout"      yaml:"timeout"`
	Insecure    bool          `json:"insecure"     yaml:"insecure"`
	Enabled     bool          `json:"enabled"      yaml:"enabled"`
}
type LoggingOptions struct {
	Level                    string `json:"level"                      yaml:"level"`
	Handler                  string `json:"handler"                    yaml:"handler"`
	Output                   string `json:"output"                     yaml:"output"`
	DisableRedirectStdStream bool   `json:"disable_redirect_stdstream" yaml:"disable_redirect_stdstream"`
}
type PrometheusOptions struct {
	ServerID string    `json:"server_id" yaml:"server_id"`
	Path     string    `json:"path"      yaml:"path"`
	Buckets  []float64 `json:"buckets"   yaml:"buckets"`
	Enabled  bool      `json:"enabled"   yaml:"enabled"`
}
type ServerTracingOptions struct {
	Enabled    *bool             `json:"enabled"    yaml:"enabled"`
	Attributes map[string]string `json:"attributes" yaml:"attributes"`
}

func (options ServerTracingOptions) IsEnabled() bool {
	if options.Enabled == nil || *options.Enabled {
		return true
	}
	return false
}

type Observability struct {
	Tracing ServerTracingOptions `json:"tracing" yaml:"tracing"`
}
type TracingOptions struct {
	ServiceName  string        `json:"service_name"  yaml:"service_name"`
	Endpoint     string        `json:"endpoint"      yaml:"endpoint"`
	Propagators  []string      `json:"propagators"   yaml:"propagators"`
	SamplingRate float64       `json:"sampling_rate" yaml:"sampling_rate"`
	BatchSize    int64         `json:"batch_size"    yaml:"batch_size"`
	QueueSize    int64         `json:"queue_size"    yaml:"queue_size"`
	Flush        time.Duration `json:"flush"         yaml:"flush"`
	Timeout      time.Duration `json:"timeout"       yaml:"timeout"`
	Enabled      bool          `json:"enabled"       yaml:"enabled"`
	Insecure     bool          `json:"insecure"      yaml:"insecure"`
}
type ServerOptions struct {
	Observability      Observability        `json:"observability"         yaml:"observability"`
	TLS                TLSOptions           `json:"tls"                   yaml:"tls"`
	ID                 string               `json:"-"                     yaml:"-"`
	Bind               string               `json:"bind"                  yaml:"bind"`
	AccessLogID        string               `json:"access_log_id"         yaml:"access_log_id"`
	Logging            LoggingOptions       `json:"logging"               yaml:"logging"`
	Middlewares        []MiddlwareOptions   `json:"middlewares"           yaml:"middlewares"`
	TrustedCIDRS       []string             `json:"trusted_cidrs"         yaml:"trusted_cidrs"`
	RemoteIPHeaders    []string             `json:"remote_ip_headers"     yaml:"remote_ip_headers"`
	Timeout            ServerTimeoutOptions `json:"timeout"               yaml:"timeout"`
	Backlog            int                  `json:"backlog"               yaml:"backlog"`
	MaxRequestBodySize int                  `json:"max_request_body_size" yaml:"max_request_body_size"`
	ReadBufferSize     int                  `json:"read_buffer_size"      yaml:"read_buffer_size"`
	ReusePort          bool                 `json:"reuse_port"            yaml:"reuse_port"`
	TCPQuickAck        bool                 `json:"tcp_quick_ack"         yaml:"tcp_quick_ack"`
	TCPFastOpen        bool                 `json:"tcp_fast_open"         yaml:"tcp_fast_open"`
	HTTP2              bool                 `json:"http2"                 yaml:"http2"`
	PPROF              bool                 `json:"pprof"                 yaml:"pprof"`
	ProxyProtocol      bool                 `json:"proxy_protocol"        yaml:"proxy_protocol"`
}
type ServerTimeoutOptions struct {
	Graceful  time.Duration `json:"graceful"     yaml:"graceful"`
	Idle      time.Duration `json:"idle_timeout" yaml:"idle"`
	KeepAlive time.Duration `json:"keepalive"    yaml:"keepalive"`
	Read      time.Duration `json:"read"         yaml:"read"`
	Write     time.Duration `json:"write"        yaml:"write"`
}
type EscapeType string

const (
	NoneEscape    EscapeType = "none"
	DefaultEscape EscapeType = "default"
	JSONEscape    EscapeType = "json"
)

type AccessLogOptions struct {
	Output     string        `json:"output"      yaml:"output"`
	Template   string        `json:"template"    yaml:"template"`
	TimeFormat string        `json:"time_format" yaml:"time_format"`
	Escape     EscapeType    `json:"escape"      yaml:"escape"`
	BufferSize int           `json:"buffer_size" yaml:"buffer_size"`
	Flush      time.Duration `json:"flush"       yaml:"flush"`
}
type MiddlwareOptions struct {
	ID     string `json:"-"      yaml:"-"`
	Type   string `json:"type"   yaml:"type"`
	Params any    `json:"params" yaml:"params"`
	Use    string `json:"use"    yaml:"use"`
}
type PassiveHealthOptions struct {
	MaxFails    *uint         `json:"max_fails"    yaml:"max_fails"`
	FailTimeout time.Duration `json:"fail_timeout" yaml:"fail_timeout"`
}
type ActiveHealthOptions struct {
	Path             string        `json:"path"              yaml:"path"`
	Method           string        `json:"method"            yaml:"method"`
	Interval         time.Duration `json:"interval"          yaml:"interval"`
	Port             int           `json:"port"              yaml:"port"`
	SuccessThreshold int           `json:"success_threshold" yaml:"success_threshold"`
	FailureThreshold int           `json:"failure_threshold" yaml:"failure_threshold"`
}
type HealthCheckOptions struct {
	Passive PassiveHealthOptions `json:"passive" yaml:"passive"`
	Active  ActiveHealthOptions  `json:"active"  yaml:"active"`
}
type TargetOptions struct {
	Target string            `json:"target" yaml:"target"`
	Weight uint32            `json:"weight" yaml:"weight"`
	Tags   map[string]string `json:"tags"   yaml:"tags"`
}
type DiscoveryOptions struct {
	Type      string `json:"type"      yaml:"type"`
	Namespace string `json:"namespace" yaml:"namespace"`
	Name      string `json:"name"      yaml:"name"`
}

type BalancerOptions struct {
	Type   string `json:"type"   yaml:"type"`
	Params any    `json:"params" yaml:"params"`
}

type UpstreamOptions struct {
	ID          string             `json:"-"            yaml:"-"`
	Balancer    BalancerOptions    `json:"balancer"     yaml:"balancer"`
	HashOn      string             `json:"hash_on"      yaml:"hash_on"`
	Discovery   DiscoveryOptions   `json:"discovery"    yaml:"discovery"`
	Targets     []TargetOptions    `json:"targets"      yaml:"targets"`
	HealthCheck HealthCheckOptions `json:"health_check" yaml:"health_check"`
}
type RouteOptions struct {
	ID          string             `json:"-"           yaml:"-"`
	Route       string             `json:"route"       yaml:"route"`
	ServiceID   string             `json:"service_id"  yaml:"service_id"`
	Methods     []string           `json:"methods"     yaml:"methods"`
	Paths       []string           `json:"paths"       yaml:"paths"`
	Servers     []string           `json:"servers"     yaml:"servers"`
	Tags        []string           `json:"tags"        yaml:"tags"`
	Middlewares []MiddlwareOptions `json:"middlewares" yaml:"middlewares"`
}
type Protocol string

const (
	ProtocolHTTP  Protocol = "http"
	ProtocolHTTP2 Protocol = "http2"
	ProtocolGRPC  Protocol = "grpc"
)

type ServiceOptions struct {
	MaxConnsPerHost *int                  `json:"max_conns_per_host" yaml:"max_conns_per_host"`
	ID              string                `json:"-"                  yaml:"-"`
	Protocol        Protocol              `json:"protocol"           yaml:"protocol"`
	URL             string                `json:"url"                yaml:"url"`
	Middlewares     []MiddlwareOptions    `json:"middlewares"        yaml:"middlewares"`
	Timeout         ServiceTimeoutOptions `json:"timeout"            yaml:"timeout"`
	TLSVerify       bool                  `json:"tls_verify"         yaml:"tls_verify"`
	PassHostHeader  *bool                 `json:"pass_host_header"   yaml:"pass_host_header"`
}

func (options ServiceOptions) IsPassHostHeader() bool {
	if options.PassHostHeader == nil || *options.PassHostHeader {
		return true
	}
	return false
}

type ServiceTimeoutOptions struct {
	Read        time.Duration `json:"read"          yaml:"read"`
	Write       time.Duration `json:"write"         yaml:"write"`
	Dail        time.Duration `json:"dail"          yaml:"dail"`
	MaxConnWait time.Duration `json:"max_conn_wait" yaml:"max_conn_wait"`
	GRPC        time.Duration `json:"grpc"          yaml:"grpc"`
}
type TLSOptions struct {
	MinVersion string `json:"min_version" yaml:"min_version"`
	CertPEM    string `json:"cert_pem"    yaml:"cert_pem"`
	KeyPEM     string `json:"key_pem"     yaml:"key_pem"`
}
type RedisOptions struct {
	ID       string   `json:"id"        yaml:"id"`
	Username string   `json:"username"  yaml:"username"`
	Password string   `json:"password"  yaml:"password"`
	Addrs    []string `json:"addrs"     yaml:"addrs"`
	DB       int      `json:"db"        yaml:"db"`
	SkipPing bool     `json:"skip_ping" yaml:"skip_ping"`
}
type ResolverOptions struct {
	Servers   []string      `json:"servers"    yaml:"servers"`
	Hostsfile string        `json:"hosts_file" yaml:"hosts_file"`
	Order     []string      `json:"order"      yaml:"order"`
	Timeout   time.Duration `json:"timeout"    yaml:"timeout"`
}
type DefaultServiceOptions struct {
	MaxConnsPerHost *int                  `json:"max_conns_per_host" yaml:"max_conns_per_host"`
	Protocol        Protocol              `json:"protocol"           yaml:"protocol"`
	Timeout         ServiceTimeoutOptions `json:"timeout"            yaml:"timeout"`
}
type DefaultUpstreamOptions struct {
	MaxFails    uint          `json:"max_fails"    yaml:"max_fails"`
	FailTimeout time.Duration `json:"fail_timeout" yaml:"fail_timeout"`
}
type DefaultOptions struct {
	Service  DefaultServiceOptions  `json:"service"  yaml:"service"`
	Upstream DefaultUpstreamOptions `json:"upstream" yaml:"upstream"`
}
