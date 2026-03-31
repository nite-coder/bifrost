package config

// InvalidConfigError represents an error in the configuration.
type InvalidConfigError struct {
	Value     any
	Message   string
	Structure []string
}

func newInvalidConfig(structure []string, value any, message string) InvalidConfigError {
	return InvalidConfigError{
		Structure: structure,
		Value:     value,
		Message:   message,
	}
}

func (e InvalidConfigError) Error() string {
	return e.Message
}
