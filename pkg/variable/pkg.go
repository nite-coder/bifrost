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

func GetString(key string, c *app.RequestContext) string {
	val, found := Get(key, c)
	if !found {
		return ""
	}
	result, _ := cast.ToString(val)
	return result
}

func GetInt64(key string, c *app.RequestContext) int64 {
	val, found := Get(key, c)
	if !found {
		return 0
	}
	result, _ := cast.ToInt64(val)
	return result
}

func GetInt32(key string, c *app.RequestContext) int32 {
	val, found := Get(key, c)
	if !found {
		return 0
	}
	result, _ := cast.ToInt32(val)
	return result
}

func GetFloat64(key string, c *app.RequestContext) float64 {
	val, found := Get(key, c)
	if !found {
		return 0
	}
	result, _ := cast.ToFloat64(val)
	return result
}

func GetFloat32(key string, c *app.RequestContext) float32 {
	val, found := Get(key, c)
	if !found {
		return 0
	}
	result, _ := cast.ToFloat32(val)
	return result
}

func GetBool(key string, c *app.RequestContext) bool {
	val, found := Get(key, c)
	if !found {
		return false
	}
	result, _ := cast.ToBool(val)
	return result
}

func directive(key string, c *app.RequestContext) (val any, found bool) {
	switch key {
	case Time:
		now := timecache.Now()
		return now, true
	case ClientIP:
		return c.ClientIP(), true
	case Host:
		val, found := c.Get(Host)

		if found {
			b := val.([]byte)
			host := cast.B2S(b)
			return host, true
		}

		host := string(c.Request.Host())
		return host, true
	case ServerID:
		serverID := c.GetString(ServerID)
		return serverID, true
	case RemoteAddr:
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
	case ReceivedSize:
		traceInfo := c.GetTraceInfo()
		if traceInfo == nil {
			return nil, false
		}
		httpStats := traceInfo.Stats()
		if httpStats == nil {
			return 0, false
		}
		return httpStats.RecvSize(), true
	case SendSize:
		traceInfo := c.GetTraceInfo()
		if traceInfo == nil {
			return nil, false
		}
		httpStats := traceInfo.Stats()
		if httpStats == nil {
			return 0, false
		}
		return httpStats.SendSize(), true
	case RequestProtocol:
		return c.Request.Header.GetProtocol(), true
	case RequestPath:
		val, found := c.Get(RequestPath)

		if found {
			b := val.([]byte)
			path := cast.B2S(b)
			return path, true
		}

		path := string(c.Request.URI().Path())
		return path, true
	case RequestURI:
		val := string(c.Request.URI().RequestURI())
		return val, true
	case RequestMethod:
		method := string(c.Request.Method())
		return method, true
	case RequestBody:
		// if content type is grpc, the $request_body will be ignored
		contentType := c.Request.Header.ContentType()
		if bytes.Equal(contentType, grpcContentType) {
			return "", true
		}

		return cast.B2S(c.Request.Body()), true
	case TraceID:
		traceID := c.GetString(TraceID)
		return traceID, true
	case RouteID:
		routeID := c.GetString(RouteID)
		return routeID, true
	case ServiceID:
		serviceID := c.GetString(ServiceID)
		return serviceID, true
	case Upstream:
		upstream := c.GetString(Upstream)
		return upstream, true
	case UpstreamURI:
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)

		_, _ = buf.Write(c.Request.Path())

		if len(c.Request.QueryString()) > 0 {
			_, _ = buf.Write(questionByte)
			_, _ = buf.Write(c.Request.QueryString())
		}

		return buf.String(), true
	case UpstreamProtocol:
		return c.Request.Header.GetProtocol(), true
	case UpstreamMethod:
		method := string(c.Request.Method())
		return method, true
	case UpstreamPath:
		return cast.B2S(c.Request.Path()), true
	case UpstreamAddr:
		addr := c.GetString(UpstreamAddr)
		return addr, true
	case Duration:
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
	case GRPCStatus:
		status := c.GetString(GRPCStatus)
		return status, true
	case GRPCMessage:
		grpcMessage := c.GetString(GRPCMessage)
		return grpcMessage, true
	case UserAgent:
		return c.Request.Header.UserAgent(), true
	default:
		return nil, false
	}
}
