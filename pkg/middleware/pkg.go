package middleware

import (
	"errors"
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
)

var (
	handlers map[string]CreateMiddlewareHandler = make(map[string]CreateMiddlewareHandler)
)

type CreateMiddlewareHandler func(param any) (app.HandlerFunc, error)

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
