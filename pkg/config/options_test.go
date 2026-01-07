package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigPath(t *testing.T) {
	t.Run("returns stored config path", func(t *testing.T) {
		opts := Options{configPath: "/etc/bifrost/config.yaml"}
		assert.Equal(t, "/etc/bifrost/config.yaml", opts.ConfigPath())
	})

	t.Run("returns empty string when not set", func(t *testing.T) {
		opts := Options{}
		assert.Equal(t, "", opts.ConfigPath())
	})
}

func TestIsEnabled(t *testing.T) {
	t.Run("enabled is nil returns true", func(t *testing.T) {
		opts := ServerTracingOptions{Enabled: nil}
		assert.True(t, opts.IsEnabled())
	})

	t.Run("enabled is true returns true", func(t *testing.T) {
		enabled := true
		opts := ServerTracingOptions{Enabled: &enabled}
		assert.True(t, opts.IsEnabled())
	})

	t.Run("enabled is false returns false", func(t *testing.T) {
		enabled := false
		opts := ServerTracingOptions{Enabled: &enabled}
		assert.False(t, opts.IsEnabled())
	})
}

func TestIsPassHostHeader(t *testing.T) {
	t.Run("pass_host_header is nil returns true", func(t *testing.T) {
		opts := ServiceOptions{PassHostHeader: nil}
		assert.True(t, opts.IsPassHostHeader())
	})

	t.Run("pass_host_header is true returns true", func(t *testing.T) {
		passHost := true
		opts := ServiceOptions{PassHostHeader: &passHost}
		assert.True(t, opts.IsPassHostHeader())
	})

	t.Run("pass_host_header is false returns false", func(t *testing.T) {
		passHost := false
		opts := ServiceOptions{PassHostHeader: &passHost}
		assert.False(t, opts.IsPassHostHeader())
	})
}

func TestIsWatch(t *testing.T) {
	t.Run("watch is nil returns true", func(t *testing.T) {
		opts := Options{Watch: nil}
		assert.True(t, opts.IsWatch())
	})

	t.Run("watch is true returns true", func(t *testing.T) {
		watch := true
		opts := Options{Watch: &watch}
		assert.True(t, opts.IsWatch())
	})

	t.Run("watch is false returns false", func(t *testing.T) {
		watch := false
		opts := Options{Watch: &watch}
		assert.False(t, opts.IsWatch())
	})
}

func TestNewOptions(t *testing.T) {
	t.Run("creates options with empty maps", func(t *testing.T) {
		opts := NewOptions()
		assert.NotNil(t, opts.AccessLogs)
		assert.NotNil(t, opts.Servers)
		assert.NotNil(t, opts.Routes)
		assert.NotNil(t, opts.Middlewares)
		assert.NotNil(t, opts.Services)
		assert.NotNil(t, opts.Upstreams)
	})
}
