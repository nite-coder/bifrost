package parallel

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/go-viper/mapstructure/v2"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

type ParallelMiddleware struct {
	options []*Options
}

type Options struct {
	MiddlewareOptions config.MiddlwareOptions
	Middleware        app.HandlerFunc
}

func NewMiddleware(options []*Options) *ParallelMiddleware {
	return &ParallelMiddleware{
		options: options,
	}
}

func (m *ParallelMiddleware) ServeHTTP(ctx context.Context, c *app.RequestContext) {

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(m.options))

	for _, option := range m.options {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// convert panic to an error
					err := fmt.Errorf("parallel middleware panic occurred: %v", r)
					_ = c.Error(err)
				}
				waitGroup.Done()
			}()

			option.Middleware(ctx, c)
		}()
	}

	waitGroup.Wait()

	if len(c.Errors) > 0 {
		c.Abort()
	}
}

func init() {
	_ = middleware.RegisterMiddleware("parallel", func(params any) (app.HandlerFunc, error) {
		if params == nil {
			return nil, errors.New("parallel middleware params is empty or invalid")
		}

		middlewareOptions := []*config.MiddlwareOptions{}

		paramsSlice, ok := params.([]interface{})
		if !ok {
			return nil, errors.New("parallel middleware params is invalid")
		}

		err := mapstructure.Decode(paramsSlice, &middlewareOptions)
		if err != nil {
			return nil, fmt.Errorf("parallel middleware params is invalid, error: %w", err)
		}

		options := make([]*Options, 0)
		for _, middlewareOption := range middlewareOptions {
			h := middleware.FindHandlerByType(middlewareOption.Type)

			m, err := h(middlewareOption.Params)
			if err != nil {
				return nil, fmt.Errorf("%s middleware params is invalid in parallel middleware, error: %w", middlewareOption.Type, err)
			}

			options = append(options, &Options{
				MiddlewareOptions: *middlewareOption,
				Middleware:        m,
			})
		}

		m := NewMiddleware(options)
		return m.ServeHTTP, nil
	})
}
