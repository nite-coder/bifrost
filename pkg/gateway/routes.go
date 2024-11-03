package gateway

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/nite-coder/bifrost/pkg/config"
	"github.com/nite-coder/bifrost/pkg/middleware"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/nite-coder/blackbear/pkg/cast"
)

type routeSetting struct {
	regex      *regexp.Regexp
	route      *config.RouteOptions
	middleware []app.HandlerFunc
}

func loadRoutes(bifrost *Bifrost, server config.ServerOptions, services map[string]*Service, middlewares map[string]app.HandlerFunc) (*Routes, error) {
	route := newRoutes()

	for routeID, routeOpts := range bifrost.options.Routes {

		routeOpts.ID = routeID

		if len(routeOpts.Servers) > 0 && !slices.Contains(routeOpts.Servers, server.ID) {
			continue
		}

		if len(routeOpts.Paths) == 0 {
			return nil, fmt.Errorf("paths can't be empty in route: '%s'", routeOpts.ID)
		}

		if len(routeOpts.ServiceID) == 0 {
			return nil, fmt.Errorf("service_id can't be empty in route: '%s'", routeOpts.ID)
		}

		routeMiddlewares := make([]app.HandlerFunc, 0)

		for _, m := range routeOpts.Middlewares {
			if len(m.Use) > 0 {
				val, found := middlewares[m.Use]
				if !found {
					return nil, fmt.Errorf("middleware '%s' was not found in route id: '%s'", m.Use, routeOpts.ID)
				}

				routeMiddlewares = append(routeMiddlewares, val)
				continue
			}

			if len(m.Type) == 0 {
				return nil, fmt.Errorf("middleware kind can't be empty in route: '%s'", routeOpts.Paths)
			}

			handler := middleware.FindHandlerByType(m.Type)
			if handler == nil {
				return nil, fmt.Errorf("middleware handler '%s' was not found in route: '%s'", m.Type, routeOpts.Paths)
			}

			appHandler, err := handler(m.Params)
			if err != nil {
				return nil, fmt.Errorf("create middleware handler '%s' failed in route: '%s'", m.Type, routeOpts.Paths)
			}

			routeMiddlewares = append(routeMiddlewares, appHandler)
		}

		// dynamic service
		if routeOpts.ServiceID[0] == '$' {
			dynamicService := newDynamicService(routeOpts.ServiceID, services)
			routeMiddlewares = append(routeMiddlewares, dynamicService.ServeHTTP)
		} else {
			service, found := services[routeOpts.ServiceID]
			if !found {
				return nil, fmt.Errorf("service_id '%s' was not found in route: %s", routeOpts.ServiceID, routeOpts.ID)
			}
			routeMiddlewares = append(routeMiddlewares, service.ServeHTTP)
		}

		err := route.Add(routeOpts, routeMiddlewares...)
		if err != nil {
			return nil, err
		}
	}

	return route, nil
}

type Routes struct {
	router       *Router
	regexpRoutes []routeSetting
}

func newRoutes() *Routes {
	return &Routes{
		router:       newRouter(),
		regexpRoutes: make([]routeSetting, 0),
	}
}

// ServeHTTP implements the http.Handler interface
func (r *Routes) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	method := cast.B2S(ctx.Method())
	path := cast.B2S(ctx.Request.Path())

	middleware, isDefered := r.router.find(method, path)

	if len(middleware) > 0 && !isDefered {
		ctx.SetIndex(-1)
		ctx.SetHandlers(middleware)
		ctx.Next(c)
		ctx.Abort()
		return
	}

	// regexp routes
	for _, route := range r.regexpRoutes {
		if checkRegexpRoute(route, method, path) {
			ctx.SetIndex(-1)
			ctx.SetHandlers(route.middleware)
			ctx.Next(c)
			ctx.Abort()
			return
		}
	}

	// general routes
	if len(middleware) > 0 {
		ctx.SetIndex(-1)
		ctx.SetHandlers(middleware)
		ctx.Next(c)
		ctx.Abort()
		return
	}

}

// Add adds a new route
func (r *Routes) Add(routeOpts config.RouteOptions, middlewares ...app.HandlerFunc) error {
	var err error

	// validate
	if len(routeOpts.Paths) == 0 {
		return errors.New("paths can't be empty")
	}

	for _, path := range routeOpts.Paths {
		path = strings.TrimSpace(path)
		var nodeType nodeType

		switch {
		case strings.HasPrefix(path, "~*"):
			expr := strings.TrimSpace(path[2:])
			if len(expr) == 0 {
				return fmt.Errorf("router: regexp expression route can't be empty in route: '%s'", routeOpts.ID)
			}
			regx, err := regexp.Compile(`(?i)` + expr)
			if err != nil {
				return err
			}

			r.regexpRoutes = append(r.regexpRoutes, routeSetting{
				regex:      regx,
				route:      &routeOpts,
				middleware: middlewares,
			})
			continue
		case strings.HasPrefix(path, "~"):
			expr := strings.TrimSpace(path[1:])
			if len(expr) == 0 {
				return fmt.Errorf("router: regexp expression route can't be empty in route: '%s'", routeOpts.ID)
			}
			regx, err := regexp.Compile(expr)
			if err != nil {
				return err
			}

			r.regexpRoutes = append(r.regexpRoutes, routeSetting{
				regex:      regx,
				route:      &routeOpts,
				middleware: middlewares,
			})
			continue
		case strings.HasPrefix(path, "="):
			nodeType = nodeTypeExact
			path = strings.TrimSpace(path[1:])
			if len(path) == 0 {
				return fmt.Errorf("router: exact route can't be empty in route: '%s'", routeOpts.ID)
			}
		case strings.HasPrefix(path, "^~"):
			nodeType = nodeTypePrefix
			path = strings.TrimSpace(path[2:])
			if len(path) == 0 {
				return fmt.Errorf("router: prefix route can't be empty in route: '%s'", routeOpts.ID)
			}

		default:
			if !strings.HasPrefix(path, "/") {
				return fmt.Errorf("router: '%s' is invalid path. Path needs to begin with '/'", path)
			}
			nodeType = nodeTypeGeneral
		}

		if len(routeOpts.Methods) == 0 {
			for _, method := range httpMethods {
				err = r.router.add(method, path, nodeType, middlewares...)
				if err != nil {
					return err
				}
			}
		}

		for _, method := range routeOpts.Methods {
			method := strings.ToUpper(method)
			if !isValidHTTPMethod(method) {
				return fmt.Errorf("http method %s is not valid", method)
			}

			err = r.router.add(method, path, nodeType, middlewares...)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func checkRegexpRoute(setting routeSetting, method string, path string) bool {
	if len(setting.route.Methods) > 0 {
		isMethodFound := false

		for _, m := range setting.route.Methods {
			if m == method {
				return true
			}
		}

		if !isMethodFound {
			return false
		}
	}

	return setting.regex.MatchString(path)
}

// nolint
func getHost(ctx *app.RequestContext) func() string {
	var (
		host string
		once sync.Once
	)

	return func() string {
		once.Do(func() {
			host = string(ctx.Request.Host())
		})
		return host
	}
}
