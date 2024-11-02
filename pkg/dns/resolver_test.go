package dns

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryHost(t *testing.T) {

	r, err := NewResolver(Options{})
	assert.NoError(t, err)

	result, err := r.Lookup(context.Background(), "www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	assert.GreaterOrEqual(t, len(result), 1)

	result, err = r.Lookup(context.Background(), "www.xxx.xxxssssssss")
	assert.ErrorIs(t, err, ErrNotFound)

}

func TestLocalNameServer(t *testing.T) {
	servers := GetDNSServers()
	assert.GreaterOrEqual(t, len(servers), 1)
}
