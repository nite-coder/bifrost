package gateway

import (
	"context"
	"errors"
	"fmt"
	"http-benchmark/pkg/config"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
)

// node represents a node in the Trie
type node struct {
	path            string           // Path name of the node
	children        map[string]*node // Child nodes, indexed by path name
	handler         *methodHandler   // Handler functions
	prefixChildren  map[string]*node
	generalChildren map[string]*node
}

type nodeType int32

const (
	nodeTypeExact nodeType = iota
	nodeTypePrefix
	nodeTypeGeneral
	nodeTypeRegex
)

type routeSetting struct {
	regex      *regexp.Regexp
	route      *config.RouteOptions
	middleware []app.HandlerFunc
}

// methodHandler contains handler functions for various HTTP methods
type methodHandler struct {
	handlers map[string][]app.HandlerFunc // Associates HTTP methods with handler functions
}

// Router struct contains the Trie and handler chain
type Router struct {
	tree         *node // Root node of the Trie
	regexpRoutes []routeSetting
}

func loadRouter(bifrost *Bifrost, entry config.EntryOptions, services map[string]*Service, middlewares map[string]app.HandlerFunc) (*Router, error) {
	router := newRouter()

	for routeID, routeOpts := range bifrost.opts.Routes {

		routeOpts.ID = routeID

		if len(routeOpts.Entries) > 0 && !slices.Contains(routeOpts.Entries, entry.ID) {
			continue
		}

		if len(routeOpts.Paths) == 0 {
			return nil, fmt.Errorf("route match can't be empty")
		}

		if len(routeOpts.ServiceID) == 0 {
			return nil, fmt.Errorf("route service_id can't be empty")
		}

		service, found := services[routeOpts.ServiceID]
		if !found {
			return nil, fmt.Errorf("service_id '%s' was not found in route: %s", routeOpts.ServiceID, routeOpts.ID)
		}

		routeMiddlewares := make([]app.HandlerFunc, 0)

		for _, middleware := range routeOpts.Middlewares {
			if len(middleware.Use) > 0 {
				val, found := middlewares[middleware.Use]
				if !found {
					return nil, fmt.Errorf("middleware '%s' was not found in route id: '%s'", middleware.Use, routeOpts.ID)
				}

				routeMiddlewares = append(routeMiddlewares, val)
				continue
			}

			if len(middleware.Type) == 0 {
				return nil, fmt.Errorf("middleware kind can't be empty in route: '%s'", routeOpts.Paths)
			}

			handler, found := middlewareFactory[middleware.Type]
			if !found {
				return nil, fmt.Errorf("middleware handler '%s' was not found in route: '%s'", middleware.Type, routeOpts.Paths)
			}

			m, err := handler(middleware.Params)
			if err != nil {
				return nil, fmt.Errorf("create middleware handler '%s' failed in route: '%s'", middleware.Type, routeOpts.Paths)
			}

			routeMiddlewares = append(routeMiddlewares, m)
		}

		routeMiddlewares = append(routeMiddlewares, service.ServeHTTP)

		err := router.AddRoute(routeOpts, routeMiddlewares...)
		if err != nil {
			return nil, err
		}
	}

	return router, nil
}

