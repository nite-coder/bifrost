package parallel

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/internal/pkg/safety"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/middleware"
)

type ParallelMiddleware struct {
	options []*Options
}
type Options struct {
	Middleware        app.HandlerFunc
	MiddlewareOptions config.MiddlwareOptions
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
		go safety.Go(ctx, func() {
			defer func() {
				if r := recover(); r != nil {
					// convert panic to an error
					err := fmt.Errorf("parallel middleware panic occurred: %v", r)
					_ = c.Error(err)
				}
				waitGroup.Done()
			}()
			option.Middleware(ctx, c)
		})
	}
	waitGroup.Wait()
	if len(c.Errors) > 0 {
		c.Abort()
	}
}
func Init() error {
	return middleware.RegisterTyped([]string{"parallel"}, func(middlewareOptions []*config.MiddlwareOptions) (app.HandlerFunc, error) {
		if len(middlewareOptions) == 0 {
			return nil, errors.New("parallel middleware params is empty or invalid")
		}

		options := make([]*Options, 0)
		for _, middlewareOption := range middlewareOptions {
			h := middleware.Factory(middlewareOption.Type)
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
