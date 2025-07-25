package hasing

import (
	"context"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/balancer"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/proxy"
	httpproxy "github.com/nite-coder/bifrost/pkg/proxy/http"
	"github.com/stretchr/testify/assert"
)

func TestHashing(t *testing.T) {
	proxyOptions1 := httpproxy.Options{
		Target:      "http://backend1",
		Protocol:    config.ProtocolHTTP,
		FailTimeout: 10 * time.Minute,
		MaxFails:    1,
	}
	proxy1, _ := httpproxy.New(proxyOptions1, nil)

	proxyOptions2 := httpproxy.Options{
		Target:      "http://backend2",
		Protocol:    config.ProtocolHTTP,
		FailTimeout: 10 * time.Minute,
		MaxFails:    1,
	}
	proxy2, _ := httpproxy.New(proxyOptions2, nil)

	proxyOptions3 := httpproxy.Options{
		Target:      "http://backend3",
		Protocol:    config.ProtocolHTTP,
		FailTimeout: 10 * time.Minute,
		MaxFails:    1,
	}
	proxy3, _ := httpproxy.New(proxyOptions3, nil)

	proxies := []proxy.Proxy{
		proxy1,
		proxy2,
		proxy3,
	}

	t.Run("success", func(t *testing.T) {
		keys := []string{"key1", "key2", "key3"}
		expected := map[string]string{
			"key1": "http://backend3",
			"key2": "http://backend2",
			"key3": "http://backend1",
		}

		for _, key := range keys {
			b := NewBalancer(proxies, "$var.uid")
			hzctx := app.NewContext(0)
			hzctx.Set("uid", key)
			proxy, err := b.Select(context.Background(), hzctx)
			assert.NoError(t, err)
			assert.NotNil(t, proxy)
			assert.Equal(t, expected[key], proxy.Target())
		}
	})

	t.Run("two proxies failed", func(t *testing.T) {
		_ = proxy1.AddFailedCount(100)
		_ = proxy2.AddFailedCount(100)

		keys := []string{"key1", "key2", "key3"}

		for _, key := range keys {
			b := NewBalancer(proxies, "$var.uid")
			hzctx := app.NewContext(0)
			hzctx.Set("uid", key)
			proxy, err := b.Select(context.Background(), hzctx)
			assert.NoError(t, err)
			assert.NotNil(t, proxy)
			assert.Equal(t, "http://backend3", proxy.Target())
		}
	})

	t.Run("no live upstream", func(t *testing.T) {
		_ = proxy1.AddFailedCount(100)
		_ = proxy2.AddFailedCount(100)
		_ = proxy3.AddFailedCount(100)

		keys := []string{"key1", "key2", "key3"}

		for _, key := range keys {
			b := NewBalancer(proxies, "$var.uid")
			hzctx := app.NewContext(0)
			hzctx.Set("uid", key)
			proxy, err := b.Select(context.Background(), hzctx)
			assert.ErrorIs(t, err, balancer.ErrNotAvailable)
			assert.Nil(t, proxy)
		}
	})
}
