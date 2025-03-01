package variable

import (
	"bytes"
	"net"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/tracer/stats"
	"github.com/nite-coder/bifrost/pkg/timecache"
	"github.com/nite-coder/blackbear/pkg/cast"
	"github.com/valyala/bytebufferpool"
)

var (
	reIsVariable    = regexp.MustCompile(`\$\w+(?:[._-]\w+)*`)
	grpcContentType = []byte("application/grpc")
	questionByte    = []byte{byte('?')}
	spaceByte       = []byte{byte(' ')}
	directives      = map[string]struct{}{
		Time:                        {},
		ClientIP:                    {},
		NetworkPeerAddress:          {},
		HTTPRequestHost:             {},
		ServerID:                    {},
		RouteID:                     {},
		Hostname:                    {},
		ServiceID:                   {},
		TraceID:                     {},
		ErrorType:                   {},
		ErrorMessage:                {},
		HTTPRequestSize:             {},
		HTTPResponseSize:            {},
		HTTPStart:                   {},
		HTTPFinish:                  {},
		HTTPRoute:                   {},
		HTTPRequest:                 {},
		HTTPRequestScheme:           {},
		HTTPRequestMethod:           {},
		HTTPRequestPath:             {},
		HTTPRequestQuery:            {},
		HTTPRequestBody:             {},
		HTTPRequestURI:              {},
		HTTPRequestProtocol:         {},
		HTTPResponseStatusCode:      {},
		UpstreamRequest:             {},
		UpstreamID:                  {},
		UpstreamRequestProtocol:     {},
		UpstreamRequestMethod:       {},
		UpstreamRequestHost:         {},
		UpstreamRequestPath:         {},
		UpstreamRequestQuery:        {},
		UpstreamRequestURI:          {},
		UpstreamResponoseStatusCode: {},
		UpstreamDuration:            {},
		HTTPRequestDuration:         {},
		GRPCStatusCode:              {},
		GRPCMessage:                 {},
	}
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

func IsDirective(key string) bool {
	if strings.HasPrefix(key, "$var.") || strings.HasPrefix(key, "$http.request.header.") || strings.HasPrefix(key, "$http.response.header.") || strings.HasPrefix(key, "$http.request.query.") {
		return true
	}

	if !strings.HasPrefix(key, "$") {
		return false
	}

	if _, found := directives[key]; found {
		return true
	}

	return false
}

var (
	hostnameOnce sync.Once
	hostname     string
)

func directive(key string, c *app.RequestContext) (val any, found bool) {
	switch key {
	case Time:
		now := timecache.Now()
		return now, true
	case ClientIP:
		return c.ClientIP(), true
	case Hostname:
		hostnameOnce.Do(func() {
			if hostname == "" {
				hostname, _ = os.Hostname()
			}
		})
		return hostname, true
	case HTTPRequestHost:
		val, found := c.Get(RequestOrig)
		if !found {
			return nil, false
		}
		info := (val).(*RequestOriginal)

		host := cast.B2S(info.Host)
		return host, true
	case ServerID:
		val, found := c.Get(RequestOrig)
		if !found {
			return nil, false
		}
		info := (val).(*RequestOriginal)
		return info.ServerID, true
	case NetworkPeerAddress:
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
	case TraceID:
		return c.GetString(TraceID), true
	case ErrorType:
		return c.GetString(ErrorType), true
	case ErrorMessage:
		return c.GetString(ErrorMessage), true
	case HTTPRequestSize:
		traceInfo := c.GetTraceInfo()
		if traceInfo == nil {
			return nil, false
		}
		httpStats := traceInfo.Stats()
		if httpStats == nil {
			return nil, false
		}
		return httpStats.RecvSize(), true
	case HTTPResponseSize:
		traceInfo := c.GetTraceInfo()
		if traceInfo == nil {
			return nil, false
		}
		httpStats := traceInfo.Stats()
		if httpStats == nil {
			return nil, false
		}
		return httpStats.SendSize(), true
	case HTTPStart:
		traceInfo := c.GetTraceInfo()
		if traceInfo == nil {
			return nil, false
		}
		httpStats := traceInfo.Stats()
		if httpStats == nil {
			return nil, false
		}

		event := httpStats.GetEvent(stats.HTTPStart)
		if event == nil {
			return nil, false
		}

		start := event.Time().UnixMicro()
		return start, true
	case HTTPFinish:
		traceInfo := c.GetTraceInfo()
		if traceInfo == nil {
			return timecache.Now().UnixMicro(), true
		}
		httpStats := traceInfo.Stats()
		if httpStats == nil {
			return timecache.Now().UnixMicro(), true
		}

		event := httpStats.GetEvent(stats.HTTPFinish)
		if event == nil {
			return timecache.Now().UnixMicro(), true
		}

		finish := event.Time().UnixMicro()
		return finish, true
	case HTTPRequest:
		val, found := c.Get(RequestOrig)
		if !found {
			return nil, false
		}
		info := (val).(*RequestOriginal)

		builder := strings.Builder{}
		builder.Write(info.Method)
		builder.Write(spaceByte)
		builder.Write(info.Path)
		if len(info.Query) > 0 {
			builder.Write(questionByte)
			builder.Write(info.Query)
		}

		builder.Write(spaceByte)
		builder.WriteString(info.Protocol)
		return builder.String(), true
	case HTTPRoute:
		val, found := c.Get(BifrostRoute)
		if !found {
			return nil, false
		}
		info := (val).(*RequestRoute)
		return info.Route, true
	case HTTPRequestScheme:
		val, found := c.Get(RequestOrig)
		if !found {
			return nil, false
		}
		info := (val).(*RequestOriginal)

		scheme := cast.B2S(info.Scheme)
		return scheme, true
	case HTTPRequestPath:
		val, found := c.Get(RequestOrig)
		if !found {
			return nil, false
		}
		info := (val).(*RequestOriginal)

		path := cast.B2S(info.Path)
		return path, true
	case HTTPRequestURI:
		val, found := c.Get(RequestOrig)
		if !found {
			return nil, false
		}
		info := (val).(*RequestOriginal)

		builder := strings.Builder{}
		builder.Write(info.Path)
		if len(info.Query) > 0 {
			builder.Write(questionByte)
			builder.Write(info.Query)
		}

		return builder.String(), true
	case HTTPRequestMethod:
		val, found := c.Get(RequestOrig)
		if !found {
			return nil, false
		}
		info := (val).(*RequestOriginal)

		method := cast.B2S(info.Method)
		return method, true
	case HTTPRequestQuery:
		val, found := c.Get(RequestOrig)
		if !found {
			return nil, false
		}
		info := (val).(*RequestOriginal)

		query := cast.B2S(info.Query)
		return query, true
	case HTTPRequestBody:
		// if content type is grpc, the $request_body will be ignored
		contentType := c.Request.Header.ContentType()
		if bytes.Equal(contentType, grpcContentType) {
			return "", true
		}

		return cast.B2S(c.Request.Body()), true
	case HTTPRequestProtocol:
		val, found := c.Get(RequestOrig)
		if !found {
			return nil, false
		}
		info := (val).(*RequestOriginal)
		return info.Protocol, true

	case RouteID:
		val, found := c.Get(BifrostRoute)
		if !found {
			return nil, false
		}
		info := (val).(*RequestRoute)
		return info.RouteID, true
	case ServiceID:
		val, found := c.Get(BifrostRoute)
		if !found {
			return nil, false
		}
		info := (val).(*RequestRoute)
		return info.ServiceID, true
	case UpstreamID:
		upstream := c.GetString(UpstreamID)
		return upstream, true
	case UpstreamRequest:
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)

		_, _ = buf.Write(c.Request.Method())
		_, _ = buf.Write(spaceByte)

		_, _ = buf.Write(c.Request.Path())
		if len(c.Request.QueryString()) > 0 {
			_, _ = buf.Write(questionByte)
			_, _ = buf.Write(c.Request.QueryString())
		}

		_, _ = buf.Write(spaceByte)
		_, _ = buf.WriteString(c.Request.Header.GetProtocol())

		return buf.String(), true
	case UpstreamRequestURI:
		buf := bytebufferpool.Get()
		defer bytebufferpool.Put(buf)

		_, _ = buf.Write(c.Request.Path())
		if len(c.Request.QueryString()) > 0 {
			_, _ = buf.Write(questionByte)
			_, _ = buf.Write(c.Request.QueryString())
		}

		return buf.String(), true
	case UpstreamRequestProtocol:
		return c.Request.Header.GetProtocol(), true
	case UpstreamRequestMethod:
		method := string(c.Request.Method())
		return method, true
	case UpstreamRequestPath:
		return cast.B2S(c.Request.Path()), true
	case UpstreamRequestHost:
		addr := c.GetString(UpstreamRequestHost)
		return addr, true
	case UpstreamRequestQuery:
		query := cast.B2S(c.Request.QueryString())
		return query, true
	case UpstreamResponoseStatusCode:
		status := c.GetInt(UpstreamResponoseStatusCode)
		return status, true
	case UpstreamDuration:
		dur := c.GetDuration(UpstreamDuration)
		mic := dur.Microseconds()
		duration := float64(mic) / 1e6
		responseTime := strconv.FormatFloat(duration, 'f', -1, 64)
		return responseTime, true
	case HTTPRequestDuration:
		traceInfo := c.GetTraceInfo()
		if traceInfo == nil {
			return nil, false
		}
		httpStats := traceInfo.Stats()
		if httpStats == nil {
			return nil, false
		}

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
	case GRPCStatusCode:
		status := c.GetString(GRPCStatusCode)
		return status, true
	case GRPCMessage:
		grpcMessage := c.GetString(GRPCMessage)
		return grpcMessage, true
	default:
		if strings.HasPrefix(key, "$http.request.header.") {
			headerKey := key[len("$http.request.header."):]

			if len(headerKey) == 0 {
				return "", false
			}

			headerVal := c.Request.Header.Get(headerKey)
			return headerVal, true
		}

		if strings.HasPrefix(key, "$http.response.header.") {
			headerKey := key[len("$http.response.header."):]

			if len(headerKey) == 0 {
				return "", false
			}

			headerVal := c.Response.Header.Get(headerKey)
			return headerVal, true
		}

		if strings.HasPrefix(key, "$http.request.query.") {
			queryKey := key[len("$http.request.query."):]

			if len(queryKey) == 0 {
				return "", false
			}

			val := c.Query(queryKey)
			return val, true
		}

		return nil, false
	}
}

func ParseDirectives(content string) []string {
	variables := reIsVariable.FindAllString(content, -1)
	sortBifrostVariables(variables)
	return variables
}

type byLengthAndContent []string

func (s byLengthAndContent) Len() int {
	return len(s)
}

func (s byLengthAndContent) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byLengthAndContent) Less(i, j int) bool {
	if len(s[i]) == len(s[j]) {
		return s[i] < s[j]
	}

	return len(s[i]) > len(s[j])
}

func sortBifrostVariables(slice []string) {
	sort.Sort(byLengthAndContent(slice))
}
