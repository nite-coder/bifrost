package middleware

import (
	"fmt"

	"github.com/cloudwego/hertz/pkg/app"
)

var (
	handlers map[string]CreateMiddlewareHandler = make(map[string]CreateMiddlewareHandler)
)

type CreateMiddlewareHandler func(param map[string]any) (app.HandlerFunc, error)

func RegisterMiddleware(kind string, handler CreateMiddlewareHandler) error {

	if _, found := handlers[kind]; found {
		return fmt.Errorf("middleware handler '%s' already exists", kind)
	}

	handlers[kind] = handler
	return nil
}

func FindHandlerByType(kind string) CreateMiddlewareHandler {
	return handlers[kind]
}
