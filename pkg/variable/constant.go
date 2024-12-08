package variable

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
	RequestProtocol   = "$request_protocol"
	RequestMethod     = "$request_method"
	RequestURI        = "$request_uri"
	RequestPath       = "$request_path"
	RequestPathAlias  = "$request_path_alias"
	RequestBody       = "$request_body"
	Duration          = "$duration"
	LogTime           = "$log_time"
	Upstream          = "$upstream"
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
