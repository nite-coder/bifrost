package config

type ErrInvalidConfig struct {
	Structure []string
	Value     any
	Message   string
}

func newInvalidConfig(structure []string, value any, message string) ErrInvalidConfig {
	return ErrInvalidConfig{
		Structure: structure,
		Value:     value,
		Message:   message,
	}
}

func (e ErrInvalidConfig) Error() string {
	return "invalid config: " + e.Message
}
