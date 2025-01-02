package variable

type RequestOriginal struct {
	ServerID string
	Scheme   []byte
	Host     []byte
	Method   []byte
	Path     []byte
	Query    []byte
	Protocol string
}

const (
	Time                        = "$time"
	ServerID                    = "$server_id"
	RouteID                     = "$route_id"
	ServiceID                   = "$service_id"
	UpstreamID                  = "$upstream_id"
	RequestOrig                 = "$request_orig"
	NetworkPeerAddress          = "$network.peer.address"
	HTTPRequest                 = "$http.request"
	HTTPRequestSize             = "$http.request.size"
	HTTPRequestScheme           = "$http.request.scheme"
	HTTPRequestHost             = "$http.request.host"
	HTTPRequestMethod           = "$http.request.method"
	HTTPRequestPath             = "$http.request.path"
	HTTPRequestQuery            = "$http.request.query"
	HTTPRequestProtocol         = "$http.request.protocol"
	HTTPRequestURI              = "$http.request.uri"
	HTTPRequestPathAlias        = "$http.request.path_alias"
	HTTPRequestBody             = "$http.request.body"
	HTTPResponseSize            = "$http.response.size"
	HTTPResponseStatusCode      = "$http.response.status_code"
	Duration                    = "$duration"
	LogTime                     = "$log_time"
	UpstreamRequest             = "$upstream.request"
	UpstreamRequestHost         = "$upstream.request.host"
	UpstreamRequestMethod       = "$upstream.request.method"
	UpstreamRequestPath         = "$upstream.request.path"
	UpstreamRequestPathAlias    = "$upstream.request.path_alias"
	UpstreamRequestQuery        = "$upstream.request.query"
	UpstreamRequestURI          = "$upstream.request.uri"
	UpstreamRequestProtocol     = "$upstream.request.protocol"
	UpstreamDuration            = "$upstream.duration"
	UpstreamResponoseStatusCode = "$upstream.response.status_code"
	Allow                       = "$allow"
	ClientIP                    = "$client_ip"
	TargetTimeout               = "target_timeout"

	// grpc
	GRPCStatusCode = "$grpc.status_code"
	GRPCMessage    = "$grpc.message"

	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)
