package router

import (
	"fmt"
	"github.com/cloudwego/hertz/pkg/app"
	"net/http"
	"sort"
	"strings"
)

var (
	HTTPMethods = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions, http.MethodTrace, http.MethodConnect}
)

// methodHandler contains handler functions for various HTTP methods
type methodHandler struct {
	handlers map[string][]app.HandlerFunc // Associates HTTP methods with handler functions
}
type NodeType int32

const (
	Exact NodeType = iota
	PreferentialPrefix
	Prefix
	Regex
)

type Children struct {
	Node *node
	Path string
}

// node represents a node in the Trie
type node struct {
	path            string           // Path name of the node
	children        map[string]*node // Child nodes, indexed by path name
	handler         *methodHandler   // Handler functions
	prefixChildren  []*Children
	generalChildren []*Children
}

// newNode creates a new node
func newNode(path string) *node {
	return &node{
		path:            path,
		children:        make(map[string]*node),
		handler:         &methodHandler{handlers: make(map[string][]app.HandlerFunc)},
		prefixChildren:  make([]*Children, 0),
		generalChildren: make([]*Children, 0),
	}
}

// addChild adds a child node to the current node
func (n *node) addChild(child *node, nodeType NodeType) {
	switch nodeType {
	case PreferentialPrefix:
		c := &Children{
			Path: child.path,
			Node: child,
		}
		for _, cc := range n.prefixChildren {
			if cc.Path == child.path {
				cc.Node = child
				return
			}
		}
		n.prefixChildren = append(n.prefixChildren, c)
		sort.Slice(n.prefixChildren, func(i, j int) bool {
			return len(n.prefixChildren[i].Path) > len(n.prefixChildren[j].Path)
		})
	case Exact:
		n.children[child.path] = child
	case Prefix:
		c := &Children{
			Path: child.path,
			Node: child,
		}
		for _, cc := range n.generalChildren {
			if cc.Path == child.path {
				cc.Node = child
				return
			}
		}
		n.generalChildren = append(n.generalChildren, c)
		sort.Slice(n.generalChildren, func(i, j int) bool {
			return len(n.generalChildren[i].Path) > len(n.generalChildren[j].Path)
		})
	default:
	}
}

// findChildByName searches for a node with the specified name among the children
func (n *node) findChildByName(name string, nodeType NodeType) *node {
	switch nodeType {
	case PreferentialPrefix:
		for _, cc := range n.prefixChildren {
			if cc.Path == name {
				return cc.Node
			}
		}
	case Exact:
		if child, ok := n.children[name]; ok {
			return child
		}
	case Prefix:
		for _, cc := range n.generalChildren {
			if cc.Path == name {
				return cc.Node
			}
		}
	default:
		return nil
	}
	return nil
}
func (n *node) matchChildByName(name string, nodeType NodeType) *node {
	switch nodeType {
	case Exact:
		if child, ok := n.children[name]; ok {
			return child
		}
	case PreferentialPrefix:
		for _, cc := range n.prefixChildren {
			if cc.Path == "/" || strings.HasPrefix(name, cc.Path) {
				return cc.Node
			}
		}
	case Prefix:
		for _, cc := range n.generalChildren {
			if cc.Path == "/" || strings.HasPrefix(name, cc.Path) {
				return cc.Node
			}
		}
	default:
		return nil
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

// Router struct contains the Trie and handler chain
type Router struct {
	tree *node // Root node of the Trie
}

// NewRouter creates and returns a new router
func NewRouter() *Router {
	r := &Router{
		tree: newNode("/"),
	}
	return r
}

// Add adds a route to radix tree
func (r *Router) Add(method, path string, nodeType NodeType, middleware ...app.HandlerFunc) error {
	if len(path) == 0 || path[0] != '/' {
		return fmt.Errorf("router: invalid path '%s'; must begin with '/'", path)
	}
	originalPath := path
	currentNode := r.tree
	// If the path is the root path, add handler functions directly
	if path == "/" {
		if nodeType == PreferentialPrefix || nodeType == Prefix {
			childNode := currentNode.findChildByName("/", nodeType)
			if childNode == nil {
				childNode = newNode("/")
				currentNode.addChild(childNode, nodeType)
			}
			currentNode = childNode
		}
		handlers := currentNode.findHandler(method)
		if len(handlers) > 0 {
			return fmt.Errorf("router: duplicate route for method '%s' and path '%s': %w", method, originalPath, ErrAlreadyExists)
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
		isLast := false
		if idx == len(parts)-1 {
			isLast = true
		}
		// Find if the current node's children contain a node with the same name
		var childNode *node
		if isLast && (nodeType == PreferentialPrefix || nodeType == Prefix) {
			childNode = currentNode.findChildByName(part, nodeType)
		} else {
			childNode = currentNode.findChildByName(part, Exact)
		}
		// If not found, create a new node
		if childNode == nil {
			if isLast && (nodeType == PreferentialPrefix || nodeType == Prefix) {
				childNode = newNode(part)
				currentNode.addChild(childNode, nodeType)
			} else {
				childNode = newNode(part)
				currentNode.addChild(childNode, Exact)
			}
		}
		currentNode = childNode
	}
	handlers := currentNode.findHandler(method)
	if len(handlers) > 0 {
		return fmt.Errorf("router: duplicate route for method '%s' and path '%s': %w", method, originalPath, ErrAlreadyExists)
	}
	// Add handler functions to the final node
	currentNode.addHandler(method, middleware)
	return nil
}

// Find searches the Trie for handler functions matching the route, returns the handler functions and whether the handler is deferred (genernal match)
func (r *Router) Find(method string, path string) ([]app.HandlerFunc, bool) {
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
		prefixChildNode := currentNode.matchChildByName("/", PreferentialPrefix)
		if prefixChildNode != nil {
			h := prefixChildNode.findHandler(method)
			if len(h) > 0 {
				return h, false
			}
		}
		generalChildNode := currentNode.matchChildByName("/", Prefix)
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
		childNode := currentNode.matchChildByName(segment, Exact)
		prefixChildNode := currentNode.matchChildByName(segment, PreferentialPrefix)
		if prefixChildNode != nil {
			h := prefixChildNode.findHandler(method)
			if len(h) > 0 {
				prefixHandlers = h
			}
		}
		generalChildNode := currentNode.matchChildByName(segment, Prefix)
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
func IsValidHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch, http.MethodHead, http.MethodOptions, http.MethodTrace, http.MethodConnect:
		return true
	default:
		return false
	}
}
