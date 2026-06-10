package middleware

import (
	"errors"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
)

var handlers = make(map[string]CreateMiddlewareHandler)

// CreateMiddlewareHandler is a function that creates an app.HandlerFunc from parameters.
type CreateMiddlewareHandler func(param any) (app.HandlerFunc, error)

// Factory returns a middleware creator for the given kind.
func Factory(kind string) CreateMiddlewareHandler {
	return handlers[kind]
}

// Register registers a middleware with a strongly typed config struct.
// It automatically handles the decoding of generic params (map[string]any) into struct T.
// Note: Fields in struct T should be tagged with `mapstructure` tags (e.g., `mapstructure:"prefix"`)
// to ensure parameters are correctly mapped and decoded from the configuration.
func Register[T any](names []string, handler func(T) (app.HandlerFunc, error)) error {
	if len(names) == 0 {
		return errors.New("middleware names cannot be empty")
	}

	wrappedHandler := func(params any) (app.HandlerFunc, error) {
		if typedParams, ok := params.(T); ok {
			return handler(typedParams)
		}

		var cfg T
		if params == nil {
			// If params is nil, we pass the zero value of T
			// If T is a pointer type, it will be nil. If T is a struct, it will be empty struct.
			return handler(cfg)
		}

		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Result:           &cfg,
			TagName:          "mapstructure",
			WeaklyTypedInput: true,
			DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create decoder: %w", err)
		}

		if err := decoder.Decode(params); err != nil {
			return nil, fmt.Errorf("failed to decode middleware params: %w", err)
		}

		return handler(cfg)
	}

	for _, name := range names {
		if _, found := handlers[name]; found {
			return fmt.Errorf("middleware handler '%s' already exists", name)
		}

		handlers[name] = wrappedHandler
	}

	return nil
}
