package log

import (
	"testing"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestLogging(t *testing.T) {
	options := config.LoggingOtions{
		Level:   "",
		Handler: "json",
		Output:  "",
	}

	logger, err := NewLogger(options)
	assert.NoError(t, err)

	logger.Info("test")
}