// newRouter creates and returns a new router
func newRouter() *Router {
	r := &Router{
		tree:         newNode("/"),
		regexpRoutes: make([]routeSetting, 0),
	}

	return r
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	method := b2s(ctx.Method())
	path := b2s(ctx.Request.Path())

	middleware, isDefered := r.find(method, path)

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

// AddRoute adds a new route
func (r *Router) AddRoute(routeOpts config.RouteOptions, middlewares ...app.HandlerFunc) error {
	var err error

	// validate
	if len(routeOpts.Paths) == 0 {
		return errors.New("paths can't be empty")
	}

	for _, path := range routeOpts.Paths {
		path = strings.TrimSpace(path)
		var nodeType nodeType

		switch {
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

		case strings.HasPrefix(path, "^="):
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
				err = r.add(method, path, nodeType, middlewares...)
				if err != nil && !errors.Is(err, ErrAlreadyExists) {
					return err
				}
			}
		}

		for _, method := range routeOpts.Methods {
			method := strings.ToUpper(method)
			if !isValidHTTPMethod(method) {
				return fmt.Errorf("http method %s is not valid", method)
			}

			err = r.add(method, path, nodeType, middlewares...)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// add adds a route to radix tree
func (r *Router) add(method, path string, nodeType nodeType, middleware ...app.HandlerFunc) error {
	if len(path) == 0 || path[0] != '/' {
		return fmt.Errorf("router: '%s' is invalid path.  Path needs to begin with '/'", path)
	}

	originalPath := path
	currentNode := r.tree

	// If the path is the root path, add handler functions directly
	if path == "/" {
		if nodeType == nodeTypePrefix || nodeType == nodeTypeGeneral {
			childNode := currentNode.findChildByName("/", nodeType)
			if childNode == nil {
				childNode = newNode("/")
				currentNode.addChild(childNode, nodeType)
			}

			currentNode = childNode
		}

		currentNode.addHandler(method, middleware)
		return nil
	}

	// Remove leading slash
	if len(path) > 1 {
		path = path[1:]
	}

	// Split the path and traverse the nodes
	parts := strings.Split(path, "/")
	for idx, part := range parts {
		if len(part) == 0 {
			continue
		}

		isLast := false
		if idx == len(parts)-1 {
			isLast = true
		}

		// Find if the current node's children contain a node with the same name
		var childNode *node
		if isLast && (nodeType == nodeTypePrefix || nodeType == nodeTypeGeneral) {
			childNode = currentNode.findChildByName(part, nodeType)
		} else {
			childNode = currentNode.findChildByName(part, nodeTypeExact)
		}

		// If not found, create a new node
		if childNode == nil {
			if isLast && (nodeType == nodeTypePrefix || nodeType == nodeTypeGeneral) {
				childNode = newNode(part)
				currentNode.addChild(childNode, nodeType)
			} else {
				childNode = newNode(part)
				currentNode.addChild(childNode, nodeTypeExact)
			}
		}

		currentNode = childNode

	}

	handlers := currentNode.findHandler(method)
	if len(handlers) > 0 {
		return fmt.Errorf("router: duplicate route http_method:%s path:%s. %w", method, originalPath, ErrAlreadyExists)
	}

	// Add handler functions to the final node
	currentNode.addHandler(method, middleware)
	return nil
}

// find searches the Trie for handler functions matching the route, returns the handler functions and whether the handler is deferred (genernal match)
func (r *Router) find(method string, path string) ([]app.HandlerFunc, bool) {
	if path == "" || path[0] != '/' {
		path = "/"
	}

	currentNode := r.tree
	var prefixHandlers, generalHandlers []app.HandlerFunc

	// If the path is the root path, return the handler functions directly
	if path == "/" {
		h := currentNode.findHandler(method)
		if len(h) > 0 {
			return h, false
		}

		prefixChildNode := currentNode.matchChildByName("/", nodeTypePrefix)
		if prefixChildNode != nil {
			h := prefixChildNode.findHandler(method)
			if len(h) > 0 {
				return h, false
			}
		}

		generalChildNode := currentNode.matchChildByName("/", nodeTypeGeneral)
		if generalChildNode != nil {
			h := generalChildNode.findHandler(method)
			if len(h) > 0 {
				return h, true
			}
		}

		return nil, false
	}

	// Remove leading slash
	if len(path) > 1 {
		path = path[1:]
	}

	// Traverse the path segments
	for {
		// Find the next segment up to the next '/'
		slashIndex := strings.IndexByte(path, '/')
		var segment string
		if slashIndex == -1 {
			segment = path
			path = ""
		} else {
			segment = path[:slashIndex]
			path = path[slashIndex+1:]
		}

		// Skip empty segments (which can happen if there are consecutive slashes)
		if segment == "" {
			if path == "" {
				break
			}
			continue
		}

		// Find if the current node's children contain a node with the same name
		childNode := currentNode.matchChildByName(segment, nodeTypeExact)

		prefixChildNode := currentNode.matchChildByName(segment, nodeTypePrefix)
		if prefixChildNode != nil {
			h := prefixChildNode.findHandler(method)
			if len(h) > 0 {
				prefixHandlers = h
			}
		}

		generalChildNode := currentNode.matchChildByName(segment, nodeTypeGeneral)
		if generalChildNode != nil {
			h := generalChildNode.findHandler(method)
			if len(h) > 0 {
				generalHandlers = h
			}
		}

		if childNode == nil {
			currentNode = nil
			break
		}

		// Move to the child node
		currentNode = childNode

		// If there are no more segments, break
		if path == "" {
			break
		}
	}

	// Return handler functions of the final node
	if currentNode != nil {
		h := currentNode.findHandler(method)
		if len(h) > 0 {
			return h, false
		}
	}

	if len(prefixHandlers) > 0 {
		return prefixHandlers, false
	}

	if len(generalHandlers) > 0 {
		return generalHandlers, true
	}

	return nil, false
}

// newNode creates a new node
func newNode(path string) *node {
	return &node{
		path:            path,
		children:        make(map[string]*node),
		handler:         &methodHandler{handlers: make(map[string][]app.HandlerFunc)},
		prefixChildren:  make(map[string]*node),
		generalChildren: make(map[string]*node),
	}
}

// addChild adds a child node to the current node
func (n *node) addChild(child *node, nodeType nodeType) {
	switch nodeType {
	case nodeTypePrefix:
		n.prefixChildren[child.path] = child
	case nodeTypeExact:
		n.children[child.path] = child
	case nodeTypeGeneral:
		n.generalChildren[child.path] = child
	}
}

// findChildByName searches for a node with the specified name among the children
func (n *node) findChildByName(name string, nodeType nodeType) *node {
	switch nodeType {
	case nodeTypePrefix:
		if child, ok := n.prefixChildren[name]; ok {
			return child
		}
	case nodeTypeExact:
		if child, ok := n.children[name]; ok {
			return child
		}
	case nodeTypeGeneral:
		if child, ok := n.generalChildren[name]; ok {
			return child
		}
	}

	return nil
}

func (n *node) matchChildByName(name string, nodeType nodeType) *node {
	switch nodeType {
	case nodeTypeExact:
		if child, ok := n.children[name]; ok {
			return child
		}
	case nodeTypePrefix:
		for _, prefix := range n.prefixChildren {
			if strings.HasPrefix(name, prefix.path) {
				return prefix
			}
		}
	case nodeTypeGeneral:
		for _, general := range n.generalChildren {
			if strings.HasPrefix(name, general.path) {
				return general
			}
		}
	}

	return nil
}

// addHandler adds handler functions to the node
func (n *node) addHandler(method string, h []app.HandlerFunc) {
	if n.handler.handlers == nil {
		n.handler.handlers = make(map[string][]app.HandlerFunc)
	}
	n.handler.handlers[method] = h
}

// findHandler searches for handler functions based on the request method
func (n *node) findHandler(method string) []app.HandlerFunc {
	if handlers, ok := n.handler.handlers[method]; ok {
		return handlers
	}
	return nil
}

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
