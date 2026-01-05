package zero

import (
	"encoding/base64"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInheritedListeners_KeyParsing(t *testing.T) {
	// Function to simulate environment and test parsing
	// Note: We can't easily test actual File inheritance in unit test without spawning processes,
	// but we can test that the keys are decoded correctly which was the main logic change.

	t.Run("valid keys", func(t *testing.T) {
		keys := []string{"127.0.0.1:8080", "127.0.0.1:9090"}
		encoded := base64.StdEncoding.EncodeToString([]byte(strings.Join(keys, ",")))

		os.Setenv("UPGRADE", "1")
		os.Setenv("BIFROST_LISTENER_KEYS", encoded)
		defer os.Unsetenv("UPGRADE")
		defer os.Unsetenv("BIFROST_LISTENER_KEYS")

		// We expect this to try to open FDs.
		// Since FDs 3, 4 are likely invalid in this test process, it might return empty map or log errors,
		// but we want to ensure it DOES attempt to read based on keys.
		// However, InheritedListeners logic is tightly coupled with os.NewFile.
		// We can't assert on the return value easily if FDs are missing.

		// Instead, let's verify the environment variable set/get logic in a helper if we extracted it,
		// but since we didn't extract it, this test acts as a sanity check that it doesn't panic.
		// For a real test, we'd need to mock os.NewFile or use exec.

		// Calling it to ensure no panic
		listeners, err := InheritedListeners()
		assert.NoError(t, err)
		assert.NotNil(t, listeners)
		assert.Contains(t, listeners, "127.0.0.1:8080")
		assert.Contains(t, listeners, "127.0.0.1:9090")
	})

	t.Run("invalid base64", func(t *testing.T) {
		os.Setenv("UPGRADE", "1")
		os.Setenv("BIFROST_LISTENER_KEYS", "invalid-base64")
		defer os.Unsetenv("UPGRADE")
		defer os.Unsetenv("BIFROST_LISTENER_KEYS")

		listeners, err := InheritedListeners()
		assert.NoError(t, err)
		assert.Nil(t, listeners)
	})

	t.Run("not upgrade mode", func(t *testing.T) {
		os.Unsetenv("UPGRADE")
		listeners, err := InheritedListeners()
		assert.NoError(t, err)
		assert.Nil(t, listeners)
	})
}
