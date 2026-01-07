package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigLoad(t *testing.T) {
	configPath := "./../../test/config/min.yaml"

	_, err := Load(configPath)
	assert.NoError(t, err)

	t.Run("routes order", func(t *testing.T) {
		configPath := "./../../test/config/good.yaml"

		mainOptions, err := Load(configPath)
		assert.NoError(t, err)

		assert.Equal(t, "all_routes1", mainOptions.Routes[1].ID)
		assert.Equal(t, "all_routes2", mainOptions.Routes[2].ID)
		assert.Equal(t, "all_routes3", mainOptions.Routes[3].ID)
	})
}

func TestConfigFailCheck(t *testing.T) {
	configPath := "./../../test/config/wrong_access_log.yaml"
	_, err := Load(configPath)
	assert.Error(t, err)

	configPath = "./../../test/config/duplicate_upstream.yaml"
	_, err = Load(configPath)
	assert.Error(t, err)
}

func TestConfigAfterDynamicLoad(t *testing.T) {
	configPath := "./../../test/config/load_after_dynamic.yaml"
	_, err := Load(configPath)

	assert.Error(t, err)
}

func TestDomainName(t *testing.T) {
	testDomains := []string{
		"example.com",
		"sub-domain.example.co.uk",
		"invalid.com",
		"valid123.com",
	}

	for _, domain := range testDomains {
		result := IsValidDomain(domain)
		assert.True(t, result)
	}

	testDomains = []string{
		"invalid-bb",
		"-invalid.com",
		"toolong" + "toolong" + "toolong" + "toolong" + "toolong" + "toolong", // Exceeds 253 characters
		"valid123_aa",
	}

	for _, domain := range testDomains {
		result := IsValidDomain(domain)
		assert.False(t, result, domain+" should be false")
	}
}

func TestDefaultPath(t *testing.T) {
	configPath := "./../../test/config/min.yaml"
	_, err := defaultPath(configPath)
	assert.NoError(t, err)

	configPath = ""
	_, err = defaultPath(configPath)
	assert.Error(t, err, "config file not found")
}

func TestDynamicProvider(t *testing.T) {
	watch := false

	mainOptions := Options{
		Providers: ProviderOptions{
			Nacos: NacosProviderOptions{
				Config: NacosConfigOptions{
					Enabled: true,
					Endpoints: []string{
						"http://10.1.1.1:8848",
					},
					Watch: &watch,
				},
			},
		},
	}

	providers, options, err := loadDynamic(mainOptions)
	assert.NoError(t, err)
	assert.Equal(t, options, mainOptions)
	assert.Equal(t, len(providers), 1)
}

func TestMergeOptions(t *testing.T) {
	t.Run("merge middlewares", func(t *testing.T) {
		mainOpts := NewOptions()
		content := `middlewares:
  rate_limit:
    type: rate_limit
`
		result, err := mergeOptions(mainOpts, content)
		assert.NoError(t, err)
		assert.Contains(t, result.Middlewares, "rate_limit")
	})

	t.Run("duplicate middleware error", func(t *testing.T) {
		mainOpts := NewOptions()
		mainOpts.Middlewares["rate_limit"] = MiddlwareOptions{}
		content := `middlewares:
  rate_limit:
    type: rate_limit
`
		_, err := mergeOptions(mainOpts, content)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicated")
	})

	t.Run("merge servers", func(t *testing.T) {
		mainOpts := NewOptions()
		content := `servers:
  web:
    bind: ":8080"
`
		result, err := mergeOptions(mainOpts, content)
		assert.NoError(t, err)
		assert.Contains(t, result.Servers, "web")
	})

	t.Run("duplicate server error", func(t *testing.T) {
		mainOpts := NewOptions()
		mainOpts.Servers["web"] = ServerOptions{}
		content := `servers:
  web:
    bind: ":8080"
`
		_, err := mergeOptions(mainOpts, content)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicated")
	})

	t.Run("merge services", func(t *testing.T) {
		mainOpts := NewOptions()
		content := `services:
  backend:
    url: http://localhost:8080
`
		result, err := mergeOptions(mainOpts, content)
		assert.NoError(t, err)
		assert.Contains(t, result.Services, "backend")
	})

	t.Run("duplicate service error", func(t *testing.T) {
		mainOpts := NewOptions()
		mainOpts.Services["backend"] = ServiceOptions{}
		content := `services:
  backend:
    url: http://localhost:8080
`
		_, err := mergeOptions(mainOpts, content)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicated")
	})

	t.Run("merge upstreams", func(t *testing.T) {
		mainOpts := NewOptions()
		content := `upstreams:
  api:
    targets:
      - target: localhost:8080
`
		result, err := mergeOptions(mainOpts, content)
		assert.NoError(t, err)
		assert.Contains(t, result.Upstreams, "api")
	})

	t.Run("duplicate upstream error", func(t *testing.T) {
		mainOpts := NewOptions()
		mainOpts.Upstreams["api"] = UpstreamOptions{}
		content := `upstreams:
  api:
    targets:
      - target: localhost:8080
`
		_, err := mergeOptions(mainOpts, content)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicated")
	})

	t.Run("invalid yaml error", func(t *testing.T) {
		mainOpts := NewOptions()
		content := `invalid: yaml: content: [`
		_, err := mergeOptions(mainOpts, content)
		assert.Error(t, err)
	})

	t.Run("merge with nil maps", func(t *testing.T) {
		mainOpts := Options{}
		content := `middlewares:
  test:
    type: test
`
		result, err := mergeOptions(mainOpts, content)
		assert.NoError(t, err)
		assert.NotNil(t, result.Middlewares)
	})
}

func TestWatch(t *testing.T) {
	t.Run("watch with nil providers", func(t *testing.T) {
		mainProvider = nil
		dynamicProviders = nil
		err := Watch()
		assert.NoError(t, err)
	})
}

func TestTestAndSkipResovler(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		configPath := "./../../test/config/min.yaml"
		path, err := TestAndSkipResovler(configPath)
		assert.NoError(t, err)
		assert.Contains(t, path, "min.yaml")
	})

	t.Run("invalid config path", func(t *testing.T) {
		configPath := "./../../test/config/nonexistent.yaml"
		_, err := TestAndSkipResovler(configPath)
		assert.Error(t, err)
	})
}

func TestFindConfigurationLine(t *testing.T) {
	t.Run("json content", func(t *testing.T) {
		content := `{"logging": {"level": "debug"}}`
		path := []string{"logging", "level"}
		line := findConfigurationLine(content, path, "debug")
		assert.Equal(t, -1, line) // JSON doesn't have line concept like YAML
	})

	t.Run("yaml content", func(t *testing.T) {
		content := `logging:
  level: debug
  handler: text
`
		path := []string{"logging", "level"}
		line := findConfigurationLine(content, path, "debug")
		assert.GreaterOrEqual(t, line, 1)
	})

	t.Run("invalid content", func(t *testing.T) {
		content := `{{{invalid`
		path := []string{"key"}
		line := findConfigurationLine(content, path, "value")
		assert.Equal(t, -1, line)
	})
}
