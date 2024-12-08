package variable

import (
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestGetDirective(t *testing.T) {
	hzCtx := app.NewContext(0)

	hzCtx.Set(ServerID, "serverA")
	hzCtx.Set("user_id", 98765)
	hzCtx.Set("enabled", true)
	hzCtx.Set("money", "123.456")
	hzCtx.Request.Header.SetUserAgentBytes([]byte("my_user_agent"))
	hzCtx.Set(TraceID, "trace_id")
	hzCtx.SetClientIPFunc(func(ctx *app.RequestContext) string {
		return "127.0.0.1"
	})
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.SetRequestURI("/foo?bar=baz")

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

	clientIP := GetString("$client_ip", hzCtx)
	assert.Equal(t, "127.0.0.1", clientIP)

	serverA := GetString(ServerID, hzCtx)
	assert.Equal(t, "serverA", serverA)

	path := GetString(RequestPath, hzCtx)
	assert.Equal(t, "/foo", path)

	uri := GetString(RequestURI, hzCtx)
	assert.Equal(t, "/foo?bar=baz", uri)

	userAgent := GetString(UserAgent, hzCtx)
	assert.Equal(t, "my_user_agent", userAgent)

	traceID := GetString(TraceID, hzCtx)
	assert.Equal(t, "trace_id", traceID)

	val, found := Get(Duration, hzCtx)
	assert.False(t, found)
	assert.Nil(t, val)

	val, found = Get("", hzCtx)
	assert.False(t, found)
	assert.Nil(t, val)

	val, found = Get("aaa", nil)
	assert.False(t, found)
	assert.Nil(t, val)
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
