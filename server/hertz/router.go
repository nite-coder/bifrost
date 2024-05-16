package main

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

type kind uint8

// node represents a node in the Trie
type node struct {
	parent   *node          // Parent node
	children []*node        // Child nodes
	kind     kind           // Node type
	name     string         // Node name
	handler  *methodHandler // Handler functions
}

// methodHandler contains handler functions for various HTTP methods
type methodHandler struct {
	connect []app.HandlerFunc
	delete  []app.HandlerFunc
	get     []app.HandlerFunc
	head    []app.HandlerFunc
	options []app.HandlerFunc
	patch   []app.HandlerFunc
	post    []app.HandlerFunc
	put     []app.HandlerFunc
	trace   []app.HandlerFunc
}

// Router structure contains Trie and handlers chain
type Router struct {
	tree     *node             // Root node of the Trie
	handlers app.HandlersChain // Handlers chain
}

// NewRouter creates and returns a new router
func NewRouter() *Router {
	r := &Router{
		tree: &node{
			parent:   nil,
			children: []*node{},
			kind:     0,
			name:     "/",
			handler:  &methodHandler{},
		},
		handlers: make([]app.HandlerFunc, 0),
	}

	// Find static routes and execute middleware
	r.use(func(c context.Context, ctx *app.RequestContext) {
		method := B2s(ctx.Method())
		path := B2s(ctx.Request.Path())
		middleware := r.find(method, path)

		if len(middleware) > 0 {
			ctx.SetIndex(-1)
			ctx.SetHandlers(middleware)
			ctx.Next(c)
			ctx.Abort()
		}
	})

	return r
}

// ServeHTTP implements the http.Handler interface
func (r *Router) ServeHTTP(c context.Context, ctx *app.RequestContext) {
	ctx.SetIndex(-1)
	ctx.SetHandlers(r.handlers)
	ctx.Next(c)
}

// Regexp adds middleware for routes matching a regular expression
func (r *Router) Regexp(expr string, middleware ...app.HandlerFunc) {
	regx, err := regexp.Compile(expr)
	if err != nil {
		panic(err)
	}

	r.use(func(c context.Context, ctx *app.RequestContext) {
		if !regx.MatchString(string(ctx.Request.Path())) {
			return
		}

		ctx.SetIndex(-1)
		ctx.SetHandlers(middleware)
		ctx.Next(c)
		ctx.Abort()
	})
}

// AddRoute adds a static route
func (r *Router) AddRoute(route Route, middleware ...app.HandlerFunc) {
	err := r.addStaticRoute("POST", route.Match, middleware...)
	if err != nil {
		panic(err)
	}
}

// use adds global middleware
func (r *Router) use(middleware ...app.HandlerFunc) {
	r.handlers = append(r.handlers, middleware...)
}

// addStaticRoute adds a static route
func (r *Router) addStaticRoute(method string, path string, middleware ...app.HandlerFunc) error {
	if len(path) == 0 || path[0] != '/' {
		return errors.New("router: invalid path")
	}

	// Remove leading slash
	if len(path) > 1 {
		path = path[1:]
	}

	currentNode := r.tree

	// If path is root, directly add handler
	if path == "" {
		currentNode.addHandler(method, middleware)
		return nil
	}

	// Split path and traverse nodes
	pathArray := strings.Split(path, "/")
	for _, element := range pathArray {
		if len(element) == 0 {
			continue
		}

		// Find if current node's children have a node with the same name
		childNode := currentNode.findChildByName(element)

		// If not found, create a new node
		if childNode == nil {
			childNode = newNode(element, skind)
			currentNode.addChild(childNode)
		}

		currentNode = childNode
	}

	// Add handler to the final node
	currentNode.addHandler(method, middleware)
	return nil
}

// find searches for matching handler functions in the Trie
func (r *Router) find(method string, path string) []app.HandlerFunc {
	path = sanitizeUrl(path)

	currentNode := r.tree
	if path == "/" {
		return currentNode.findHandler(method)
	}

	pathArray := strings.Split(path, "/")
	for _, element := range pathArray {
		if len(element) == 0 {
			continue
		}

		childNode := currentNode.findChildByName(element)

		// If no matching node found, return nil
		if childNode == nil {
			return nil
		}

		currentNode = childNode
	}

	// Return handler functions of the final node
	return currentNode.findHandler(method)
}

// newNode creates a new node
func newNode(name string, t kind) *node {
	return &node{
		kind:    t,
		name:    name,
		handler: &methodHandler{},
	}
}

// addChild adds a child node to the current node
func (n *node) addChild(node *node) {
	node.parent = n
	n.children = append(n.children, node)
}

// findChildByName searches for a node with the specified name in the children of the current node
func (n *node) findChildByName(name string) *node {
	for _, element := range n.children {
		if element.name == name && element.kind == skind {
			return element
		}
	}
	return nil
}

// addHandler adds handler functions to the node
func (n *node) addHandler(method string, h []app.HandlerFunc) {
	switch method {
	case GET:
		n.handler.get = h
	case POST:
		n.handler.post = h
	case PUT:
		n.handler.put = h
	case DELETE:
		n.handler.delete = h
	case PATCH:
		n.handler.patch = h
	case OPTIONS:
		n.handler.options = h
	case HEAD:
		n.handler.head = h
	case CONNECT:
		n.handler.connect = h
	case TRACE:
		n.handler.trace = h
	default:
		panic("router: invalid method")
	}
}

// findHandler finds handler functions based on request method
func (n *node) findHandler(method string) []app.HandlerFunc {
	switch method {
	case GET:
		return n.handler.get
	case POST:
		return n.handler.post
	case PUT:
		return n.handler.put
	case DELETE:
		return n.handler.delete
	case PATCH:
		return n.handler.patch
	case OPTIONS:
		return n.handler.options
	case HEAD:
		return n.handler.head
	case CONNECT:
		return n.handler.connect
	case TRACE:
		return n.handler.trace
	default:
		panic("router: invalid method")
	}
}

// sanitizeUrl cleans the path
func sanitizeUrl(path string) string {
	if len(path) > 1 && path[0] == '/' && path[1] != '/' && path[1] != '\\' {
		return path
	}
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

const (
	// skind represents a static node
	skind kind = iota
)
