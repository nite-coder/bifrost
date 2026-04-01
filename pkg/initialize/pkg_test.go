package initialize

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	err := Bifrost()
	require.NoError(t, err)
}
