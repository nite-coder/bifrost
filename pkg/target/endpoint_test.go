package target_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nite-coder/bifrost/pkg/target"
)

func TestEndpoint_Fields(t *testing.T) {
	s := target.NewState(1, 0)
	ep := &target.Endpoint{
		Address: "10.0.1.5:8080",
		Weight:  10,
		Tags:    map[string]string{"server_name": "example.com"},
		State:   s,
	}
	assert.Equal(t, "10.0.1.5:8080", ep.Address)
	assert.Equal(t, uint32(10), ep.Weight)
	assert.Equal(t, "example.com", ep.Tags["server_name"])
	assert.Same(t, s, ep.State)
}
