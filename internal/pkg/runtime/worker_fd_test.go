package runtime

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeListenerKeys(t *testing.T) {
	t.Run("valid keys", func(t *testing.T) {
		keys := []string{"127.0.0.1:8080", "127.0.0.1:9090"}
		encoded := base64.StdEncoding.EncodeToString([]byte(strings.Join(keys, ",")))

		decodedKeys, err := decodeListenerKeys(encoded)
		assert.NoError(t, err)
		assert.NotNil(t, decodedKeys)
		assert.Contains(t, decodedKeys, "127.0.0.1:8080")
		assert.Contains(t, decodedKeys, "127.0.0.1:9090")
	})

	t.Run("invalid base64", func(t *testing.T) {
		decodedKeys, err := decodeListenerKeys("invalid-base64")
		assert.Error(t, err)
		assert.Nil(t, decodedKeys)
	})

	t.Run("empty environment variable", func(t *testing.T) {
		decodedKeys, err := decodeListenerKeys("")
		assert.NoError(t, err)
		assert.Nil(t, decodedKeys)
	})
}
