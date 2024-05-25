package gateway

import "regexp"

const (
	VAR_HOST               = "$host"
	REMOTE_ADDR            = "$remote_addr"
	TIME                   = "$time"
	BYTE_SENT              = "$bytes_sent" // the number of bytes sent to a client
	STATUS                 = "$status"
	REQUEST                = "$request"
	REQUEST_PROTOCOL       = "$request_protocol"
	REQUEST_METHOD         = "$request_method"
	REQUEST_PATH           = "$request_path"
	REQUEST_BODY           = "$request_body"
	REQUEST_LENGTH         = "$request_length" // request length (including request line, header, and request body)
	Duration               = "$duration"
	LOG_TIME               = "$log_time"
	UPSTREAM_ADDR          = "$upstream_addr"
	UPSTREAM_RESPONSE_TIME = "$upstream_response_time"
	UPSTREAM_STATUS        = "$upstream_status"

	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)

var (
	reIsVariable = regexp.MustCompile(`\$\w+(-\w+)*`)
)
