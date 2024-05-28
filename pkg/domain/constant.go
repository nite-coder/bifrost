package domain

const (
	REMOTE_ADDR        = "$remote_addr"
	TIME               = "$time"
	BYTE_SENT          = "$bytes_sent" // the number of bytes sent to a client
	STATUS             = "$status"
	REQUEST            = "$request"
	REQUEST_PROTOCOL   = "$request_protocol"
	REQUEST_METHOD     = "$request_method"
	REQUEST_URI        = "$request_uri"
	REQUEST_PATH       = "$request_path"
	REQUEST_BODY       = "$request_body"
	REQUEST_LENGTH     = "$request_length" // request length (including request line, header, and request body)
	DURATION           = "$duration"
	LOG_TIME           = "$log_time"
	UPSTREAM_URI       = "$upstream_uri"
	UPSTREAM_METHOD    = "$upstream_method"
	UPSTREAM_PROTOCOL  = "$upstream_protocol"
	UPSTREAM_PATH      = "$upstream_path"
	UPSTREAM_ADDR      = "$upstream_addr"
	UPSTREAM_DURATION  = "$upstream_duration"
	UPSTREAM_STATUS    = "$upstream_status"
	CLIENT_CANCELED_AT = "$client_canceled_at"

	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)
