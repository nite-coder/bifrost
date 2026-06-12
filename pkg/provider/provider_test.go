package provider_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nite-coder/bifrost/pkg/provider"
)

func TestDiscoveryResult_Fields(t *testing.T) {
	r := provider.DiscoveryResult{
		Target: "example.com:80",
		Weight: 100,
		Tags:   map[string]string{"region": "us"},
	}
	assert.Equal(t, "example.com:80", r.Target)
	assert.Equal(t, uint32(100), r.Weight)
	assert.Equal(t, "us", r.Tags["region"])
}
