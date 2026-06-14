package target_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nite-coder/bifrost/pkg/target"
)

func TestEndpointHash_Deterministic(t *testing.T) {
	eps := []*target.Endpoint{
		{Address: "10.0.1.1:80", Weight: 1, Tags: map[string]string{"region": "us"}},
		{Address: "10.0.1.2:80", Weight: 2, Tags: nil},
	}
	h1 := target.EndpointHash(eps)
	h2 := target.EndpointHash(eps)
	assert.Equal(t, h1, h2, "same input must produce same hash")

	eps2 := []*target.Endpoint{
		{Address: "10.0.1.2:80", Weight: 2, Tags: nil},
		{Address: "10.0.1.1:80", Weight: 1, Tags: map[string]string{"region": "us"}},
	}
	h3 := target.EndpointHash(eps2)
	assert.Equal(t, h1, h3, "order-independent hash")
}

func TestEndpointHash_ChangesOnDiff(t *testing.T) {
	ep := &target.Endpoint{Address: "10.0.1.1:80", Weight: 1}
	eps := []*target.Endpoint{ep}

	h := target.EndpointHash(eps)

	ep.Weight = 2
	h2 := target.EndpointHash([]*target.Endpoint{ep})
	assert.NotEqual(t, h, h2, "weight change must change hash")

	ep2 := &target.Endpoint{Address: "10.0.1.2:80", Weight: 1}
	h3 := target.EndpointHash([]*target.Endpoint{ep2})
	assert.NotEqual(t, h2, h3, "address change must change hash")
}

func TestEndpointHash_ExcludesState(t *testing.T) {
	s1 := target.NewState(1, 0)
	s2 := target.NewState(2, 0)
	ep := &target.Endpoint{Address: "10.0.1.1:80", Weight: 1, State: s1}
	eps := []*target.Endpoint{ep}
	h := target.EndpointHash(eps)

	ep.State = s2
	h2 := target.EndpointHash(eps)
	assert.Equal(t, h, h2, "State change must NOT change hash")
}
