package gateway

import (
	"context"
	"fmt"
	"http-benchmark/pkg/config"
	"http-benchmark/pkg/log"
	"http-benchmark/pkg/middleware/addprefix"
	"http-benchmark/pkg/middleware/headers"
	"http-benchmark/pkg/middleware/prommetric"
	"http-benchmark/pkg/middleware/replacepath"
	"http-benchmark/pkg/middleware/replacepathregex"
	"http-benchmark/pkg/middleware/stripprefix"
	"http-benchmark/pkg/middleware/timinglogger"
	"log/slog"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/blackbear/pkg/cast"
	"go.opentelemetry.io/otel/trace"
)

type initMiddleware struct {
	logger   *slog.Logger
	serverID string
}

func newInitMiddleware(serverID string, logger *slog.Logger) *initMiddleware {
	return &initMiddleware{
		logger:   logger,
		serverID: serverID,
	}
}

func (m *initMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	logger := m.logger

	if len(ctx.Request.Header.Get("X-Forwarded-For")) > 0 {
		ctx.Set("X-Forwarded-For", ctx.Request.Header.Get("X-Forwarded-For"))
	}

	spanCtx := trace.SpanContextFromContext(c)
	if spanCtx.HasTraceID() {
		traceID := spanCtx.TraceID().String()
		ctx.Set(config.TRACE_ID, traceID)

		logger = logger.With(slog.String("trace_id", traceID))
	}

	ctx.Set(config.SERVER_ID, m.serverID)

	c = log.NewContext(c, logger)
	ctx.Next(c)
}

type CreateMiddlewareHandler func(param map[string]any) (app.HandlerFunc, error)

var middlewareFactory map[string]CreateMiddlewareHandler = make(map[string]CreateMiddlewareHandler)

func RegisterMiddleware(kind string, handler CreateMiddlewareHandler) error {

	if _, found := middlewareFactory[kind]; found {
		return fmt.Errorf("middleware handler '%s' already exists", kind)
	}

	middlewareFactory[kind] = handler

	return nil
}

func loadMiddlewares(opts map[string]config.MiddlwareOptions) (map[string]app.HandlerFunc, error) {

	middlewares := map[string]app.HandlerFunc{}
	for id, middlewareOpts := range opts {

		if len(id) == 0 {
			return nil, fmt.Errorf("middleware id can't be empty")
		}

		middlewareOpts.ID = id

		if len(middlewareOpts.Type) == 0 {
			return nil, fmt.Errorf("middleware type can't be empty")
		}

		handler, found := middlewareFactory[middlewareOpts.Type]
		if !found {
			return nil, fmt.Errorf("middleware type '%s' was not found", middlewareOpts.Type)
		}

		m, err := handler(middlewareOpts.Params)
		if err != nil {
			return nil, fmt.Errorf("middleware type '%s' params is invalid. error: %w", middlewareOpts.Type, err)
		}

		middlewares[middlewareOpts.ID] = m
	}

	return middlewares, nil
}

func init() {
	_ = RegisterMiddleware("strip_prefix", func(params map[string]any) (app.HandlerFunc, error) {
		val := params["prefixes"].([]any)

		prefixes := make([]string, 0)
		for _, v := range val {
			prefixes = append(prefixes, v.(string))
		}

		m := stripprefix.NewMiddleware(prefixes)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("add_prefix", func(params map[string]any) (app.HandlerFunc, error) {
		prefix, ok := params["prefix"].(string)
		if !ok {
			return nil, fmt.Errorf("prefix is not set or prefix is invalid")
		}
		m := addprefix.NewMiddleware(prefix)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("replace_path", func(params map[string]any) (app.HandlerFunc, error) {
		newPath, ok := params["path"].(string)
		if !ok {
			return nil, fmt.Errorf("path is not set or path is invalid")
		}
		m := replacepath.NewMiddleware(newPath)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("replace_path_regex", func(params map[string]any) (app.HandlerFunc, error) {
		regex, ok := params["regex"].(string)
		if !ok {
			return nil, fmt.Errorf("regex is not set or regex is invalid")
		}
		replacement, ok := params["replacement"].(string)
		if !ok {
			return nil, fmt.Errorf("replacement is not set or replacement is invalid")
		}
		m := replacepathregex.NewMiddleware(regex, replacement)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("prom_metric", func(param map[string]any) (app.HandlerFunc, error) {
		path, ok := param["path"].(string)
		if !ok {
			return nil, fmt.Errorf("path is not set or path is invalid")
		}

		m := prommetric.New(path)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("headers", func(param map[string]any) (app.HandlerFunc, error) {
		requestHeader := map[string]string{}
		val, found := param["request_headers"]
		if found {
			headers, ok := val.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("request_headers is not set or request_headers is invalid")
			}

			for k, v := range headers {
				val, err := cast.ToString(v)
				if err != nil {
					continue
				}
				requestHeader[k] = val
			}
		}

		respHeader := map[string]string{}
		val, found = param["response_headers"]
		if found {
			headers, ok := val.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("response_headers is not set or response_headers is invalid")
			}

			for k, v := range headers {
				val, err := cast.ToString(v)
				if err != nil {
					continue
				}
				respHeader[k] = val
			}
		}

		m := headers.NewMiddleware(requestHeader, respHeader)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("timing_logger", func(param map[string]any) (app.HandlerFunc, error) {
		m := timinglogger.NewMiddleware()
		return m.ServeHTTP, nil
	})

}
