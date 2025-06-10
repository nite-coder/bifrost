package config

type ErrInvalidConfig struct {
	Value     any
	Message   string
	Structure []string
}

func newInvalidConfig(structure []string, value any, message string) ErrInvalidConfig {
	return ErrInvalidConfig{
		Structure: structure,
		Value:     value,
		Message:   message,
	}
}
func (e ErrInvalidConfig) Error() string {
	return e.Message
}
