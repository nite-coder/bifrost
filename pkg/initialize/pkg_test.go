package initialize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	err := Bifrost()
	assert.NoError(t, err)
}
