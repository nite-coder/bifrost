package initialize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitMiddleware(t *testing.T) {
	err := Middleware()
	assert.NoError(t, err)
}
