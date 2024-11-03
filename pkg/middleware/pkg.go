package middleware

import (
	"errors"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/middleware/addprefix"
	"github.com/nite-coder/bifrost/pkg/middleware/headers"
	"github.com/nite-coder/bifrost/pkg/middleware/prommetric"
	"github.com/nite-coder/bifrost/pkg/middleware/ratelimiting"
	"github.com/nite-coder/bifrost/pkg/middleware/replacepath"
	"github.com/nite-coder/bifrost/pkg/middleware/replacepathregex"
	"github.com/nite-coder/bifrost/pkg/middleware/response"
	"github.com/nite-coder/bifrost/pkg/middleware/stripprefix"
	"github.com/nite-coder/bifrost/pkg/middleware/timinglogger"
	"github.com/nite-coder/bifrost/pkg/middleware/tracing"
	"github.com/nite-coder/blackbear/pkg/cast"
)

var (
	middlewareFactory map[string]CreateMiddlewareHandler = make(map[string]CreateMiddlewareHandler)
)

type CreateMiddlewareHandler func(param map[string]any) (app.HandlerFunc, error)

func RegisterMiddleware(kind string, handler CreateMiddlewareHandler) error {

	if _, found := middlewareFactory[kind]; found {
		return fmt.Errorf("middleware handler '%s' already exists", kind)
	}

	middlewareFactory[kind] = handler

	return nil
}

func FindHandlerByType(kind string) CreateMiddlewareHandler {
	return middlewareFactory[kind]
}

func LoadMiddlewares(middlewareOptions map[string]config.MiddlwareOptions) (map[string]app.HandlerFunc, error) {

	middlewares := map[string]app.HandlerFunc{}
	for id, middlewareOpts := range middlewareOptions {

		if len(id) == 0 {
			return nil, errors.New("middleware id can't be empty")
		}

		middlewareOpts.ID = id

		if len(middlewareOpts.Type) == 0 {
			return nil, errors.New("middleware type can't be empty")
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
			return nil, errors.New("prefix is not set or prefix is invalid")
		}
		m := addprefix.NewMiddleware(prefix)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("replace_path", func(params map[string]any) (app.HandlerFunc, error) {
		newPath, ok := params["path"].(string)
		if !ok {
			return nil, errors.New("path is not set or path is invalid")
		}
		m := replacepath.NewMiddleware(newPath)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("replace_path_regex", func(params map[string]any) (app.HandlerFunc, error) {
		regex, ok := params["regex"].(string)
		if !ok {
			return nil, errors.New("regex is not set or regex is invalid")
		}
		replacement, ok := params["replacement"].(string)
		if !ok {
			return nil, errors.New("replacement is not set or replacement is invalid")
		}
		m := replacepathregex.NewMiddleware(regex, replacement)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("prom_metric", func(param map[string]any) (app.HandlerFunc, error) {
		path, ok := param["path"].(string)
		if !ok {
			return nil, errors.New("path is not set or path is invalid")
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
				return nil, errors.New("request_headers is not set or request_headers is invalid")
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
				return nil, errors.New("response_headers is not set or response_headers is invalid")
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

	_ = RegisterMiddleware("tracing", func(param map[string]any) (app.HandlerFunc, error) {
		m := tracing.NewMiddleware()
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("response", func(param map[string]any) (app.HandlerFunc, error) {
		options := response.ResponseOptions{}
		val, found := param["status_code"]
		if found {
			statusCode, err := cast.ToInt(val)
			if err != nil {
				return nil, errors.New("status_code is invalid in response middleware")
			}
			options.StatusCode = statusCode
		}

		val, found = param["content_type"]
		if found {
			contentType, err := cast.ToString(val)
			if err != nil {
				return nil, errors.New("content_type is invalid in response middleware")
			}
			options.ContentType = contentType
		}

		val, found = param["content"]
		if found {
			content, err := cast.ToString(val)
			if err != nil {
				return nil, errors.New("content is invalid in response middleware")
			}
			options.Content = content
		}

		m := response.NewMiddleware(options)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("rate-limiting", func(param map[string]any) (app.HandlerFunc, error) {
		option := ratelimiting.Options{}

		// strategy
		strategyVal, found := param["strategy"]
		if !found {
			return nil, errors.New("strategy is not found in rate-limiting middleware")
		}
		strategy, err := cast.ToString(strategyVal)
		if err != nil {
			return nil, errors.New("strategy is invalid in rate-limiting middleware")
		}
		option.Strategy = ratelimiting.StrategyMode(strategy)

		// limit
		limitVal, found := param["limit"]
		if !found {
			return nil, errors.New("limit is not found in rate-limiting middleware1")
		}
		limit, err := cast.ToUint64(limitVal)
		if err != nil {
			return nil, errors.New("limit is invalid in rate-limiting middleware")
		}
		option.Limit = limit

		// limit_by
		limitByVal, found := param["limit_by"]
		if !found {
			return nil, errors.New("limit_by is not found in rate-limiting middleware")
		}
		limitBy, err := cast.ToString(limitByVal)
		if err != nil {
			return nil, errors.New("limit_by is invalid in rate-limiting middleware")
		}
		option.LimitBy = limitBy

		// window_size
		windowSizeVal, found := param["window_size"]
		if !found {
			return nil, errors.New("window_size is not found in rate-limiting middleware")
		}
		s, _ := cast.ToString(windowSizeVal)
		windowSize, err := time.ParseDuration(s)
		if err != nil {
			return nil, errors.New("window_size is invalid in rate-limiting middleware")
		}
		option.WindowSize = windowSize

		// http status
		statusVal, found := param["http_status"]
		if found {
			status, err := cast.ToInt(statusVal)
			if err != nil {
				return nil, errors.New("http_status is invalid in rate-limiting middleware")
			}
			option.HTTPStatus = status
		}

		// http content type
		contentTypeVal, found := param["http_content_type"]
		if found {
			contentType, err := cast.ToString(contentTypeVal)
			if err != nil {
				return nil, errors.New("http_content_type is invalid in rate-limiting middleware")
			}
			option.HTTPContentType = contentType
		}

		// http body
		bodyVal, found := param["http_response_body"]
		if found {
			body, err := cast.ToString(bodyVal)
			if err != nil {
				return nil, errors.New("http_response_body is invalid in rate-limiting middleware")
			}
			option.HTTPResponseBody = body
		}

		// redis id
		if option.Strategy == ratelimiting.Redis {
			redisIDVal, found := param["redis_id"]
			if found {
				redisID, err := cast.ToString(redisIDVal)
				if err != nil {
					return nil, errors.New("redis_id is invalid in rate-limiting middleware")
				}
				option.RedisID = redisID
			} else {
				return nil, errors.New("redis_id is not found in rate-limiting middleware")
			}
		}

		m, err := ratelimiting.NewMiddleware(option)
		if err != nil {
			return nil, err
		}
		return m.ServeHTTP, nil
	})

}
