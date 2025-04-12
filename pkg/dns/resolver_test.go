package dns

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryHost(t *testing.T) {

	r, err := NewResolver(Options{
		Servers: []string{"8.8.8.8", "1.1.1.1:53"},
		Order:   []string{"a"},
	})
	assert.NoError(t, err)

	result, err := r.Lookup(context.Background(), "www.google.com")
	assert.NoError(t, err)
	t.Log(result)
	assert.GreaterOrEqual(t, len(result), 1)

	_, err = r.Lookup(context.Background(), "www.xxx.xxxssssssss")
	assert.ErrorIs(t, err, ErrNotFound)

}

func TestLocalNameServer(t *testing.T) {
	servers := GetDNSServers()
	assert.GreaterOrEqual(t, len(servers), 1)

	validServers, err := ValidateDNSServer([]string{servers[0].String()})
	assert.NoError(t, err)
	assert.Equal(t, len(validServers), 1)
}
