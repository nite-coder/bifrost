package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigLoad(t *testing.T) {
	configPath := "./../../test/config/min.yaml"

	_, err := Load(configPath)
	assert.NoError(t, err)

	configPath = "./../../test/config/good.yaml"

	mainOptions, err := Load(configPath)
	assert.NoError(t, err)
	assert.Len(t, mainOptions.Middlewares, 1)
}

func TestConfigFailCheck(t *testing.T) {
	configPath := "./../../test/config/wrong_access_log.yaml"
	_, err := Load(configPath)
	assert.Error(t, err)

	configPath = "./../../test/config/duplicate_upstream.yaml"
	_, err = Load(configPath)
	assert.Error(t, err)
}
