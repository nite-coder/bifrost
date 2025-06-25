package variable

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/tracer/traceinfo"
	"github.com/stretchr/testify/assert"
)

func TestGetDirective(t *testing.T) {
	hzCtx := app.NewContext(0)

	hzCtx.Set(ServerID, "serverA")
	hzCtx.Set("user_id", 98765)
	hzCtx.Set("enabled", true)
	hzCtx.Set("money", "123.456")
	hzCtx.Request.Header.SetUserAgentBytes([]byte("my_user_agent"))
	hzCtx.Set(RouteID, "routeA")
	hzCtx.Set(ServiceID, "serviceA")
	hzCtx.Set("myservice", "serviceA")
	hzCtx.Set(UpstreamID, "upstreamA")
	hzCtx.Set(UpstreamRequestHost, "1.2.3.4")
	hzCtx.Set(UpstreamResponoseStatusCode, 200)
	hzCtx.Set(UpstreamDuration, time.Duration(1*time.Second))
	hzCtx.Set(UpstreamConnAcquisitionTime, time.Duration(1*time.Microsecond))
	hzCtx.Set(HTTPRoute, "/orders/{order_id}")

	tracerInfo := traceinfo.NewTraceInfo()
	hzCtx.SetTraceInfo(tracerInfo)

	hzCtx.SetClientIPFunc(func(ctx *app.RequestContext) string {
		return "127.0.0.1"
	})
	hzCtx.Request.SetHost("abc.com")
	hzCtx.Request.Header.SetProtocol("HTTP/1.1")
	hzCtx.Request.SetMethod("POST")
	hzCtx.Request.SetRequestURI("http://abc.com/foo?bar=baz")
	hzCtx.Request.SetBody([]byte("hello world"))
	hzCtx.Response.Header.Set("x-trace-id", "1234")
	hzCtx.Request.SetCookie("hello", "world")

	reqInfo := &RequestOriginal{
		ServerID: "serverA",
		Host:     hzCtx.Request.Host(),
		Path:     hzCtx.Request.Path(),
		Protocol: hzCtx.Request.Header.GetProtocol(),
		Method:   MethodToString(hzCtx.Request.Method()),
		Query:    hzCtx.Request.QueryString(),
		Scheme:   hzCtx.Request.Scheme(),
	}
	hzCtx.Set(RequestOrig, reqInfo)

	reqRoute := &RequestRoute{
		RouteID:   "routeA",
		Route:     "/orders/{order_id}",
		ServiceID: "$var.myservice",
		Tags:      []string{"tag1", "tag2"},
	}

	hzCtx.Set(BifrostRoute, reqRoute)

	userID := GetInt64("$var.user_id", hzCtx)
	assert.Equal(t, int64(98765), userID)

	userID32 := GetInt32("$var.user_id", hzCtx)
	assert.Equal(t, int32(98765), userID32)

	enabled := GetBool("$var.enabled", hzCtx)
	assert.Equal(t, true, enabled)

	money := GetFloat64("$var.money", hzCtx)
	assert.Equal(t, 123.456, money)

	money32 := GetFloat32("$var.money", hzCtx)
	assert.Equal(t, float32(123.456), money32)

	remoteAddr := GetString(NetworkPeerAddress, hzCtx)
	assert.Equal(t, "0.0.0.0", remoteAddr)

	host := GetString(HTTPRequestHost, hzCtx)
	assert.Equal(t, "abc.com", host)

	receivedSize := GetString(HTTPRequestSize, hzCtx)
	assert.Equal(t, "0", receivedSize)

	sendSize := GetString(HTTPResponseSize, hzCtx)
	assert.Equal(t, "0", sendSize)

	httpStart := GetTime(HTTPStart, hzCtx)
	assert.True(t, httpStart.IsZero())

	httpFinish := GetTime(HTTPFinish, hzCtx)
	assert.False(t, httpFinish.IsZero())

	clientIP := GetString(ClientIP, hzCtx)
	assert.Equal(t, "127.0.0.1", clientIP)

	serverID := GetString(ServerID, hzCtx)
	assert.Equal(t, "serverA", serverID)

	routeID := GetString(RouteID, hzCtx)
	assert.Equal(t, "routeA", routeID)

	serviceID := GetString(ServiceID, hzCtx)
	assert.Equal(t, "serviceA", serviceID)

	serviceID = GetString("$var.myservice", hzCtx)
	assert.Equal(t, "serviceA", serviceID)

	upstreamID := GetString(UpstreamID, hzCtx)
	assert.Equal(t, "upstreamA", upstreamID)

	request := GetString(HTTPRequest, hzCtx)
	assert.Equal(t, "POST /foo?bar=baz HTTP/1.1", request)

	requestScheme := GetString(HTTPRequestScheme, hzCtx)
	assert.Equal(t, "http", requestScheme)

	requestMethod := GetString(HTTPRequestMethod, hzCtx)
	assert.Equal(t, "POST", requestMethod)

	requestBody := GetString(HTTPRequestBody, hzCtx)
	assert.Equal(t, "hello world", requestBody)

	requestPath := GetString(HTTPRequestPath, hzCtx)
	assert.Equal(t, "/foo", requestPath)

	requestURI := GetString(HTTPRequestURI, hzCtx)
	assert.Equal(t, "/foo?bar=baz", requestURI)

	requestProtocol := GetString(HTTPRequestProtocol, hzCtx)
	assert.Equal(t, "HTTP/1.1", requestProtocol)

	upstream := GetString(UpstreamRequest, hzCtx)
	assert.Equal(t, "POST /foo?bar=baz HTTP/1.1", upstream)

	upstreamURI := GetString(UpstreamRequestURI, hzCtx)
	assert.Equal(t, "/foo?bar=baz", upstreamURI)

	upstreamPath := GetString(UpstreamRequestPath, hzCtx)
	assert.Equal(t, "/foo", upstreamPath)

	upstreamAddr := GetString(UpstreamRequestHost, hzCtx)
	assert.Equal(t, "1.2.3.4", upstreamAddr)

	upstreamMethod := GetString(UpstreamRequestMethod, hzCtx)
	assert.Equal(t, "POST", upstreamMethod)

	upstreamProtocol := GetString(UpstreamRequestProtocol, hzCtx)
	assert.Equal(t, "HTTP/1.1", upstreamProtocol)

	upstreamStatusCode := GetString(UpstreamResponoseStatusCode, hzCtx)
	assert.Equal(t, "200", upstreamStatusCode)

	httpRoute := GetString(HTTPRoute, hzCtx)
	assert.Equal(t, "/orders/{order_id}", httpRoute)

	userAgent := GetString("$http.request.header.user-Agent", hzCtx)
	assert.Equal(t, "my_user_agent", userAgent)

	traceID := GetString("$http.response.header.x-trace-id", hzCtx)
	assert.Equal(t, "1234", traceID)

	queryBar := GetString("$http.request.query.bar", hzCtx)
	assert.Equal(t, "baz", queryBar)

	errType := GetString(ErrorType, hzCtx)
	assert.Equal(t, "", errType)

	hostname := GetString(Hostname, hzCtx)
	assert.NotEmpty(t, hostname)

	val, found := Get(HTTPRequestDuration, hzCtx)
	assert.False(t, found)
	assert.Nil(t, val)

	val, found = Get(UpstreamDuration, hzCtx)
	assert.True(t, found)
	assert.Equal(t, "1", val)

	val, found = Get(UpstreamConnAcquisitionTime, hzCtx)
	assert.True(t, found)
	assert.Equal(t, "0.000001", val)

	val, found = Get("", hzCtx)
	assert.False(t, found)
	assert.Nil(t, val)

	val, found = Get("aaa", nil)
	assert.False(t, found)
	assert.Nil(t, val)

	val, found = Get(HTTPRequestTags, hzCtx)
	assert.True(t, found)
	tags := val.([]string)
	assert.Equal(t, 2, len(tags))

	aaa := GetString(HTTPRequestTags, hzCtx)
	fmt.Println("aaa", aaa)

	cookie := GetString("$http.request.cookie.hello", hzCtx)
	assert.Equal(t, "world", cookie)
}

