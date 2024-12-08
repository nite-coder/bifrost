package variable

const (
	SERVER_ID           = "$server_id"
	REMOTE_ADDR         = "$remote_addr"
	HOST                = "$host"
	TIME                = "$time"
	RECEIVED_SIZE       = "$received_size"
	SEND_SIZE           = "$send_size"
	STATUS              = "$status"
	REQUEST_PROTOCOL    = "$request_protocol"
	REQUEST_METHOD      = "$request_method"
	REQUEST_URI         = "$request_uri"
	REQUEST_PATH        = "$request_path"
	REQUEST_PATH_ALIAS  = "$request_path_alias"
	REQUEST_BODY        = "$request_body"
	DURATION            = "$duration"
	LOG_TIME            = "$log_time"
	UPSTREAM            = "$upstream"
	UPSTREAM_URI        = "$upstream_uri"
	UPSTREAM_METHOD     = "$upstream_method"
	UPSTREAM_PROTOCOL   = "$upstream_protocol"
	UPSTREAM_PATH       = "$upstream_path"
	UPSTREAM_PATH_ALIAS = "$upstream_path_alias"
	UPSTREAM_ADDR       = "$upstream_addr"
	UPSTREAM_DURATION   = "$upstream_duration"
	UPSTREAM_STATUS     = "$upstream_status"
	TRACE_ID            = "$trace_id"
	ALLOW               = "$allow"
	ClientIP            = "$client_ip"
	UserAgent           = "$user_agent"
	TARGET_TIMEOUT      = "target_timeout"

	// grpc
	GRPC_STATUS  = "$grpc_status"
	GRPC_MESSAGE = "$grpc_message"

	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)
