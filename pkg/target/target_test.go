package target_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nite-coder/bifrost/pkg/target"
)

func TestTarget_Fields(t *testing.T) {
	tgt := &target.Target{
		Name:   "example.com:80",
		Weight: 100,
		Tags:   map[string]string{"region": "us"},
		Endpoints: map[string]*target.Endpoint{
			"10.0.1.1:80": {Address: "10.0.1.1:80", Weight: 100},
			"10.0.1.2:80": {Address: "10.0.1.2:80", Weight: 100},
		},
	}
	assert.Equal(t, "example.com:80", tgt.Name)
	assert.Equal(t, uint32(100), tgt.Weight)
	assert.Equal(t, "us", tgt.Tags["region"])
	assert.Len(t, tgt.Endpoints, 2)
}
