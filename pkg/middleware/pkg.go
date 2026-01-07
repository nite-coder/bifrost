package middleware

import (
	"errors"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
)

var (
	handlers map[string]CreateMiddlewareHandler = make(map[string]CreateMiddlewareHandler)
)

type CreateMiddlewareHandler func(param any) (app.HandlerFunc, error)

// Register registers a legacy middleware handler.
//
// Deprecated: use RegisterTyped instead.
func Register(names []string, handler CreateMiddlewareHandler) error {
	if len(names) == 0 {
		return errors.New("middleware names cannot be empty")
	}

	for _, name := range names {
		if _, found := handlers[name]; found {
			return fmt.Errorf("middleware handler '%s' already exists", name)
		}

		handlers[name] = handler
	}

	return nil
}

func Factory(kind string) CreateMiddlewareHandler {
	return handlers[kind]
}

// RegisterTyped registers a middleware with a strongly typed config struct.
// It automatically handles the decoding of generic params (map[string]any) into struct T.
func RegisterTyped[T any](names []string, handler func(T) (app.HandlerFunc, error)) error {
	return Register(names, func(params any) (app.HandlerFunc, error) {
		var cfg T
		if params == nil {
			// If params is nil, we pass the zero value of T
			// If T is a pointer type, it will be nil. If T is a struct, it will be empty struct.
			return handler(cfg)
		}

		// Check if T is a pointer and params is nil? No, params is any.

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
	})
}
