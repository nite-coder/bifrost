package variable

import (
	"bytes"
	"net"
	"strconv"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/tracer/stats"
	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/blackbear/pkg/cast"
	"github.com/valyala/bytebufferpool"
)

var (
	grpcContentType = []byte("application/grpc")
	questionByte    = []byte{byte('?')}
)

func Get(key string, c *app.RequestContext) (val any, found bool) {
	key = strings.TrimSpace(key)
	key = strings.ToLower(key)

	if key == "" || key[0] != '$' || c == nil {
		return nil, false
	}

	if strings.HasPrefix(key, "$var.") {
		key = key[5:]
		return c.Get(key)
	}

	return directive(key, c)

}

func directive(key string, c *app.RequestContext) (val any, found bool) {
	switch key {
	case TIME:
		now := timecache.Now()
		return now, true
	case ClientIP:
		return c.ClientIP(), true
	case HOST:
		val, found := c.Get(HOST)

		if found {
			b := val.([]byte)
			host := cast.B2S(b)
			return host, true
		}

		host := string(c.Request.Host())
		return host, true
	case SERVER_ID:
		serverID := c.GetString(SERVER_ID)
		return serverID, true
	case REMOTE_ADDR:
		var ip string
		switch addr := c.RemoteAddr().(type) {
		case *net.UDPAddr:
			ip = addr.IP.String()
		case *net.TCPAddr:
			ip = addr.IP.String()
		default:
			return "", false
		}
		return ip, true
	case RECEIVED_SIZE:
		traceInfo := c.GetTraceInfo()
		if traceInfo == nil {
			return nil, false
		}
		httpStats := traceInfo.Stats()
		if httpStats == nil {
			return 0, false
		}
		return httpStats.RecvSize(), true
	case SEND_SIZE:
		traceInfo := c.GetTraceInfo()
		if traceInfo == nil {
			return nil, false
		}
		httpStats := traceInfo.Stats()
		if httpStats == nil {
			return 0, false
		}
		return httpStats.SendSize(), true
	case REQUEST_PROTOCOL:
		return c.Request.Header.GetProtocol(), true
	case REQUEST_PATH:
		val, found := c.Get(REQUEST_PATH)

		if found {
			b := val.([]byte)
			path := cast.B2S(b)
			return path, true
		}

		path := string(c.Request.Path())
		return path, true
	case REQUEST_METHOD:
		method := string(c.Request.Method())
		return method, true
	case REQUEST_BODY:
		// if content type is grpc, the $request_body will be ignored
		contentType := c.Request.Header.ContentType()
		if bytes.Equal(contentType, grpcContentType) {
			return "", true
		}

		return cast.B2S(c.Request.Body()), true
	case TRACE_ID:
		traceID := c.GetString(TRACE_ID)
		return traceID, true
	case UPSTREAM:
		upstream := c.GetString(UPSTREAM)
		return upstream, true
	case UPSTREAM_URI:
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)

		_, _ = buf.Write(c.Request.Path())

		if len(c.Request.QueryString()) > 0 {
			_, _ = buf.Write(questionByte)
			_, _ = buf.Write(c.Request.QueryString())
		}

		return buf.String(), true
	case UPSTREAM_PROTOCOL:
		return c.Request.Header.GetProtocol(), true
	case UPSTREAM_METHOD:
		method := string(c.Request.Method())
		return method, true
	case UPSTREAM_PATH:
		return cast.B2S(c.Request.Path()), true
	case UPSTREAM_ADDR:
		addr := c.GetString(UPSTREAM_ADDR)
		return addr, true
	case DURATION:
		traceInfo := c.GetTraceInfo()
		if traceInfo == nil {
			return nil, false
		}
		httpStats := traceInfo.Stats()
		httpStart := httpStats.GetEvent(stats.HTTPStart)
		if httpStart == nil {
			return nil, false
		}

		httpFinish := httpStats.GetEvent(stats.HTTPFinish)
		if httpFinish == nil {
			return nil, false
		}

		dur := httpFinish.Time().Sub(httpStart.Time()).Microseconds()
		duration := strconv.FormatFloat(float64(dur)/1e6, 'f', -1, 64)
		return duration, true
	case GRPC_STATUS:
		status := c.GetString(GRPC_STATUS)
		return status, true
	case GRPC_MESSAGE:
		grpcMessage := c.GetString(GRPC_MESSAGE)
		return grpcMessage, true
	case UserAgent:
		return c.Request.Header.UserAgent(), true
	default:
		return nil, false
	}
}
