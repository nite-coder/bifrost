package gateway

import (
	"fmt"
	"strings"

	"github.com/cloudwego/hertz/pkg/app"
)

// methodHandler contains handler functions for various HTTP methods
type methodHandler struct {
	handlers map[string][]app.HandlerFunc // Associates HTTP methods with handler functions
}

type nodeType int32

const (
	nodeTypeExact nodeType = iota
	nodeTypePrefix
	nodeTypeGeneral
	nodeTypeRegex
)

// node represents a node in the Trie
type node struct {
	path            string           // Path name of the node
	children        map[string]*node // Child nodes, indexed by path name
	handler         *methodHandler   // Handler functions
	prefixChildren  map[string]*node
	generalChildren map[string]*node
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
			if prefix.path == "/" || strings.HasPrefix(name, prefix.path) {
				return prefix
			}
		}
	case nodeTypeGeneral:
		for _, general := range n.generalChildren {
			if general.path == "/" || strings.HasPrefix(name, general.path) {
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

// Router struct contains the Trie and handler chain
type Router struct {
	tree *node // Root node of the Trie
}

// newRouter creates and returns a new router
func newRouter() *Router {
	r := &Router{
		tree: newNode("/"),
	}

	return r
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
