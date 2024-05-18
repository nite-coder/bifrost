package gateway

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouters(t *testing.T) {
	r := NewRouter()

	err := r.add(POST, "/spot/orders", nil)
	assert.NoError(t, err)

	err = r.add(POST, "/futures/acc*", nil)
	assert.NoError(t, err)

	m := r.find(POST, "/spot/orders")
	assert.Len(t, m, 1)

	m = r.find(POST, "/futures/account")
	assert.Len(t, m, 1)
}
