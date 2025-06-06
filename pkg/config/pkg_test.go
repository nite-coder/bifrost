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
		Providers: ProviderOtions{
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
