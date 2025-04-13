package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLocalNameServer(t *testing.T) {
	servers := GetDNSServers()
	assert.GreaterOrEqual(t, len(servers), 1)

	validServers, err := ValidateDNSServer([]string{servers[0].String()})
	assert.NoError(t, err)
	assert.Equal(t, len(validServers), 1)
}

func TestQueryHost(t *testing.T) {

	t.Run("default resolver", func(t *testing.T) {
		r, err := NewResolver(Options{})
		assert.NoError(t, err)

		result, err := r.Lookup(context.Background(), "www.google.com")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(result), 1)

		result, err = r.Lookup(context.Background(), "192.168.1.1")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(result), 1)

		result, err = r.Lookup(context.Background(), "127.0.0.1")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(result), 1)

		_, err = r.Lookup(context.Background(), "www.xxx.xxxssssssss")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("specific dns server", func(t *testing.T) {
		r, err := NewResolver(Options{
			Servers: []string{"8.8.8.8", "1.1.1.1:53"},
			Order:   []string{"a"},
			Timeout: 1 * time.Second,
		})
		assert.NoError(t, err)

		result, err := r.Lookup(context.Background(), "www.google.com")
		assert.NoError(t, err)
		t.Log(result)
		assert.GreaterOrEqual(t, len(result), 1)

		_, err = r.Lookup(context.Background(), "www.xxx.xxxssssssss")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("cname resolver", func(t *testing.T) {
		r, err := NewResolver(Options{
			Servers: []string{"1.1.1.1:53"},
			Order:   []string{"cname"},
		})
		assert.NoError(t, err)

		result, err := r.Lookup(context.Background(), "test-cname-cloaking.testpanw.com")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(result), 1)
	})

}
