package config

import (
	"time"

	"gopkg.in/yaml.v3"
)

type Options struct {
	configPath      string                      `yaml:"-" json:"-"`
	IsDaemon        bool                        `yaml:"-" json:"-"`
	SkipResolver    bool                        `yaml:"-" json:"-"`
	PIDFile         string                      `yaml:"pid_file" json:"pid_file"`
	UpgradeSock     string                      `yaml:"upgrade_sock" json:"upgrade_sock"`
	Gopool          bool                        `yaml:"gopool" json:"gopool"`
	Resolver        ResolverOptions             `yaml:"resolver" json:"resolver"`
	NumLoops        int                         `yaml:"num_loops" json:"num_loops"`
	Watch           *bool                       `yaml:"watch" json:"watch"`
	User            string                      `yaml:"user" json:"user"`
	Group           string                      `yaml:"group" json:"group"`
	Providers       ProviderOtions              `yaml:"providers" json:"providers"`
	TimerResolution time.Duration               `yaml:"timer_resolution" json:"timer_resolution"`
	Logging         LoggingOtions               `yaml:"logging" json:"logging"`
	Metrics         MetricsOptions              `yaml:"metrics" json:"metrics"`
	Tracing         TracingOptions              `yaml:"tracing" json:"tracing"`
	Default         DefaultOptions              `yaml:"default" json:"default"`
	AccessLogs      map[string]AccessLogOptions `yaml:"access_logs" json:"access_logs"`
	Servers         map[string]ServerOptions    `yaml:"servers" json:"servers"`
	RoutesMap       *yaml.Node                  `yaml:"routes"`
	Routes          []*RouteOptions             `yaml:"-"`
	Middlewares     map[string]MiddlwareOptions `yaml:"middlewares" json:"middlewares"`
	Services        map[string]ServiceOptions   `yaml:"services" json:"services"`
	Upstreams       map[string]UpstreamOptions  `yaml:"upstreams" json:"upstreams"`
	Redis           []RedisOptions              `yaml:"redis" json:"redis"`
	Experiment      ExperimentOptions           `yaml:"experiment" json:"experiment"`
}

func (o *Options) UnmarshalYAML(value *yaml.Node) error {
	type options Options
	if err := value.Decode((*options)(o)); err != nil {
		return err
	}

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

					o.Routes = append(o.Routes, &routeOption)
				}
			}
		}
	}

	return nil
}

func NewOptions() Options {
	mainOptions := Options{
		AccessLogs:  make(map[string]AccessLogOptions),
		Servers:     make(map[string]ServerOptions),
		Routes:      make([]*RouteOptions, 0),
		Middlewares: make(map[string]MiddlwareOptions),
		Services:    make(map[string]ServiceOptions),
		Upstreams:   make(map[string]UpstreamOptions),
	}

	return mainOptions
}

func (opt Options) IsWatch() bool {
	if opt.Watch == nil {
		return true
	}

	return *opt.Watch
}

func (opt Options) ConfigPath() string {
	return opt.configPath
}

type ExperimentOptions struct {
	ChunkedTransfer bool `yaml:"chunked_transfer" json:"chunked_transfer"`
}

type ProviderOtions struct {
	File  FileProviderOptions  `yaml:"file" json:"file"`
	Nacos NacosProviderOptions `yaml:"nacos" json:"nacos"`
	DNS   DNSProviderOptions   `yaml:"dns" json:"dns"`
	K8S   K8SProviderOptions   `yaml:"k8s" json:"k8s"`
}

type FileProviderOptions struct {
	Enabled    bool     `yaml:"enabled" json:"enabled"`
	Paths      []string `yaml:"paths" json:"paths"`
	Extensions []string `yaml:"extensions" json:"extensions"`
}

type File struct {
	DataID string `yaml:"data_id" json:"data_id"`
	Group  string `yaml:"group" json:"group"`
}

