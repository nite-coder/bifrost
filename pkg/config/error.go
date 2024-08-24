package config

import "fmt"

type ErrInvalidConfig struct {
	FullPath []string
	Value    any
	Message  string
}

func newInvalidConfig(fullPath []string, value any, message string) ErrInvalidConfig {
	return ErrInvalidConfig{
		FullPath: fullPath,
		Value:    value,
		Message:  message,
	}
}

func (e ErrInvalidConfig) Error() string {
	return fmt.Sprintf("invalid config: %s", e.Message)
}
