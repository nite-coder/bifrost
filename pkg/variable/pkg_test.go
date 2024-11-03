package variable

import (
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/blackbear/pkg/cast"
	"github.com/stretchr/testify/assert"
)

func TestGetDirective(t *testing.T) {
	hzCtx := app.NewContext(0)

	hzCtx.Set(SERVER_ID, "serverA")
	hzCtx.Request.Header.SetUserAgentBytes([]byte("my_user_agent"))
	hzCtx.Set(TRACE_ID, "trace_id")
	hzCtx.SetClientIPFunc(func(ctx *app.RequestContext) string {
		return "127.0.0.1"
	})
	hzCtx.Request.SetMethod("GET")
	hzCtx.Request.SetRequestURI("/foo?bar=baz")

	val, found := Get("$client_ip", hzCtx)
	assert.True(t, found)
	assert.Equal(t, "127.0.0.1", val)

	val, found = Get(SERVER_ID, hzCtx)
	assert.True(t, found)
	assert.Equal(t, "serverA", val)

	val, found = Get(REQUEST_PATH, hzCtx)
	assert.True(t, found)
	assert.Equal(t, "/foo", val)

	val, found = Get(REQUEST_URI, hzCtx)
	assert.True(t, found)
	assert.Equal(t, "/foo?bar=baz", val)

	val, found = Get(UserAgent, hzCtx)
	userAgent, _ := cast.ToString(val)
	assert.True(t, found)
	assert.Equal(t, "my_user_agent", userAgent)

	val, found = Get(TRACE_ID, hzCtx)
	traceID, _ := cast.ToString(val)
	assert.True(t, found)
	assert.Equal(t, "trace_id", traceID)

	val, found = Get(DURATION, hzCtx)
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