type NacosConfigOptions struct {
	Enabled     bool          `yaml:"enabled" json:"enabled"`
	Username    string        `yaml:"username" json:"username"`
	Password    string        `yaml:"password" json:"password"`
	NamespaceID string        `yaml:"namespace_id" json:"namespace_id"`
	Prefix      string        `yaml:"prefix" json:"prefix"`
	LogLevel    string        `yaml:"log_level" json:"log_level"`
	LogDir      string        `yaml:"log_dir" json:"log_dir"`
	CacheDir    string        `yaml:"cache_dir" json:"cache_dir"`
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`
	Watch       *bool         `yaml:"watch" json:"watch"`
	Endpoints   []string      `yaml:"endpoints" json:"endpoints"`
	Files       []*File       `yaml:"files" json:"files"`
}

type NacosDiscoveryOptions struct {
	Enabled     bool          `yaml:"enabled" json:"enabled"`
	Username    string        `yaml:"username" json:"username"`
	Password    string        `yaml:"password" json:"password"`
	NamespaceID string        `yaml:"namespace_id" json:"namespace_id"`
	Prefix      string        `yaml:"prefix" json:"prefix"`
	LogDir      string        `yaml:"log_dir" json:"log_dir"`
	LogLevel    string        `yaml:"log_level" json:"log_level"`
	CacheDir    string        `yaml:"cache_dir" json:"cache_dir"`
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`
	Endpoints   []string      `yaml:"endpoints" json:"endpoints"`
}

type NacosProviderOptions struct {
	Config    NacosConfigOptions    `yaml:"config" json:"config"`
	Discovery NacosDiscoveryOptions `yaml:"discovery" json:"discovery"`
}

type DNSProviderOptions struct {
	Enabled bool          `yaml:"enabled" json:"enabled"`
	Servers []string      `yaml:"servers" json:"servers"`
	Valid   time.Duration `yaml:"valid" json:"valid"`
}

type K8SProviderOptions struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	APIServer string `yaml:"api_server" json:"api_server"`
}

type MetricsOptions struct {
	Prometheus PrometheusOptions `yaml:"prometheus" json:"prometheus"`
}

type LoggingOtions struct {
	Level                 string `yaml:"level" json:"level"`
	Handler               string `yaml:"handler" json:"handler"`
	Output                string `yaml:"output" json:"output"`
	DisableRedirectStdErr bool   `yaml:"disable_redirect_stderr" json:"disable_redirect_stderr"`
}

type PrometheusOptions struct {
	Enabled  bool      `yaml:"enabled" json:"enabled"`
	ServerID string    `yaml:"server_id" json:"server_id"`
	Path     string    `yaml:"path" json:"path"`
	Buckets  []float64 `yaml:"buckets" json:"buckets"`
}

type ServerTracingOptions struct {
	Enabled    *bool             `yaml:"enabled" json:"enabled"`
	Attributes map[string]string `yaml:"attributes" json:"attributes"`
}

func (options ServerTracingOptions) IsEnabled() bool {
	if options.Enabled == nil || *options.Enabled {
		return true
	}

	return false
}

type Observability struct {
	Tracing ServerTracingOptions `yaml:"tracing" json:"tracing"`
}

type TracingOptions struct {
	Enabled      bool          `yaml:"enabled" json:"enabled"`
	ServiceName  string        `yaml:"service_name" json:"service_name"`
	Propagators  []string      `yaml:"propagators" json:"propagators"`
	Endpoint     string        `yaml:"endpoint" json:"endpoint"`
	Insecure     bool          `yaml:"insecure" json:"insecure"`
	SamplingRate float64       `yaml:"sampling_rate" json:"sampling_rate"`
	BatchSize    int64         `yaml:"batch_size" json:"batch_size"`
	QueueSize    int64         `yaml:"queue_size" json:"queue_size"`
	Flush        time.Duration `yaml:"flush" json:"flush"`
	Timeout      time.Duration `yaml:"timeout" json:"timeout"`
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
	Observability      Observability        `yaml:"observability" json:"observability"`
	TrustedCIDRS       []string             `yaml:"trusted_cidrs" json:"trusted_cidrs"`
	RemoteIPHeaders    []string             `yaml:"remote_ip_headers" json:"remote_ip_headers"`
	ProxyProtocol      bool                 `yaml:"proxy_protocol" json:"proxy_protocol"`
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
	BufferSize int           `yaml:"buffer_size" json:"buffer_size"`
	Output     string        `yaml:"output" json:"output"`
	Template   string        `yaml:"template" json:"template"`
	TimeFormat string        `yaml:"time_format" json:"time_format"`
	Escape     EscapeType    `yaml:"escape" json:"escape"`
	Flush      time.Duration `yaml:"flush" json:"flush"`
}

