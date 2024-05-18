package gateway

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unsafe"

	"github.com/cloudwego/hertz/pkg/app"
)

type Route struct {
	ID       string
	Match    string
	Method   []string
	Entry    []string
	Upstream string
}

// node represents a node in the Trie
type node struct {
	path     string           // Path name of the node
	children map[string]*node // Child nodes, indexed by path name
	handler  *methodHandler   // Handler functions
}

type routeSetting struct {
	regex      *regexp.Regexp
	prefixPath string
	route      *Route
	middleware []app.HandlerFunc
}

// methodHandler contains handler functions for various HTTP methods
type methodHandler struct {
	handlers map[string][]app.HandlerFunc // Associates HTTP methods with handler functions
}

// Router struct contains the Trie and handler chain
type Router struct {
	tree map[string]*node // Root node of the Trie, indexed by HTTP method

	prefixRoutes []routeSetting
	regexpRoutes []routeSetting
}

// NewRouter creates and returns a new router
func NewRouter() *Router {
	r := &Router{
		tree:         make(map[string]*node),
		prefixRoutes: make([]routeSetting, 0),
		regexpRoutes: make([]routeSetting, 0),
	}

	return r
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	method := B2s(ctx.Method())
	path := B2s(ctx.Request.Path())
	middleware := r.find(method, path)

	if len(middleware) > 0 {
		ctx.SetIndex(-1)
		ctx.SetHandlers(middleware)
		ctx.Next(c)
		ctx.Abort()
	}

	// prefix
	for _, route := range r.prefixRoutes {
		if checkPrefixRoute(route, method, path) {
			ctx.SetIndex(-1)
			ctx.SetHandlers(route.middleware)
			ctx.Next(c)
			ctx.Abort()
		}
	}

	// regexp
	for _, route := range r.regexpRoutes {
		if checkRegexpRoute(route, method, path) {
			ctx.SetIndex(-1)
			ctx.SetHandlers(route.middleware)
			ctx.Next(c)
			ctx.Abort()
		}
	}

}

func checkPrefixRoute(prefixSetting routeSetting, method, path string) bool {
	if len(prefixSetting.route.Method) > 0 {
		isMethodFound := false

		for _, m := range prefixSetting.route.Method {
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
	if len(prefixSetting.route.Method) > 0 {
		isMethodFound := false

		for _, m := range prefixSetting.route.Method {
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
func (r *Router) AddRoute(route Route, middleware ...app.HandlerFunc) error {
	var err error

	// check prefix
	if strings.HasSuffix(route.Match, "*") {
		prefixPath := strings.TrimSpace(route.Match[:len(route.Match)-1])

		prefixRoute := routeSetting{
			prefixPath: prefixPath,
			route:      &route,
			middleware: middleware,
		}

		r.prefixRoutes = append(r.prefixRoutes, prefixRoute)
		return nil
	}

	// regexp
	if route.Match[:1] == "~" {
		expr := strings.TrimSpace(route.Match[1:])
		regx, err := regexp.Compile(expr)
		if err != nil {
			return err
		}

		prefixRoute := routeSetting{
			regex:      regx,
			route:      &route,
			middleware: middleware,
		}

		r.regexpRoutes = append(r.regexpRoutes, prefixRoute)
		return nil
	}

	if len(route.Method) == 0 {
		err = r.add(GET, route.Match, middleware...)
		if err != nil {
			return err
		}
		err = r.add(POST, route.Match, middleware...)
		if err != nil {
			return err
		}
		err = r.add(PUT, route.Match, middleware...)
		if err != nil {
			return err
		}
		err = r.add(DELETE, route.Match, middleware...)
		if err != nil {
			return err
		}
		err = r.add(PATCH, route.Match, middleware...)
		if err != nil {
			return err
		}
		err = r.add(CONNECT, route.Match, middleware...)
		if err != nil {
			return err
		}
		err = r.add(HEAD, route.Match, middleware...)
		if err != nil {
			return err
		}
		err = r.add(TRACE, route.Match, middleware...)
		if err != nil {
			return err
		}
		err = r.add(OPTIONS, route.Match, middleware...)
		if err != nil {
			return err
		}
	}

	for _, method := range route.Method {
		if matches := upperLetterReg.MatchString(method); !matches {
			panic("http method " + method + " is not valid")
		}
		err = r.add(method, route.Match, middleware...)
		if err != nil {
			return err
		}
	}

	return nil
}

// add adds a static route
// TODO: support prefix route
func (r *Router) add(method string, path string, middleware ...app.HandlerFunc) error {
	if len(path) == 0 || path[0] != '/' {
		return errors.New("router: invalid path")
	}

	// Remove leading slash
	if len(path) > 1 {
		path = path[1:]
	}

	// Ensure the Trie root node for the HTTP method exists
	if _, ok := r.tree[method]; !ok {
		r.tree[method] = newNode("/")
	}

	currentNode := r.tree[method]

	// If the path is the root path, add handler functions directly
	if path == "" {
		currentNode.addHandler(method, middleware)
		return nil
	}

	// Split the path and traverse the nodes
	pathArray := strings.Split(path, "/")
	for _, element := range pathArray {
		if len(element) == 0 {
			continue
		}

		// Find if the current node's children contain a node with the same name
		childNode := currentNode.findChildByName(element)

		// If not found, create a new node
		if childNode == nil {
			childNode = newNode(element)
			currentNode.addChild(childNode)
		}

		currentNode = childNode
	}

	// Add handler functions to the final node
	currentNode.addHandler(method, middleware)
	return nil
}

// find searches the Trie for handler functions matching the route
// check order:
// 1. static routes
// 2. prefix routes
func (r *Router) find(method string, path string) []app.HandlerFunc {
	// Ensure the path is valid and sanitized
	path = sanitizeUrl(path)

	// Check if the Trie root node for the HTTP method exists
	rootNode, ok := r.tree[method]
	if !ok {
		return nil
	}

	currentNode := rootNode

	// If the path is the root path, return the handler functions directly
	if path == "/" {
		return currentNode.findHandler(method)
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

// Regular expression to match valid URL paths
var validPathRegex = regexp.MustCompile(`^/[^/]+(/[^/]+)*$`)

// sanitizeUrl cleans the path
func sanitizeUrl(path string) string {
	if validPathRegex.MatchString(path) {
		return path
	}

	// If the path is not valid, return the default path
	return "/"
}

const (
	// CONNECT HTTP method
	CONNECT = "CONNECT"
	// DELETE HTTP method
	DELETE = "DELETE"
	// GET HTTP method
	GET = "GET"
	// HEAD HTTP method
	HEAD = "HEAD"
	// OPTIONS HTTP method
	OPTIONS = "OPTIONS"
	// PATCH HTTP method
	PATCH = "PATCH"
	// POST HTTP method
	POST = "POST"
	// PUT HTTP method
	PUT = "PUT"
	// TRACE HTTP method
	TRACE = "TRACE"
)

func B2s(b []byte) string {
	/* #nosec G103 */
	return *(*string)(unsafe.Pointer(&b))
}
