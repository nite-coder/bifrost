package variable

type RequestOriginal struct {
	ServerID string
	Scheme   []byte
	Host     []byte
	Path     []byte
	Protocol string
	Method   []byte
	Query    []byte
}

const (
	ServerID          = "$server_id"
	RouteID           = "$route_id"
	ServiceID         = "$service_id"
	RemoteAddr        = "$remote_addr"
	Host              = "$host"
	Time              = "$time"
	ReceivedSize      = "$received_size"
	SendSize          = "$send_size"
	Status            = "$status"
	RequestOrig       = "$request_orig"
	Request           = "$request"
	RequestScheme     = "$request_scheme"
	RequestProtocol   = "$request_protocol"
	RequestMethod     = "$request_method"
	RequestURI        = "$request_uri"
	RequestPath       = "$request_path"
	RequestQuery      = "$request_query"
	RequestPathAlias  = "$request_path_alias"
	RequestBody       = "$request_body"
	Duration          = "$duration"
	LogTime           = "$log_time"
	Upstream          = "$upstream"
	UpstreamID        = "$upstream_id"
	UpstreamURI       = "$upstream_uri"
	UpstreamMethod    = "$upstream_method"
	UpstreamProtocol  = "$upstream_protocol"
	UpstreamPath      = "$upstream_path"
	UpstreamPathAlias = "$upstream_path_alias"
	UpstreamAddr      = "$upstream_addr"
	UpstreamDuration  = "$upstream_duration"
	UpstreamStatus    = "$upstream_status"
	TraceID           = "$trace_id"
	Allow             = "$allow"
	ClientIP          = "$client_ip"
	UserAgent         = "$user_agent"
	TargetTimeout     = "target_timeout"

	// grpc
	GRPCStatus  = "$grpc_status"
	GRPCMessage = "$grpc_message"

	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)