type MiddlwareOptions struct {
	ID     string `yaml:"-" json:"-"`
	Type   string `yaml:"type" json:"type"`
	Params any    `yaml:"params" json:"params"`
	Use    string `yaml:"use" json:"use"`
}

type UpstreamStrategy string

const (
	RandomStrategy     UpstreamStrategy = "random"
	RoundRobinStrategy UpstreamStrategy = "round_robin"
	WeightedStrategy   UpstreamStrategy = "weighted"
	HashingStrategy    UpstreamStrategy = "hashing"
)

type PassiveHealthOptions struct {
	FailTimeout time.Duration `yaml:"fail_timeout" json:"fail_timeout"`
	MaxFails    *uint         `yaml:"max_fails" json:"max_fails"`
}

type ActiveHealthOptions struct {
	Interval         time.Duration `yaml:"interval" json:"interval"`
	Path             string        `yaml:"path" json:"path"`
	Method           string        `yaml:"method" json:"method"`
	Port             int           `yaml:"port" json:"port"`
	SuccessThreshold int           `yaml:"success_threshold" json:"success_threshold"`
	FailureThreshold int           `yaml:"failure_threshold" json:"failure_threshold"`
}

type HealthCheckOptions struct {
	Passive PassiveHealthOptions `yaml:"passive" json:"passive"`
	Active  ActiveHealthOptions  `yaml:"active" json:"active"`
}

type TargetOptions struct {
	Target string `yaml:"target" json:"target"`
	Weight uint32 `yaml:"weight" json:"weight"`
}

type DiscoveryOptions struct {
	Type      string `yaml:"type" json:"type"`
	Namespace string `yaml:"namespace" json:"namespace"`
	Name      string `yaml:"name" json:"name"`
}

type UpstreamOptions struct {
	ID          string             `yaml:"-" json:"-"`
	Strategy    UpstreamStrategy   `yaml:"strategy" json:"strategy"`
	HashOn      string             `yaml:"hash_on" json:"hash_on"`
	Discovery   DiscoveryOptions   `yaml:"discovery" json:"discovery"`
	Targets     []TargetOptions    `yaml:"targets" json:"targets"`
	HealthCheck HealthCheckOptions `yaml:"health_check" json:"health_check"`
}

type RouteOptions struct {
	ID          string             `yaml:"-" json:"-"`
	Methods     []string           `yaml:"methods" json:"methods"`
	Paths       []string           `yaml:"paths" json:"paths"`
	Route       string             `yaml:"route" json:"route"`
	Servers     []string           `yaml:"servers" json:"servers"`
	Tags        []string           `yaml:"tags" json:"tags"`
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

type RedisOptions struct {
	ID       string   `yaml:"id" json:"id"`
	Addrs    []string `yaml:"addrs" json:"addrs"`
	Username string   `yaml:"username" json:"username"`
	Password string   `yaml:"password" json:"password"`
	DB       int      `yaml:"db" json:"db"`
	SkipPing bool     `yaml:"skip_ping" json:"skip_ping"`
}

type ResolverOptions struct {
	Servers   []string      `yaml:"servers" json:"servers"`
	Hostsfile string        `yaml:"hosts_file" json:"hosts_file"`
	Order     []string      `yaml:"order" json:"order"`
	Timeout   time.Duration `yaml:"timeout" json:"timeout"`
}

type DefaultServiceOptions struct {
	MaxConnsPerHost *int                  `yaml:"max_conns_per_host" json:"max_conns_per_host"`
	Protocol        Protocol              `yaml:"protocol" json:"protocol"`
	Timeout         ServiceTimeoutOptions `yaml:"timeout" json:"timeout"`
}

type DefaultUpstreamOptions struct {
	MaxFails    uint          `yaml:"max_fails" json:"max_fails"`
	FailTimeout time.Duration `yaml:"fail_timeout" json:"fail_timeout"`
}

type DefaultOptions struct {
	Service  DefaultServiceOptions  `yaml:"service" json:"service"`
	Upstream DefaultUpstreamOptions `yaml:"upstream" json:"upstream"`
}
