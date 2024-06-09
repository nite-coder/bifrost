package gateway

import (
	"context"
	"fmt"
	"http-benchmark/pkg/domain"
	"http-benchmark/pkg/log"
	"http-benchmark/pkg/middleware/addprefix"
	"http-benchmark/pkg/middleware/replacepath"
	"http-benchmark/pkg/middleware/replacepathregex"
	"http-benchmark/pkg/middleware/stripprefix"
	"http-benchmark/pkg/middleware/timinglogger"
	"log/slog"

	"github.com/cloudwego/hertz/pkg/app"
)

type initMiddleware struct {
	logger  *slog.Logger
	entryID string
}

func newInitMiddleware(entryID string, logger *slog.Logger) *initMiddleware {
	return &initMiddleware{
		logger:  logger,
		entryID: entryID,
	}
}

func (m *initMiddleware) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	if len(ctx.Request.Header.Get("X-Forwarded-For")) > 0 {
		ctx.Set("X-Forwarded-For", ctx.Request.Header.Get("X-Forwarded-For"))
	}

	ctx.Set(domain.ENTRY_ID, m.entryID)

	c = log.NewContext(c, m.logger)
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
		prefix := params["prefix"].(string)
		m := addprefix.NewMiddleware(prefix)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("replace_path", func(params map[string]any) (app.HandlerFunc, error) {
		newPath := params["path"].(string)
		m := replacepath.NewMiddleware(newPath)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("replace_path_regex", func(params map[string]any) (app.HandlerFunc, error) {
		regex := params["regex"].(string)
		replacement := params["replacement"].(string)
		m := replacepathregex.NewMiddleware(regex, replacement)
		return m.ServeHTTP, nil
	})

	_ = RegisterMiddleware("timing_logger", func(param map[string]any) (app.HandlerFunc, error) {
		m := timinglogger.NewMiddleware()
		return m.ServeHTTP, nil
	})
}
