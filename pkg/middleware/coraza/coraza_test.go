package coroza

import (
	"context"
	"testing"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/stretchr/testify/assert"
)

func TestCorazaMiddleware_ServeHTTP(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		method   string
		body     []byte
		headers  map[string]string
		wantCode int
	}{
		{
			name:     "Normal Request",
			path:     "/test",
			method:   "GET",
			body:     nil,
			headers:  map[string]string{"Content-Type": "application/json"},
			wantCode: 200,
		},
		{
			name:     "Missing Content-Type Header",
			path:     "/test",
			method:   "POST",
			body:     []byte("{\"key\":\"value\"}"),
			headers:  nil,
			wantCode: 200,
		},
		{
			name:     "Path Traversal Attack",
			path:     "/test/../../../etc/passwd",
			method:   "GET",
			body:     nil,
			headers:  map[string]string{"Content-Type": "application/json"},
			wantCode: 403,
		},
		{
			name:     "Command Injection Attack",
			path:     "/test?cmd=cat /etc/passwd",
			method:   "GET",
			body:     nil,
			headers:  map[string]string{"Content-Type": "application/json"},
			wantCode: 403,
		},
		{
			name:   "XSS in User-Agent Header",
			path:   "/test",
			method: "GET",
			body:   nil,
			headers: map[string]string{
				"Content-Type": "application/json",
				"User-Agent":   "<script>alert(document.cookie)</script>",
			},
			wantCode: 403,
		},
		{
			name:   "SQLi in Cookie",
			path:   "/test?id=1' OR '1'='1",
			method: "GET",
			body:   nil,
			headers: map[string]string{
				"Content-Type": "application/json",
				"Cookie":       "id=1' OR '1'='1",
			},
			wantCode: 403,
		},
	}

	options := Options{
		Directives: `
			Include @coraza.conf-recommended
			Include @crs-setup.conf.example
			Include @owasp_crs/*.conf
			SecRuleEngine On
			`,
	}
	m, err := NewMiddleware(options)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hzctx := app.NewContext(0)
			hzctx.Request.SetRequestURI(tt.path)
			hzctx.Request.SetMethod(tt.method)
			hzctx.Request.Header.SetProtocol("HTTP/1.1")
			hzctx.Request.Header.Set("Host", "localhost")

			if tt.headers != nil {
				for k, v := range tt.headers {
					if k == "Cookie" {
						hzctx.Request.Header.SetCookie(k, v)
					} else {
						hzctx.Request.Header.Set(k, v)
					}
				}
			}

			if tt.body != nil {
				hzctx.Request.SetBody(tt.body)
			}

			m.ServeHTTP(context.Background(), hzctx)

			if code := hzctx.Response.StatusCode(); code != tt.wantCode {
				t.Errorf("%s: got status code %d, want %d", tt.name, code, tt.wantCode)
			}
		})
	}
}

func TestCorazaMiddleware_SQLInjection(t *testing.T) {
	options := Options{
		Directives: `
			Include @coraza.conf-recommended
			Include @crs-setup.conf.example
			Include @owasp_crs/*.conf
			SecRuleEngine On
			`,
	}
	middleware, err := NewMiddleware(options)
	assert.Nil(t, err)

	t.Run("SQL Injection Blocked", func(t *testing.T) {
		path := "/test?input=1' UNION SELECT 1,2,3 FROM users WHERE 1=1 AND (SELECT 1 FROM users LIMIT 1) LIKE '1' OR SLEEP(5)-- " // SQLi payload
		hzctx := app.NewContext(0)
		hzctx.Request.SetRequestURI(path)
		hzctx.Request.SetMethod("GET")
		hzctx.Request.Header.SetProtocol("HTTP/1.1")
		hzctx.Request.Header.Set("Host", "localhost")
		hzctx.Request.Header.Set("Content-Type", "application/json")
		hzctx.Request.Header.Set("User-Agent", "Mozilla/5.0")

		middleware.ServeHTTP(context.Background(), hzctx)
		if code := hzctx.Response.StatusCode(); code != 403 {
			t.Errorf("%s: got status code %d, want %d", "SQL Injection Blocked", code, 403)
		}
	})

	t.Run("Simple SQL Injection Blocked", func(t *testing.T) {
		path := "/test?input=1%20OR%201=1" // SQLi payload
		hzctx := app.NewContext(0)
		hzctx.Request.SetRequestURI(path)
		hzctx.Request.SetMethod("POST")
		hzctx.Request.Header.SetProtocol("HTTP/1.1")
		hzctx.Request.Header.Set("Host", "localhost")
		hzctx.Request.Header.Set("Content-Type", "application/json")
		hzctx.Request.Header.Set("User-Agent", "Mozilla/5.0")

		middleware.ServeHTTP(context.Background(), hzctx)
		if code := hzctx.Response.StatusCode(); code != 403 {
			t.Errorf("%s: got status code %d, want %d", "SQL Injection Blocked", code, 403)
		}
	})

}
