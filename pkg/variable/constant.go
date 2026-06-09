package variable

// RequestOriginal stores the original request information.
type RequestOriginal struct {
	ServerID string
	Method   string
	Protocol string
	Scheme   []byte
	Host     []byte
	Path     []byte
	Query    []byte
}

// RequestRoute stores information about the routed request.
type RequestRoute struct {
	RouteID   string
	Route     string
	ServiceID string
	Tags      []string
}

// Variable names used in directives.
const (
	// Time is the current time.
	Time = "$time"
	// ServerID is the unique identifier of the gateway server.
	ServerID = "$server_id"
	// RouteID is the unique identifier of the matching route.
	RouteID = "$route_id"
	// ServiceID is the unique identifier of the target service.
	ServiceID = "$service_id"
	// UpstreamID is the unique identifier of the selected upstream.
	UpstreamID = "$upstream_id"
	// RequestOrig is the key storing original request details in the context.
	RequestOrig = "$request_orig"
	// NetworkPeerAddress is the peer IP address of the network connection.
	NetworkPeerAddress = "$network.peer.address"
	// Hostname is the hostname of the machine running the gateway.
	Hostname = "$hostname"
	// TraceID is the trace identifier for request tracking.
	TraceID = "$trace_id"
	// HTTPStart is the start time of the HTTP request.
	HTTPStart = "$http.start"
	// HTTPFinish is the finish time of the HTTP request.
	HTTPFinish = "$http.finish"
	// HTTPRoute is the matched HTTP route template.
	HTTPRoute = "$http.route"
	// HTTPRequest is the full formatted HTTP request line.
	HTTPRequest = "$http.request"
	// HTTPRequestSize is the received size of the HTTP request in bytes.
	HTTPRequestSize = "$http.request.size"
	// HTTPRequestScheme is the scheme (http/https) of the HTTP request.
	HTTPRequestScheme = "$http.request.scheme"
	// HTTPRequestHost is the host header value of the HTTP request.
	HTTPRequestHost = "$http.request.host"
	// HTTPRequestMethod is the HTTP method of the request.
	HTTPRequestMethod = "$http.request.method"
	// HTTPRequestPath is the path component of the request URL.
	HTTPRequestPath = "$http.request.path"
	// HTTPRequestQuery is the query string of the request URL.
	HTTPRequestQuery = "$http.request.query"
	// HTTPRequestProtocol is the protocol version of the HTTP request.
	HTTPRequestProtocol = "$http.request.protocol"
	// HTTPRequestURI is the request URI (path + query).
	HTTPRequestURI = "$http.request.uri"
	// HTTPRequestTags represents the tags associated with the matched route.
	HTTPRequestTags = "$http.request.tags"
	// HTTPRequestBody is the body content of the HTTP request.
	HTTPRequestBody = "$http.request.body"
	// HTTPResponseSize is the sent size of the HTTP response in bytes.
	HTTPResponseSize = "$http.response.size"
	// HTTPResponseStatusCode is the status code of the HTTP response.
	HTTPResponseStatusCode = "$http.response.status_code"
	// HTTPRequestDuration is the duration taken to process the HTTP request.
	HTTPRequestDuration = "$http.request.duration"
	// ErrorType represents the type/category of error encountered.
	ErrorType = "$error.type"
	// ErrorMessage represents the detailed error message.
	ErrorMessage = "$error.message"
	// LogTime is the timestamp when the access log is written.
	LogTime = "$log_time"
	// UpstreamRequest is the full formatted request line sent upstream.
	UpstreamRequest = "$upstream.request"
	// UpstreamRequestHost is the target host address of the upstream.
	UpstreamRequestHost = "$upstream.request.host"
	// UpstreamRequestMethod is the HTTP method sent upstream.
	UpstreamRequestMethod = "$upstream.request.method"
	// UpstreamRequestPath is the request path sent upstream.
	UpstreamRequestPath = "$upstream.request.path"
	// UpstreamRequestQuery is the query string sent upstream.
	UpstreamRequestQuery = "$upstream.request.query"
	// UpstreamRequestURI is the request URI sent upstream.
	UpstreamRequestURI = "$upstream.request.uri"
	// UpstreamRequestProtocol is the protocol version used with the upstream.
	UpstreamRequestProtocol = "$upstream.request.protocol"
	// UpstreamDuration is the time taken to receive the upstream response.
	UpstreamDuration = "$upstream.duration"
	// UpstreamResponoseStatusCode is the HTTP response status code from the upstream.
	UpstreamResponoseStatusCode = "$upstream.response.status_code"
	// Allow is a flag indicating if the request is permitted.
	Allow = "$allow"
	// ClientIP is the client IP address.
	ClientIP = "$client_ip"
	// BifrostRoute is the context key for route info.
	BifrostRoute = "$bifrost.route"
	// TargetTimeout is the configured timeout duration for the target.
	TargetTimeout = "target_timeout"
	// GRPCStatusCode is the gRPC response status code.
	GRPCStatusCode = "$grpc.status_code"
	// GRPCMessage is the gRPC response status message.
	GRPCMessage = "$grpc.message"
	// Model is the virtual model name.
	Model = "$model"
	// ModelID is the concrete model ID/identifier.
	ModelID = "$model_id"
	// InputTokens is the number of input tokens consumed by the AI request.
	InputTokens = "$input_tokens"
	// OutputTokens is the number of output tokens consumed by the AI request.
	OutputTokens = "$output_tokens"
	// InputCachedTokens is the number of cached input tokens in the AI request.
	InputCachedTokens = "$input_cached_tokens"
	// TotalTokens is the total number of tokens (input + output) consumed by the AI request.
	TotalTokens = "$total_tokens"
	// InputCost is the calculated cost of the input tokens.
	InputCost = "$input_cost"
	// OutputCost is the calculated cost of the output tokens.
	OutputCost = "$output_cost"
	// TotalCost is the total calculated cost of the AI request.
	TotalCost = "$total_cost"
	// B represents a byte unit (1).
	B = 1
	// KB represents a kilobyte unit (1024 bytes).
	KB = 1024 * B
	// MB represents a megabyte unit.
	MB = 1024 * KB
	// GB represents a gigabyte unit.
	GB = 1024 * MB
)
