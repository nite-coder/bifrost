package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalNameServer(t *testing.T) {
	servers := GetDNSServers()
	assert.GreaterOrEqual(t, len(servers), 1)

	validServers, err := ValidateDNSServer([]string{servers[0].String()})
	if err != nil {
		t.Skip("skipping test due to DNS validation failure: " + err.Error())
	}
	assert.NoError(t, err)
	assert.Equal(t, len(validServers), 1)
}

func TestQueryHost(t *testing.T) {

	t.Run("default resolver", func(t *testing.T) {
		r, err := NewResolver(Options{SkipTest: true})
		require.NoError(t, err)

		result, err := r.Lookup(context.Background(), "www.google.com")
		if err != nil {
			t.Logf("lookup failed (expected in offline env): %v", err)
			return // stop here if lookup fails, don't fail test
		}
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
		require.NoError(t, err)

		result, err := r.Lookup(context.Background(), "www.google.com")
		if err != nil {
			t.Logf("lookup failed (expected in offline env): %v", err)
			return
		}
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
		require.NoError(t, err)

		result, err := r.Lookup(context.Background(), "test-cname-cloaking.testpanw.com")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(result), 1)
	})

}
