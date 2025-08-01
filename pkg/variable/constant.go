package variable

type RequestOriginal struct {
	ServerID string
	Method   string
	Protocol string
	Scheme   []byte
	Host     []byte
	Path     []byte
	Query    []byte
}
type RequestRoute struct {
	RouteID   string
	Route     string
	ServiceID string
	Tags      []string
}

const (
	Time                        = "$time"
	ServerID                    = "$server_id"
	RouteID                     = "$route_id"
	ServiceID                   = "$service_id"
	UpstreamID                  = "$upstream_id"
	RequestOrig                 = "$request_orig"
	NetworkPeerAddress          = "$network.peer.address"
	Hostname                    = "$hostname"
	TraceID                     = "$trace_id"
	HTTPStart                   = "$http.start"
	HTTPFinish                  = "$http.finish"
	HTTPRoute                   = "$http.route"
	HTTPRequest                 = "$http.request"
	HTTPRequestSize             = "$http.request.size"
	HTTPRequestScheme           = "$http.request.scheme"
	HTTPRequestHost             = "$http.request.host"
	HTTPRequestMethod           = "$http.request.method"
	HTTPRequestPath             = "$http.request.path"
	HTTPRequestQuery            = "$http.request.query"
	HTTPRequestProtocol         = "$http.request.protocol"
	HTTPRequestURI              = "$http.request.uri"
	HTTPRequestTags             = "$http.request.tags"
	HTTPRequestBody             = "$http.request.body"
	HTTPResponseSize            = "$http.response.size"
	HTTPResponseStatusCode      = "$http.response.status_code"
	HTTPRequestDuration         = "$http.request.duration"
	ErrorType                   = "$error.type"
	ErrorMessage                = "$error.message"
	LogTime                     = "$log_time"
	UpstreamRequest             = "$upstream.request"
	UpstreamRequestHost         = "$upstream.request.host"
	UpstreamRequestMethod       = "$upstream.request.method"
	UpstreamRequestPath         = "$upstream.request.path"
	UpstreamRequestQuery        = "$upstream.request.query"
	UpstreamRequestURI          = "$upstream.request.uri"
	UpstreamRequestProtocol     = "$upstream.request.protocol"
	UpstreamDuration            = "$upstream.duration"
	UpstreamConnAcquisitionTime = "$upstream.conn_acquisition_time"
	UpstreamResponoseStatusCode = "$upstream.response.status_code"
	Allow                       = "$allow"
	ClientIP                    = "$client_ip"
	BifrostRoute                = "$bifrost.route"
	TargetTimeout               = "target_timeout"
	// grpc
	GRPCStatusCode = "$grpc.status_code"
	GRPCMessage    = "$grpc.message"
	B              = 1
	KB             = 1024 * B
	MB             = 1024 * KB
	GB             = 1024 * MB
)