func TestGetVariable(t *testing.T) {
	hzCtx := app.NewContext(0)

	hzCtx.Set("uid", "123456")
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.URI().SetPath("/foo")

	uid, found := Get("$var.uid", hzCtx)
	assert.True(t, found)
	assert.Equal(t, "123456", uid)

	val, found := Get("$var.aaa", nil)
	assert.False(t, found)
	assert.Nil(t, val)
}

func TestIsDirective(t *testing.T) {
	assert.True(t, IsDirective("$var.uid"))
	assert.True(t, IsDirective("$http.request"))
	assert.True(t, IsDirective("$http.request.header.user-Agent"))
	assert.True(t, IsDirective("$http.response.header.x-trace-id"))
	assert.True(t, IsDirective("$upstream.conn_acquisition_time"))
	assert.False(t, IsDirective("$abc"))
}

func TestEnvDirective(t *testing.T) {
	os.Setenv("foo", "bar")
	val, _ := Get("$env.foo", nil)
	assert.Equal(t, "bar", val)

	os.Setenv("BOO", "boo")
	val, _ = Get("$env.BOO", nil)
	assert.Equal(t, "boo", val)
}

func TestParseDirectives(t *testing.T) {

	template := `{"time":"$time",
	"remote_addr":"$network.peer.address",
	"host": "$http.request.host",
	"request":"$http.request",
	"req_body":"$http.request.body"}`

	directives := ParseDirectives(template)
	assert.Equal(t, 5, len(directives))

	template = "/orders/$type"
	directives = ParseDirectives(template)
	assert.Equal(t, 1, len(directives))
	assert.Equal(t, "$type", directives[0])
}

func TestJSONPath(t *testing.T) {
	hzCtx := app.NewContext(0)
	hzCtx.Request.SetMethod("POST")
	hzCtx.Request.URI().SetPath("/foo")

	body := []byte(`{"name":{"first":"Janet","last":"Prichard"},"age":47}`)
	hzCtx.Request.SetBody(body)
	hzCtx.Response.SetBody([]byte(`{"student":{"first":"Janet","last":"Prichard"},"age":47}`))

	val := GetString("$http.request.body.json.name.last", hzCtx)
	assert.Equal(t, "Prichard", val)

	val = GetString("$http.request.body.json.name_not_Found", hzCtx)
	assert.Equal(t, "", val)

	val = GetString("$http.response.body.json.age", hzCtx)
	assert.Equal(t, "47", val)
}
