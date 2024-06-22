package gateway

import (
	"context"
	"fmt"
	"http-benchmark/pkg/config"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/cloudwego/hertz/pkg/app"
)

// node represents a node in the Trie
type node struct {
	path               string           // Path name of the node
	children           map[string]*node // Child nodes, indexed by path name
	parameterizedChild *node
	handler            *methodHandler // Handler functions
}

type routeSetting struct {
	regex      *regexp.Regexp
	prefixPath string
	route      *config.RouteOptions
	middleware []app.HandlerFunc
}

// methodHandler contains handler functions for various HTTP methods
type methodHandler struct {
	handlers map[string][]app.HandlerFunc // Associates HTTP methods with handler functions
}

// Router struct contains the Trie and handler chain
type Router struct {
	isHostEnabled bool
	tree          *node // Root node of the Trie

	prefixRoutes []routeSetting
	regexpRoutes []routeSetting
}

// newRouter creates and returns a new router
func newRouter(isHostEnabled bool) *Router {
	r := &Router{
		isHostEnabled: isHostEnabled,
		tree:          newNode("/"),
		prefixRoutes:  make([]routeSetting, 0),
		regexpRoutes:  make([]routeSetting, 0),
	}

	return r
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	method := b2s(ctx.Method())
	path := b2s(ctx.Request.Path())
	middleware := r.find(method, path)

	if len(middleware) > 0 {
		ctx.SetIndex(-1)
		ctx.SetHandlers(middleware)
		ctx.Next(c)
		ctx.Abort()
		return
	}

	// regexp
	for _, route := range r.regexpRoutes {
		if len(route.route.Hosts) > 0 {
			host := getHost(ctx)
			if !slices.Contains(route.route.Hosts, host()) {
				continue
			}
		}

		if checkRegexpRoute(route, method, path) {
			ctx.SetIndex(-1)
			ctx.SetHandlers(route.middleware)
			ctx.Next(c)
			ctx.Abort()
			return
		}
	}

	// prefix
	for _, route := range r.prefixRoutes {
		if len(route.route.Hosts) > 0 {
			host := getHost(ctx)
			if !slices.Contains(route.route.Hosts, host()) {
				continue
			}
		}

		if checkPrefixRoute(route, method, path) {
			ctx.SetIndex(-1)
			ctx.SetHandlers(route.middleware)
			ctx.Next(c)
			ctx.Abort()
			return
		}
	}

}

func checkPrefixRoute(prefixSetting routeSetting, method, path string) bool {
	if len(prefixSetting.route.Methods) > 0 {
		isMethodFound := false

		for _, m := range prefixSetting.route.Methods {
			if m == method {
				return true
			}
		}

		if !isMethodFound {
			return false
		}
	}

	return strings.HasPrefix(path, prefixSetting.prefixPath)
}

func checkRegexpRoute(prefixSetting routeSetting, method, path string) bool {
	if len(prefixSetting.route.Methods) > 0 {
		isMethodFound := false

		for _, m := range prefixSetting.route.Methods {
			if m == method {
				return true
			}
		}

		if !isMethodFound {
			return false
		}
	}

	return prefixSetting.regex.MatchString(path)
}

var upperLetterReg = regexp.MustCompile("^[A-Z]+$")

