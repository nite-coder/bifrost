package config

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
	return "invalid config: " + e.Message
}