// AddRoute adds a static route
func (r *Router) AddRoute(routeOpts config.RouteOptions, middlewares ...app.HandlerFunc) error {
	var err error

	for _, path := range routeOpts.Paths {
		first := path[:1]
		// check prefix match
		if first == "/" && strings.HasSuffix(path, "*") {
			prefixPath := strings.TrimSpace(path[:len(path)-1])

			prefixRoute := routeSetting{
				prefixPath: prefixPath,
				route:      &routeOpts,
				middleware: middlewares,
			}

			r.prefixRoutes = append(r.prefixRoutes, prefixRoute)

			sort.SliceStable(r.prefixRoutes, func(i, j int) bool {
				return len(r.prefixRoutes[i].prefixPath) > len(r.prefixRoutes[j].prefixPath)
			})

			return nil
		}

		// regexp match
		if first == "~" {
			expr := strings.TrimSpace(path[1:])
			regx, err := regexp.Compile(expr)
			if err != nil {
				return err
			}

			prefixRoute := routeSetting{
				regex:      regx,
				route:      &routeOpts,
				middleware: middlewares,
			}

			r.regexpRoutes = append(r.regexpRoutes, prefixRoute)
			return nil
		}

		// static route
		if first != "/" {
			return fmt.Errorf("router: '%s' is invalid path.  Path needs to begin with '/'", path)
		}
		hosts := routeOpts.Hosts
		if len(hosts) == 0 {
			hosts = []string{"/:host"}
		}

		for _, host := range hosts {

			if r.isHostEnabled {
				path = fmt.Sprintf("/%s%s", host, path)
			}

			if len(routeOpts.Methods) == 0 {
				err = r.add(GET, path, middlewares...)
				if err != nil {
					return err
				}
				err = r.add(POST, path, middlewares...)
				if err != nil {
					return err
				}
				err = r.add(PUT, path, middlewares...)
				if err != nil {
					return err
				}
				err = r.add(DELETE, path, middlewares...)
				if err != nil {
					return err
				}
				err = r.add(PATCH, path, middlewares...)
				if err != nil {
					return err
				}
				err = r.add(CONNECT, path, middlewares...)
				if err != nil {
					return err
				}
				err = r.add(HEAD, path, middlewares...)
				if err != nil {
					return err
				}
				err = r.add(TRACE, path, middlewares...)
				if err != nil {
					return err
				}
				err = r.add(OPTIONS, path, middlewares...)
				if err != nil {
					return err
				}
			}

			for _, method := range routeOpts.Methods {
				if matches := upperLetterReg.MatchString(method); !matches {
					panic("http method " + method + " is not valid")
				}
				err = r.add(method, path, middlewares...)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// add adds a static route
func (r *Router) add(method string, path string, middleware ...app.HandlerFunc) error {
	if len(path) == 0 || path[0] != '/' {
		return fmt.Errorf("router: '%s' is invalid path.  Path needs to begin with '/'", path)
	}

	// Remove leading slash
	if len(path) > 1 {
		path = path[1:]
	}

	currentNode := r.tree

	// If the path is the root path, add handler functions directly
	if path == "" {
		currentNode.addHandler(method, middleware)
		return nil
	}

	// Split the path and traverse the nodes
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}

		isParameter := len(part) > 0 && part[0] == ':'
		if len(part) == 0 {
			continue
		}

		if isParameter {
			var childNode *node
			if currentNode.parameterizedChild != nil {
				childNode = currentNode.parameterizedChild
			} else {
				childNode = newNode(part)
				currentNode.parameterizedChild = childNode
			}
			currentNode = childNode
		} else {
			// Find if the current node's children contain a node with the same name
			childNode := currentNode.findChildByName(part)

			// If not found, create a new node
			if childNode == nil {
				childNode = newNode(part)
				currentNode.addChild(childNode)
			}

			currentNode = childNode
		}

	}

	// Add handler functions to the final node
	currentNode.addHandler(method, middleware)
	return nil
}

// find searches the Trie for handler functions matching the route
func (r *Router) find(method string, path string) []app.HandlerFunc {
	if path == "" || path[0] != '/' {
		path = "/"
	}

	currentNode := r.tree

	// If the path is the root path, return the handler functions directly
	if path == "/" {
		return currentNode.findHandler(method)
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
		childNode := currentNode.findChildByName(segment)

		// If no matching node is found, return nil
		if childNode == nil && currentNode.parameterizedChild != nil {
			childNode = currentNode.parameterizedChild
		}

		if childNode == nil {
			return nil
		}

		// Move to the child node
		currentNode = childNode

		// If there are no more segments, break
		if path == "" {
			break
		}
	}

	// Return handler functions of the final node
	return currentNode.findHandler(method)
}

// newNode creates a new node
func newNode(path string) *node {
	return &node{
		path:     path,
		children: make(map[string]*node),
		handler:  &methodHandler{handlers: make(map[string][]app.HandlerFunc)},
	}
}

// addChild adds a child node to the current node
func (n *node) addChild(child *node) {
	n.children[child.path] = child
}

// findChildByName searches for a node with the specified name among the children
func (n *node) findChildByName(name string) *node {
	if child, ok := n.children[name]; ok {
		return child
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
